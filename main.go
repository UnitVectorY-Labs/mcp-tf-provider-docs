package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/adrg/frontmatter"
)

// Config holds settings loaded from the YAML file.
type Config struct {
	DocsPath        string `yaml:"docs_path"`
	MatchPattern    string `yaml:"match_pattern"`
	ToolDescription string `yaml:"tool_description"`
}

// providerIndex maps a resource name to documentation file paths.
var providerIndex = make(map[string][]string)

func main() {

	// Load config from YAML
	configPath := os.Getenv("TF_CONFIG")
	if configPath == "" {
		log.Fatalf("Environment variable TF_CONFIG is required")
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Build the in-memory index
	if err := buildIndex(cfg); err != nil {
		log.Fatalf("Failed to build index: %v", err)
	}

	// Log the number of providers found
	log.Printf("Found %d unique providers in documentation", len(providerIndex))
	// If no providers were found, exit early
	if len(providerIndex) == 0 {
		log.Println("No providers found in documentation, exiting.")
		return
	}

	// Log all of the patterns to the console for debugging
	for pname, paths := range providerIndex {
		log.Printf("Found provider '%s' in %d files", pname, len(paths))
		for _, p := range paths {
			log.Printf("  - %s", p)
		}
	}

	// Create the MCP server
	srv := server.NewMCPServer("mcp-tf-provider-docs", "0.1.0")

	// Register the lookup tool with description from config
	tool := mcp.NewTool(
		"lookupProviderDocs",
		mcp.WithDescription(cfg.ToolDescription),
		mcp.WithString(
			"provider_name",
			mcp.Description("Fully qualified Terraform/Tofu resource or data source name (e.g., google_compute_instance)."),
			mcp.Required(),
		),
	)
	srv.AddTool(tool, handleLookup)

	if err := server.ServeStdio(srv); err != nil {
		log.Fatalf("MCP server terminated: %v", err)
	}
}

// loadConfig reads and parses the YAML configuration file.
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// buildIndex walks the docs directory, finds matching files, and builds an index.
func buildIndex(cfg *Config) error {
	docsRoot := cfg.DocsPath
	filePattern := regexp.MustCompile(`(?i)\.md$|\.markdown$`)
	matchPattern, err := compileRegex(cfg.MatchPattern)
	if err != nil {
		return fmt.Errorf("invalid match_pattern regex: %w", err)
	}

	return filepath.Walk(docsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if !filePattern.MatchString(info.Name()) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)

		matches := matchPattern.FindAllString(content, -1)
		if len(matches) == 0 {
			return nil
		}

		seen := make(map[string]struct{})
		for _, m := range matches {
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			providerIndex[m] = append(providerIndex[m], path)
		}
		return nil
	})

}

// compileRegex compiles a string into a regexp.Regexp, returning an error if invalid.
func compileRegex(expr string) (*regexp.Regexp, error) {
	return regexp.Compile(expr)
}

// handleLookup is the MCP tool handler that returns docs for a given provider name.
func handleLookup(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	args := req.GetArguments()
	pname, ok := args["provider_name"].(string)
	if !ok {
		return mcp.NewToolResultError("invalid or missing 'provider_name' parameter"), nil
	}

	paths, found := providerIndex[pname]
	if !found || len(paths) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("no docs found for '%s'", pname)), nil
	}

	var builder strings.Builder
	for _, p := range paths {
		contentBytes, err := os.ReadFile(p)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error reading '%s': %v", p, err)), nil
		}

		// The Markdown files may contan front matter, since this is not valuable to the MCP tool, we strip it out
		content, err := StripFrontMatterWithLib(string(contentBytes))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error stripping front matter from '%s': %v", p, err)), nil
		}

		builder.WriteString(content)
		builder.WriteString("\n\n---\n\n")
	}
	return mcp.NewToolResultText(builder.String()), nil
}

func StripFrontMatterWithLib(content string) (string, error) {
	// Use an empty struct since we don't need to capture metadata
	var meta struct{}
	rest, err := frontmatter.Parse(strings.NewReader(content), &meta)
	if err != nil {
		// No front matter? frontmatter.ErrNotFound is returned ➝ just return original
		if err == frontmatter.ErrNotFound {
			return content, nil
		}
		return "", err
	}
	return string(rest), nil
}
