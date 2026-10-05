package godot

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error: %v", path, err)
	}
}

func TestFindProjectName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "project.godot"), "config_version=5\n\n[application]\n\nconfig/name=\"Windup Wonderland\"\nrun/main_scene=\"res://main.tscn\"\n")
	sub := filepath.Join(root, "src", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	name, ok := FindProjectName(sub)
	if !ok || name != "Windup Wonderland" {
		t.Fatalf("FindProjectName(%q) = %q, %v; want %q, true", sub, name, ok, "Windup Wonderland")
	}
}

func TestFindProjectNameNotFound(t *testing.T) {
	root := t.TempDir()
	if name, ok := FindProjectName(root); ok || name != "" {
		t.Fatalf("FindProjectName(%q) = %q, %v; want empty, false", root, name, ok)
	}
}

func TestParseProjectNameWrongSection(t *testing.T) {
	content := "config/name=\"Nope\"\n\n[application]\nconfig/name=\"Yep\"\n"
	if got := parseProjectName(content); got != "Yep" {
		t.Fatalf("parseProjectName = %q, want %q", got, "Yep")
	}
}

func TestParseProjectNameMissing(t *testing.T) {
	content := "config_version=5\n\n[application]\nrun/main_scene=\"res://main.tscn\"\n"
	if got := parseProjectName(content); got != "" {
		t.Fatalf("parseProjectName = %q, want empty", got)
	}
}
