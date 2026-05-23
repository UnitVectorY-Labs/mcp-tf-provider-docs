package main

import (
	"fmt"
	"runtime"
	"testing"
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
