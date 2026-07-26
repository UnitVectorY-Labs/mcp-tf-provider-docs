package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/adrg/frontmatter"
)

// Config holds settings loaded from the YAML file.
type Config struct {
	DocsPath        string `yaml:"docs_path"`
	MatchPattern    string `yaml:"match_pattern"`
	ToolDescription string `yaml:"tool_description"`
	ToolName        string `yaml:"tool_name"`
}

// providerIndex maps a resource name to documentation file paths.
var providerIndex = make(map[string][]string)

var Version = "dev" // This will be set by the build systems to the release version

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+`)

func buildVersionOutput(version string) string {
	normalized := version
	if semverRe.MatchString(normalized) && !strings.HasPrefix(normalized, "v") {
		normalized = "v" + normalized
	}
	return fmt.Sprintf("%s (%s, %s/%s)", normalized, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

type LookupProviderDocsInput struct {
	ProviderName string `json:"provider_name" jsonschema:"Fully qualified Terraform/Tofu resource or data source name (e.g., google_compute_instance)."`
}

type LookupProviderDocsOutput struct {
	Content string `json:"content" jsonschema:"the documentation content"`
}

func main() {
	// Set the build version from the build info if not set by the build system
	if Version == "dev" || Version == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				Version = bi.Main.Version
			}
		}
	}

	// Handle --version flag
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("mcp-tf-provider-docs version %s\n", buildVersionOutput(Version))
		os.Exit(0)
	}

	// Load config from YAML
	configPath := os.Getenv("TF_CONFIG")
	if configPath == "" {
		log.Fatalf("Environment variable TF_CONFIG is required")
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set default tool name if not provided
	toolName := cfg.ToolName
	if toolName == "" {
		toolName = "lookup_provider_docs"
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
	srv := mcp.NewServer(&mcp.Implementation{Name: "mcp-tf-provider-docs", Version: Version}, nil)

	// Register the lookup tool with name and description from config
	mcp.AddTool(srv, &mcp.Tool{
		Name:        toolName,
		Description: cfg.ToolDescription,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input LookupProviderDocsInput) (
		*mcp.CallToolResult, LookupProviderDocsOutput, error,
	) {
		pname := input.ProviderName

		paths, found := providerIndex[pname]
		if !found || len(paths) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("no docs found for '%s'", pname)}},
				IsError: true,
			}, LookupProviderDocsOutput{}, nil
		}

		var builder strings.Builder
		for _, p := range paths {
			contentBytes, err := os.ReadFile(p)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("error reading '%s': %v", p, err)}},
					IsError: true,
				}, LookupProviderDocsOutput{}, nil
			}

			// The Markdown files may contain front matter, since this is not valuable to the MCP tool, we strip it out
			content, err := StripFrontMatterWithLib(string(contentBytes))
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("error stripping front matter from '%s': %v", p, err)}},
					IsError: true,
				}, LookupProviderDocsOutput{}, nil
			}

			builder.WriteString(content)
			builder.WriteString("\n\n---\n\n")
		}
		return nil, LookupProviderDocsOutput{Content: builder.String()}, nil
	})

	if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
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
