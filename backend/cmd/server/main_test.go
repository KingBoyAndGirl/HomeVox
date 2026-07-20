package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFrontendDirRequiresIndex(t *testing.T) {
	frontendDir := t.TempDir()
	if err := validateFrontendDir(frontendDir); err == nil {
		t.Fatal("validateFrontendDir succeeded without index.html")
	}

	if err := os.WriteFile(filepath.Join(frontendDir, "index.html"), []byte("homevox"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := validateFrontendDir(frontendDir); err != nil {
		t.Fatalf("validateFrontendDir with index.html: %v", err)
	}
}
