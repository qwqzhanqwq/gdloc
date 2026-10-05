package godot

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanPlugins(t *testing.T) {
	addons := t.TempDir()
	writeFile(t, filepath.Join(addons, "plugA", "plugin.cfg"), "[plugin]\n\nname=\"Alpha Tools\"\nversion=\"1.2.3\"\nscript=\"a.gd\"\n")
	if err := os.MkdirAll(filepath.Join(addons, "plugB"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(addons, "plugC", "plugin.cfg"), "[plugin]\nname=\n")

	plugins := ScanPlugins(addons, io.Discard)
	if len(plugins) != 3 {
		t.Fatalf("plugins = %d, want 3", len(plugins))
	}
	a, b, c := plugins[0], plugins[1], plugins[2]
	if a.Name != "Alpha Tools" || a.Version != "1.2.3" || !a.HasPluginCfg {
		t.Errorf("plugA = %+v", a)
	}
	if b.Name != "plugB" || b.Version != "" || b.HasPluginCfg {
		t.Errorf("plugB = %+v", b)
	}
	if c.Name != "plugC" || c.HasPluginCfg {
		t.Errorf("plugC = %+v", c)
	}
}

func TestScanPluginsWarnsOnBrokenCfg(t *testing.T) {
	addons := t.TempDir()
	writeFile(t, filepath.Join(addons, "broken", "plugin.cfg"), "this is not ini\n")

	var warn bytes.Buffer
	plugins := ScanPlugins(addons, &warn)
	if len(plugins) != 1 || plugins[0].Name != "broken" || plugins[0].HasPluginCfg {
		t.Errorf("plugins = %+v", plugins)
	}
	if !strings.Contains(warn.String(), "cannot parse") {
		t.Errorf("expected warning, got %q", warn.String())
	}
}

func TestScanPluginsMissingDir(t *testing.T) {
	if plugins := ScanPlugins(filepath.Join(t.TempDir(), "nope"), io.Discard); plugins != nil {
		t.Errorf("plugins = %+v, want nil", plugins)
	}
}

func TestFindProjectRootFromSubdir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "project.godot"), "config_version=5\n")
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := FindProjectRoot(sub)
	if !ok {
		t.Fatal("FindProjectRoot not found")
	}
	if filepath.Clean(got) != filepath.Clean(root) {
		t.Errorf("FindProjectRoot = %q, want %q", got, root)
	}
}
