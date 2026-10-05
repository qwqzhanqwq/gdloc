package scan

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		want FileType
	}{
		{"main.gd", TypeGDScript},
		{"main.GD", TypeGDScript},
		{"main.Gd", TypeGDScript},
		{"a.test.gd", TypeGDScript},
		{"player.gdshader", TypeShader},
		{"player.GDSHADER", TypeShader},
		{"common.gdshaderinc", TypeShader},
		{"common.GDShaderInc", TypeShader},
		{"util.cs", TypeCSharp},
		{"util.CS", TypeCSharp},
		{"main.tscn", TypeScene},
		{"main.TSCN", TypeScene},
		{"data.tres", TypeResource},
		{"data.TRES", TypeResource},
		{"sprite.png", TypeUnknown},
		{"data.import", TypeUnknown},
		{"Makefile", TypeUnknown},
		{".gitignore", TypeUnknown},
		{"archive.tar.gz", TypeUnknown},
	}
	for _, c := range cases {
		if got := Classify(c.name); got != c.want {
			t.Errorf("Classify(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestScanSample(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample")
	got, err := Scan(root, Options{})
	if err != nil {
		t.Fatalf("Scan(%q) error: %v", root, err)
	}
	want := []FileEntry{
		{Path: "assets/data.import", Type: TypeUnknown},
		{Path: "assets/sprite.png", Type: TypeUnknown},
		{Path: "main.gd", Type: TypeGDScript},
		{Path: "nested/deep/logic.gd", Type: TypeGDScript},
		{Path: "resources/data.tres", Type: TypeResource},
		{Path: "scenes/main.tscn", Type: TypeScene},
		{Path: "shaders/include.gdshaderinc", Type: TypeShader},
		{Path: "shaders/player.gdshader", Type: TypeShader},
		{Path: "src/util.cs", Type: TypeCSharp},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scan(%q) =\n%v\nwant\n%v", root, got, want)
	}
}

func TestScanSkipsGitDir(t *testing.T) {
	// 名为 .git 的目录无法提交到 git，用临时目录动态构造以覆盖其跳过逻辑。
	root := t.TempDir()
	for _, p := range []string{"main.gd", ".git/hooks.gd"} {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error: %v", full, err)
		}
		if err := os.WriteFile(full, []byte("# placeholder\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error: %v", full, err)
		}
	}
	got, err := Scan(root, Options{})
	if err != nil {
		t.Fatalf("Scan(%q) error: %v", root, err)
	}
	want := []FileEntry{{Path: "main.gd", Type: TypeGDScript}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scan(%q) =\n%v\nwant\n%v", root, got, want)
	}
}

// writeTree 在 root 下按相对路径批量创建文本文件。
func writeTree(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error: %v", full, err)
		}
		if err := os.WriteFile(full, []byte("# placeholder\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error: %v", full, err)
		}
	}
}

func TestScanGDIgnore(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, "main.gd", "keep/ok.gd", "skip/.gdignore", "skip/a.gd", "skip/deep/b.gd")
	got, err := Scan(root, Options{})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	want := []FileEntry{
		{Path: "keep/ok.gd", Type: TypeGDScript},
		{Path: "main.gd", Type: TypeGDScript},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scan =\n%v\nwant\n%v", got, want)
	}
}

func TestScanExcludeDir(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, "a/keep.gd", "a/ex/one.gd", "b/ex/two.gd", "ex/top.gd")
	got, err := Scan(root, Options{ExcludeDirs: []string{"ex"}})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	want := []FileEntry{{Path: "a/keep.gd", Type: TypeGDScript}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scan =\n%v\nwant\n%v", got, want)
	}
}
