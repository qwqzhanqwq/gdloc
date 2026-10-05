package report

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"gdloc/internal/counter"
	"gdloc/internal/scan"
)

func TestComma(t *testing.T) {
	cases := map[int]string{
		0:       "0",
		5:       "5",
		999:     "999",
		1000:    "1,000",
		12345:   "12,345",
		1234567: "1,234,567",
		-1000:   "-1,000",
	}
	for in, want := range cases {
		if got := comma(in); got != want {
			t.Errorf("comma(%d) = %q, want %q", in, got, want)
		}
	}
}

func sampleReport() Report {
	return Report{
		ProjectName: "Demo",
		Root:        "/tmp/demo",
		Languages: []LangStat{
			{Language: "GDScript", Files: 115, Result: counter.Result{Lines: 24927, Code: 19676, Comments: 2263, Doc: 1498, Blanks: 2988}},
			{Language: "Shader", Files: 17, Result: counter.Result{Lines: 1358, Code: 891, Comments: 287, Doc: 160, Blanks: 180}},
		},
		Total: LangStat{Language: "Total", Files: 132, Result: counter.Result{Lines: 26285, Code: 20567, Comments: 2550, Doc: 1658, Blanks: 3168}},
		Files: []FileStat{
			{Path: "a.gd", Language: "GDScript", Result: counter.Result{Lines: 10, Code: 8, Comments: 1, Blanks: 1}},
			{Path: "b.gdshader", Language: "Shader", Result: counter.Result{Lines: 20, Code: 18, Comments: 1, Blanks: 1}},
		},
		CSharpFiles: 1,
	}
}

func TestSortBy(t *testing.T) {
	rep := sampleReport()
	rep.SortBy("code")
	if rep.Languages[0].Language != "GDScript" {
		t.Errorf("sort by code: got %q first", rep.Languages[0].Language)
	}
	rep.SortBy("files")
	if rep.Languages[0].Language != "GDScript" {
		t.Errorf("sort by files: got %q first", rep.Languages[0].Language)
	}
	rep.SortBy("lines")
	if rep.Languages[0].Language != "GDScript" {
		t.Errorf("sort by lines: got %q first", rep.Languages[0].Language)
	}

	rep2 := sampleReport()
	rep2.Languages[0].Result.Comments = 1
	rep2.Languages[1].Result.Comments = 9999
	rep2.SortBy("comments")
	if rep2.Languages[0].Language != "Shader" {
		t.Errorf("sort by comments: got %q first", rep2.Languages[0].Language)
	}

	rep3 := sampleReport()
	rep3.SortBy("code")
	if rep3.Files[0].Path != "b.gdshader" {
		t.Errorf("file sort by code: got %q first", rep3.Files[0].Path)
	}
}

func TestTableTopKeepsTotal(t *testing.T) {
	rep := sampleReport()
	rep.SortBy("code")
	var buf bytes.Buffer
	WriteTable(&buf, rep, ModeLanguage, 1)
	out := buf.String()
	if !strings.Contains(out, "Project: Demo") || !strings.Contains(out, "Root:    /tmp/demo") {
		t.Errorf("missing header:\n%s", out)
	}
	if strings.Contains(out, "Shader") {
		t.Errorf("top=1 should hide Shader row:\n%s", out)
	}
	if !strings.Contains(out, "26,285") || !strings.Contains(out, "20,567") {
		t.Errorf("total should be full:\n%s", out)
	}
	if !strings.Contains(out, "1 C# file(s) not counted") {
		t.Errorf("missing C# note:\n%s", out)
	}
}

func TestTableNoHeaderWithoutProject(t *testing.T) {
	rep := sampleReport()
	rep.ProjectName = ""
	var buf bytes.Buffer
	WriteTable(&buf, rep, ModeLanguage, 0)
	out := buf.String()
	if strings.Contains(out, "Project:") || strings.Contains(out, "Root:") {
		t.Errorf("unexpected header without project name:\n%s", out)
	}
	// Scene/Resource 文件数为 0 时不显示。
	if strings.Contains(out, "Scene") || strings.Contains(out, "Resource") {
		t.Errorf("empty Scene/Resource rows should be hidden:\n%s", out)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	rep := sampleReport()
	rep.SortBy("code")
	var buf bytes.Buffer
	WriteJSON(&buf, rep, ModeFile, 0)

	var got struct {
		ProjectName string `json:"project_name"`
		Root        string `json:"root"`
		Languages   []struct {
			Language string `json:"language"`
			Files    int    `json:"files"`
			Code     int    `json:"code"`
		} `json:"languages"`
		Total struct {
			Files    int `json:"files"`
			Lines    int `json:"lines"`
			Code     int `json:"code"`
			Comments int `json:"comments"`
			Doc      int `json:"doc"`
			Blanks   int `json:"blanks"`
		} `json:"total"`
		Files []struct {
			Path string `json:"path"`
			Code int    `json:"code"`
		} `json:"files"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal JSON: %v\n%s", err, buf.String())
	}
	if got.ProjectName != "Demo" || got.Root != "/tmp/demo" {
		t.Errorf("header fields = %q, %q", got.ProjectName, got.Root)
	}
	if got.Total.Code != 20567 || got.Total.Lines != 26285 || got.Total.Files != 132 {
		t.Errorf("total = %+v", got.Total)
	}
	if len(got.Languages) != 2 || got.Languages[0].Language != "GDScript" || got.Languages[0].Code != 19676 {
		t.Errorf("languages = %+v", got.Languages)
	}
	if len(got.Files) != 2 || got.Files[0].Path != "b.gdshader" {
		t.Errorf("files = %+v", got.Files)
	}
}

func TestJSONTopTruncatesButTotalFull(t *testing.T) {
	rep := sampleReport()
	rep.SortBy("code")
	var buf bytes.Buffer
	WriteJSON(&buf, rep, ModeLanguage, 1)
	var got struct {
		Languages []json.RawMessage `json:"languages"`
		Total     struct {
			Code int `json:"code"`
		} `json:"total"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Languages) != 1 {
		t.Errorf("languages length = %d, want 1", len(got.Languages))
	}
	if got.Total.Code != 20567 {
		t.Errorf("total code = %d, want 20567", got.Total.Code)
	}
}

func TestBuildSample(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample")
	entries, err := scan.Scan(root, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	rep := Build(root, entries, nil)
	if rep.CSharpFiles != 1 {
		t.Errorf("CSharpFiles = %d, want 1", rep.CSharpFiles)
	}
	byLang := map[string]LangStat{}
	for _, l := range rep.Languages {
		byLang[l.Language] = l
	}
	if byLang["GDScript"].Files != 2 {
		t.Errorf("GDScript files = %d, want 2", byLang["GDScript"].Files)
	}
	if byLang["Shader"].Files != 2 {
		t.Errorf("Shader files = %d, want 2", byLang["Shader"].Files)
	}
}

func TestBuildEmbedded(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "embedded")
	entries, err := scan.Scan(root, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	rep := Build(root, entries, nil)
	byLang := map[string]LangStat{}
	for _, l := range rep.Languages {
		byLang[l.Language] = l
	}

	gd := byLang["GDScript (embedded)"]
	if gd.Files != 3 || gd.Blocks != 3 {
		t.Errorf("GDScript (embedded) files=%d blocks=%d, want 3/3", gd.Files, gd.Blocks)
	}
	if gd.Result != (counter.Result{Lines: 20, Code: 13, Comments: 4, Doc: 2, Blanks: 3}) {
		t.Errorf("GDScript (embedded) = %+v", gd.Result)
	}

	sh := byLang["Shader (embedded)"]
	if sh.Files != 3 || sh.Blocks != 3 {
		t.Errorf("Shader (embedded) files=%d blocks=%d, want 3/3", sh.Files, sh.Blocks)
	}
	if sh.Result != (counter.Result{Lines: 25, Code: 12, Comments: 9, Doc: 6, Blanks: 4}) {
		t.Errorf("Shader (embedded) = %+v", sh.Result)
	}

	if rep.Scenes.Files != 7 {
		t.Errorf("Scenes.Files = %d, want 7", rep.Scenes.Files)
	}
	if rep.Resources.Files != 1 {
		t.Errorf("Resources.Files = %d, want 1", rep.Resources.Files)
	}
	if rep.VisualShaders != 1 {
		t.Errorf("VisualShaders = %d, want 1", rep.VisualShaders)
	}

	// 内嵌块以 parent::id 形式出现在 by-file 列表。
	found := false
	for _, f := range rep.Files {
		if f.Path == "embedded_script.tscn::GDScript_qr1jj" && f.Language == "GDScript (embedded)" {
			found = true
		}
	}
	if !found {
		t.Error("embedded by-file entry not found")
	}
}

func TestTableShowsSceneResource(t *testing.T) {
	rep := sampleReport()
	rep.Scenes = LangStat{Language: "Scene", Files: 35, Result: counter.Result{Lines: 3210}}
	rep.Resources = LangStat{Language: "Resource", Files: 12, Result: counter.Result{Lines: 500}}
	rep.VisualShaders = 2
	var buf bytes.Buffer
	WriteTable(&buf, rep, ModeLanguage, 0)
	out := buf.String()
	if !strings.Contains(out, "Scene") || !strings.Contains(out, "3,210") {
		t.Errorf("missing scene row:\n%s", out)
	}
	if !strings.Contains(out, "Resource") || !strings.Contains(out, "500") {
		t.Errorf("missing resource row:\n%s", out)
	}
	if !strings.Contains(out, "2 VisualShader resource(s) not counted") {
		t.Errorf("missing visual shader note:\n%s", out)
	}
}
