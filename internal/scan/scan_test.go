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

// writeFile 写入指定内容的文件，用于构造 .gitignore。
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error: %v", path, err)
	}
}

func checkScan(t *testing.T, root string, opts Options, want []FileEntry) {
	t.Helper()
	got, err := Scan(root, opts)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scan =\n%v\nwant\n%v", got, want)
	}
}

func TestGitignoreBasic(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "*.log\n")
	writeTree(t, root, "a.gd", "a.log", "sub/b.gd", "sub/b.log")
	checkScan(t, root, Options{}, []FileEntry{
		{Path: ".gitignore", Type: TypeUnknown},
		{Path: "a.gd", Type: TypeGDScript},
		{Path: "sub/b.gd", Type: TypeGDScript},
	})
}

func TestGitignoreNegation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "*.log\n!keep.log\n")
	writeTree(t, root, "a.log", "keep.log", "b.gd")
	checkScan(t, root, Options{}, []FileEntry{
		{Path: ".gitignore", Type: TypeUnknown},
		{Path: "b.gd", Type: TypeGDScript},
		{Path: "keep.log", Type: TypeUnknown},
	})
}

func TestGitignoreAnchored(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "/build\n")
	writeTree(t, root, "build/a.gd", "sub/build/b.gd", "root.gd")
	checkScan(t, root, Options{}, []FileEntry{
		{Path: ".gitignore", Type: TypeUnknown},
		{Path: "root.gd", Type: TypeGDScript},
		{Path: "sub/build/b.gd", Type: TypeGDScript},
	})
}

func TestGitignoreDirOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "cache/\n")
	writeTree(t, root, "cache/x.gd", "top/cache", "note.txt")
	checkScan(t, root, Options{}, []FileEntry{
		{Path: ".gitignore", Type: TypeUnknown},
		{Path: "note.txt", Type: TypeUnknown},
		{Path: "top/cache", Type: TypeUnknown},
	})
}

func TestGitignoreDoubleStar(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "**/temp\n")
	writeTree(t, root, "temp", "a/temp", "a/b/temp", "a/keep.gd")
	checkScan(t, root, Options{}, []FileEntry{
		{Path: ".gitignore", Type: TypeUnknown},
		{Path: "a/keep.gd", Type: TypeGDScript},
	})
}

func TestGitignoreSubdirOverride(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "*.log\n")
	writeFile(t, filepath.Join(root, "sub", ".gitignore"), "!keep.log\n")
	writeTree(t, root, "root.log", "sub/other.log", "sub/keep.log", "sub/ok.gd")
	checkScan(t, root, Options{}, []FileEntry{
		{Path: ".gitignore", Type: TypeUnknown},
		{Path: "sub/.gitignore", Type: TypeUnknown},
		{Path: "sub/keep.log", Type: TypeUnknown},
		{Path: "sub/ok.gd", Type: TypeGDScript},
	})
}

func TestGitignoreSubdirScoped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sub", ".gitignore"), "*.tmp\n")
	writeTree(t, root, "x.tmp", "sub/y.tmp", "sub/z.gd")
	checkScan(t, root, Options{}, []FileEntry{
		{Path: "sub/.gitignore", Type: TypeUnknown},
		{Path: "sub/z.gd", Type: TypeGDScript},
		{Path: "x.tmp", Type: TypeUnknown},
	})
}

func TestGitignoreNoIgnore(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "*.log\n")
	writeTree(t, root, "a.log", "b.gd")
	checkScan(t, root, Options{NoIgnore: true}, []FileEntry{
		{Path: ".gitignore", Type: TypeUnknown},
		{Path: "a.log", Type: TypeUnknown},
		{Path: "b.gd", Type: TypeGDScript},
	})
}
