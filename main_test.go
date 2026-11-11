package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestVersionDefault(t *testing.T) {
	if Version == "" {
		t.Error("expected non-empty default version")
	}
}

func TestBuildVersionOutputAddsVPrefixAndMetadata(t *testing.T) {
	got := buildVersionOutput("1.2.3")
	want := fmt.Sprintf("v1.2.3 (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if got != want {
		t.Fatalf("unexpected version output: got %q, want %q", got, want)
	}
}

func TestBuildVersionOutputPreservesExistingVPrefix(t *testing.T) {
	got := buildVersionOutput("v1.2.3")
	want := fmt.Sprintf("v1.2.3 (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if got != want {
		t.Fatalf("unexpected version output: got %q, want %q", got, want)
	}
}

func TestBuildVersionOutputNoVPrefixForDev(t *testing.T) {
	got := buildVersionOutput("dev")
	want := fmt.Sprintf("dev (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if got != want {
		t.Fatalf("unexpected version output: got %q, want %q", got, want)
	}
}

func TestLookupProviderDocsJSONArguments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docs.md")
	if err := os.WriteFile(path, []byte("provider documentation"), 0600); err != nil {
		t.Fatal(err)
	}
	previous := providerIndex
	providerIndex = map[string][]string{"example_resource": {path}}
	t.Cleanup(func() { providerIndex = previous })

	tests := []struct {
		name      string
		arguments []byte
		wantError bool
		wantText  string
	}{
		{"valid", []byte(`{"provider_name":"example_resource"}`), false, "provider documentation"},
		{"duplicate names", []byte(`{"provider_name":"example_resource","provider_name":"other"}`), true, "invalid arguments"},
		{"invalid UTF-8", []byte("{\"provider_name\":\"example_\xff\"}"), true, "invalid arguments"},
		{"malformed", []byte(`{"provider_name":`), true, "invalid arguments"},
		{"missing name", []byte(`{}`), true, "invalid or missing 'provider_name' parameter"},
		{"wrong type", []byte(`{"provider_name":42}`), true, "invalid or missing 'provider_name' parameter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := lookupProviderDocs(context.Background(), &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: tt.arguments}})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError != tt.wantError {
				t.Fatalf("IsError = %v, want %v", result.IsError, tt.wantError)
			}
			if len(result.Content) != 1 {
				t.Fatalf("got %d content entries, want 1", len(result.Content))
			}
			content, ok := result.Content[0].(*mcp.TextContent)
			if !ok || !strings.Contains(content.Text, tt.wantText) {
				t.Fatalf("content = %#v, want text containing %q", result.Content[0], tt.wantText)
			}
		})
	}
}
