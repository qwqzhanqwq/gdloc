package main

import (
	"bytes"
	"encoding/json"
	"os"
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

func writeTemp(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newTempProject 构造一个含 project.godot、一个插件和一个根脚本的项目。
func newTempProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTemp(t, filepath.Join(root, "project.godot"), "config_version=5\n\n[application]\nconfig/name=\"Temp Game\"\n")
	writeTemp(t, filepath.Join(root, "addons", "plug", "plugin.cfg"), "[plugin]\nname=\"My Plug\"\nversion=\"1.2.3\"\nscript=\"plug.gd\"\n")
	writeTemp(t, filepath.Join(root, "addons", "plug", "plug.gd"), "extends Node\n# c\nvar x := 1\n")
	writeTemp(t, filepath.Join(root, "main.gd"), "extends Node\n\nfunc _ready():\n\tpass\n")
	return root
}

func TestByAddon(t *testing.T) {
	root := newTempProject(t)
	var out, errOut bytes.Buffer
	if code := run([]string{root, "--by-addon"}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr=%s", code, errOut.String())
	}
	s := out.String()
	if !strings.Contains(s, "My Plug") || !strings.Contains(s, "1.2.3") || !strings.Contains(s, "(project)") {
		t.Errorf("by-addon output:\n%s", s)
	}
	if !strings.Contains(s, "Addon") || !strings.Contains(s, "Version") {
		t.Errorf("missing addon header:\n%s", s)
	}
}

func TestByDir(t *testing.T) {
	root := newTempProject(t)
	var out, errOut bytes.Buffer
	if code := run([]string{root, "--by-dir"}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d", code)
	}
	s := out.String()
	if !strings.Contains(s, "(root)") || !strings.Contains(s, "addons") {
		t.Errorf("by-dir output:\n%s", s)
	}
	if strings.Contains(s, "Version") {
		t.Errorf("by-dir should not have Version column:\n%s", s)
	}
}

func TestMutuallyExclusiveModes(t *testing.T) {
	root := newTempProject(t)
	for _, argv := range [][]string{
		{root, "--by-file", "--by-addon"},
		{root, "--by-file", "--by-dir"},
		{root, "--by-dir", "--by-addon"},
		{root, "--by-addon", "--exclude-addons"},
	} {
		var out, errOut bytes.Buffer
		if code := run(argv, &out, &errOut); code != 1 {
			t.Errorf("run(%v) = %d, want 1", argv, code)
		}
	}
}

func TestExcludeAddons(t *testing.T) {
	root := newTempProject(t)
	var out, errOut bytes.Buffer
	if code := run([]string{root, "--by-file", "--exclude-addons"}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d", code)
	}
	s := out.String()
	if strings.Contains(s, "addons/") {
		t.Errorf("addons should be excluded:\n%s", s)
	}
	if !strings.Contains(s, "main.gd") {
		t.Errorf("main.gd missing:\n%s", s)
	}
}

func TestByAddonExcludeComparison(t *testing.T) {
	// --exclude-addons 的 Total 应等于 --by-addon 中 (project) 一行。
	root := newTempProject(t)

	var addonOut, addonErr bytes.Buffer
	if code := run([]string{root, "--by-addon", "--json"}, &addonOut, &addonErr); code != 0 {
		t.Fatalf("by-addon run = %d", code)
	}
	var addons struct {
		Addons []struct {
			Name  string `json:"name"`
			Files int    `json:"files"`
			Lines int    `json:"lines"`
		} `json:"addons"`
	}
	if err := json.Unmarshal(addonOut.Bytes(), &addons); err != nil {
		t.Fatalf("addons json: %v", err)
	}
	var project struct{ Files, Lines int }
	for _, a := range addons.Addons {
		if a.Name == "(project)" {
			project.Files, project.Lines = a.Files, a.Lines
		}
	}

	var exclOut, exclErr bytes.Buffer
	if code := run([]string{root, "--exclude-addons", "--json"}, &exclOut, &exclErr); code != 0 {
		t.Fatalf("exclude-addons run = %d", code)
	}
	var excl struct {
		Total struct {
			Files int `json:"files"`
			Lines int `json:"lines"`
		} `json:"total"`
	}
	if err := json.Unmarshal(exclOut.Bytes(), &excl); err != nil {
		t.Fatalf("exclude json: %v", err)
	}
	if project.Files != excl.Total.Files || project.Lines != excl.Total.Lines {
		t.Errorf("(project) %d/%d != exclude-addons total %d/%d", project.Files, project.Lines, excl.Total.Files, excl.Total.Lines)
	}
}

func TestByAddonJSON(t *testing.T) {
	root := newTempProject(t)
	var out, errOut bytes.Buffer
	if code := run([]string{root, "--by-addon", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d", code)
	}
	var got struct {
		Addons []struct {
			Name         string `json:"name"`
			Dir          string `json:"dir"`
			Version      string `json:"version"`
			HasPluginCfg bool   `json:"has_plugin_cfg"`
		} `json:"addons"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json: %v\n%s", err, out.String())
	}
	var found bool
	for _, a := range got.Addons {
		if a.Name == "My Plug" {
			found = true
			if a.Version != "1.2.3" || !a.HasPluginCfg || a.Dir != "addons/plug" {
				t.Errorf("addon entry = %+v", a)
			}
		}
	}
	if !found {
		t.Errorf("plugin not in JSON addons: %s", out.String())
	}
}
