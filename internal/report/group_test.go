package report

import (
	"path/filepath"
	"testing"

	"gdloc/internal/counter"
	"gdloc/internal/godot"
	"gdloc/internal/scan"
)

func TestGroupByDir(t *testing.T) {
	rep := Report{Files: []FileStat{
		{Path: "main.gd", Language: "GDScript", Result: counter.Result{Lines: 5, Code: 4, Comments: 1}},
		{Path: "scripts/a.gd", Language: "GDScript", Result: counter.Result{Lines: 3, Code: 3}},
		{Path: "addons/x/x.gd", Language: "GDScript", Result: counter.Result{Lines: 2, Code: 2}},
	}}
	groups := GroupByDir(rep)
	byName := map[string]GroupStat{}
	for _, g := range groups {
		byName[g.Name] = g
	}
	if g := byName["(root)"]; g.Files != 1 || g.Result.Lines != 5 {
		t.Errorf("(root) = %+v", g)
	}
	if g := byName["scripts"]; g.Files != 1 || g.Result.Lines != 3 {
		t.Errorf("scripts = %+v", g)
	}
	if g := byName["addons"]; g.Files != 1 || g.Result.Lines != 2 {
		t.Errorf("addons = %+v", g)
	}
}

func TestGroupByAddon(t *testing.T) {
	root := t.TempDir()
	addons := filepath.Join(root, "addons")
	plugins := []godot.Plugin{
		{Dir: filepath.Join(addons, "plug"), Name: "My Plug", Version: "1.0", HasPluginCfg: true},
	}
	rep := Report{Files: []FileStat{
		{Path: "addons/plug/a.gd", Language: "GDScript", Result: counter.Result{Lines: 7, Code: 7}},
		{Path: "main.gd", Language: "GDScript", Result: counter.Result{Lines: 3, Code: 2, Comments: 1}},
		// 嵌套的 addons 不在项目根下，不应被当作插件。
		{Path: "other/addons/foo/foo.gd", Language: "GDScript", Result: counter.Result{Lines: 1, Code: 1}},
	}}
	groups := GroupByAddon(rep, root, addons, plugins)
	byName := map[string]GroupStat{}
	for _, g := range groups {
		byName[g.Name] = g
	}
	if g, ok := byName["My Plug"]; !ok || g.Files != 1 || g.Result.Lines != 7 || g.Version != "1.0" || !g.HasPluginCfg {
		t.Errorf("My Plug = %+v (ok=%v)", g, ok)
	}
	if g, ok := byName["(project)"]; !ok || g.Files != 2 || g.Result.Lines != 4 {
		t.Errorf("(project) = %+v (ok=%v)", g, ok)
	}
}

func TestGroupByAddonEnsuresProject(t *testing.T) {
	root := t.TempDir()
	groups := GroupByAddon(Report{}, root, filepath.Join(root, "addons"), nil)
	if len(groups) != 1 || groups[0].Name != "(project)" {
		t.Errorf("groups = %+v, want lone (project)", groups)
	}
}

func TestGroupInvariant(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample")
	entries, err := scan.Scan(root, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	rep := Build(root, entries, nil)

	for _, tc := range []struct {
		name   string
		groups []GroupStat
	}{
		{"dir", GroupByDir(rep)},
		{"addon", GroupByAddon(rep, root, filepath.Join(root, "addons"), nil)},
	} {
		var files, lines, code, comments, doc, blanks int
		for _, g := range tc.groups {
			files += g.Files
			lines += g.Result.Lines
			code += g.Result.Code
			comments += g.Result.Comments
			doc += g.Result.Doc
			blanks += g.Result.Blanks
		}
		if files != rep.Total.Files || lines != rep.Total.Result.Lines || code != rep.Total.Result.Code ||
			comments != rep.Total.Result.Comments || doc != rep.Total.Result.Doc || blanks != rep.Total.Result.Blanks {
			t.Errorf("%s groups sum (%d,%d,%d,%d,%d,%d) != total (%d,%d,%d,%d,%d,%d)",
				tc.name, files, lines, code, comments, doc, blanks,
				rep.Total.Files, rep.Total.Result.Lines, rep.Total.Result.Code,
				rep.Total.Result.Comments, rep.Total.Result.Doc, rep.Total.Result.Blanks)
		}
	}
}
