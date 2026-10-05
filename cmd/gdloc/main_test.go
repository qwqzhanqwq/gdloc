package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func samplePath() string {
	return filepath.Join("..", "..", "testdata", "sample")
}

func TestFlagOrderIndependent(t *testing.T) {
	var a, b bytes.Buffer
	if code := run([]string{"--by-file", samplePath()}, &a, &b); code != 0 {
		t.Fatalf("run returned %d, stderr=%s", code, b.String())
	}
	var c, d bytes.Buffer
	if code := run([]string{samplePath(), "--by-file"}, &c, &d); code != 0 {
		t.Fatalf("run returned %d, stderr=%s", code, d.String())
	}
	if a.String() != c.String() {
		t.Errorf("flag order changed output:\nbefore path:\n%s\nafter path:\n%s", a.String(), c.String())
	}
}

func TestVersionAnywhere(t *testing.T) {
	for _, argv := range [][]string{{"--version"}, {"dummy", "--version"}} {
		var out, errOut bytes.Buffer
		if code := run(argv, &out, &errOut); code != 0 {
			t.Fatalf("run(%v) = %d", argv, code)
		}
		if !strings.HasPrefix(out.String(), "gdloc ") {
			t.Errorf("run(%v) stdout = %q", argv, out.String())
		}
	}
}

func TestInvalidSort(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{samplePath(), "--sort", "complexity"}, &out, &errOut); code != 1 {
		t.Errorf("run invalid sort = %d, want 1", code)
	}
}

func TestNegativeTop(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"--top", "-1", samplePath()}, &out, &errOut); code != 1 {
		t.Errorf("run negative top = %d, want 1", code)
	}
}

func TestTooManyArgs(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{samplePath(), samplePath()}, &out, &errOut); code != 1 {
		t.Errorf("run too many args = %d, want 1", code)
	}
}

func TestNonexistentPath(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"definitely-not-a-real-path-xyz"}, &out, &errOut); code != 2 {
		t.Errorf("run nonexistent = %d, want 2", code)
	}
}

func TestJSONOutput(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{samplePath(), "--json", "--by-file"}, &out, &errOut); code != 0 {
		t.Fatalf("run json = %d, stderr=%s", code, errOut.String())
	}
	var got struct {
		Root      string `json:"root"`
		Languages []struct {
			Language string `json:"language"`
			Files    int    `json:"files"`
		} `json:"languages"`
		Total struct {
			Files int `json:"files"`
		} `json:"total"`
		Files []json.RawMessage `json:"files"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if len(got.Languages) == 0 || got.Total.Files == 0 || len(got.Files) == 0 {
		t.Errorf("unexpected JSON content: %s", out.String())
	}
}

func TestExcludeDirFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{samplePath(), "--exclude-dir", "shaders", "--by-file"}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d", code)
	}
	if strings.Contains(out.String(), "shaders/") {
		t.Errorf("excluded dir still present:\n%s", out.String())
	}
}
