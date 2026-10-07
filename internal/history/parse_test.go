package history

import (
	"strings"
	"testing"
	"time"
)

// TestParseRawEntries 覆盖 git --raw -z 的解析：新增、修改、删除、重命名与中文路径。
func TestParseRawEntries(t *testing.T) {
	input := strings.Join([]string{
		":000000 100644 " + zeroSHA + " 76e6e36ed7a3470fd1511237502622d5e5896af3 A",
		"added.gd",
		":100644 100644 196edf65d5cdcc8c89a50f54ec5d65a300705177 54bcea8cd0c558b25b1a0ebfc5050000000000 M",
		"sub/中文.gd",
		":100644 000000 cd0c558b25b1a0ebfc5050000000000000000000 " + zeroSHA + " D",
		"deleted.gd",
		":100644 100644 196edf65d5cdcc8c89a50f54ec5d65a300705177 196edf65d5cdcc8c89a50f54ec5d65a300705177 R100",
		"old.gd",
		"new.gd",
		"",
	}, "\x00")

	got, err := parseRawEntries([]byte(input))
	if err != nil {
		t.Fatalf("parseRawEntries: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("entries = %d, want 4", len(got))
	}
	if got[0].status != 'A' || got[0].newPath != "added.gd" || got[0].oldPath != "added.gd" {
		t.Errorf("added entry = %+v", got[0])
	}
	if !regularFile(got[0].newMode) || regularFile(got[0].oldMode) {
		t.Errorf("added modes = %q/%q", got[0].oldMode, got[0].newMode)
	}
	if got[1].status != 'M' || got[1].newPath != "sub/中文.gd" {
		t.Errorf("modified entry = %+v", got[1])
	}
	if got[2].status != 'D' || got[2].newSHA != zeroSHA {
		t.Errorf("deleted entry = %+v", got[2])
	}
	if got[3].status != 'R' || got[3].oldPath != "old.gd" || got[3].newPath != "new.gd" {
		t.Errorf("renamed entry = %+v", got[3])
	}
}

// TestParseLog 覆盖 log 头部与 raw 记录交替出现的流。
func TestParseLog(t *testing.T) {
	data := "\x1e" + strings.Repeat("a", 40) + "\x00" +
		"2024-01-02T10:00:00+08:00" + "\x00" + strings.Repeat("b", 40) + "\x1f\x00\n" +
		":100644 100644 " + strings.Repeat("c", 40) + " " + strings.Repeat("d", 40) + " M\x00main.gd\x00" +
		"\x1e" + strings.Repeat("e", 40) + "\x00" +
		"2024-01-03T10:00:00+08:00" + "\x00\x1f\x00\n" // 空提交：没有改动

	got, err := parseLog([]byte(data))
	if err != nil {
		t.Fatalf("parseLog: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("commits = %d, want 2", len(got))
	}
	want := time.Date(2024, 1, 2, 10, 0, 0, 0, time.FixedZone("", 8*3600))
	if !got[0].date.Equal(want) {
		t.Errorf("date = %v, want %v", got[0].date, want)
	}
	if len(got[0].changes) != 1 || got[0].changes[0].newPath != "main.gd" {
		t.Errorf("changes = %+v", got[0].changes)
	}
	if len(got[1].changes) != 0 {
		t.Errorf("empty commit has changes: %+v", got[1].changes)
	}
}

// TestParseTree 覆盖 ls-tree -z 的解析，并跳过软链接与子模块。
func TestParseTree(t *testing.T) {
	data := "100644 blob " + strings.Repeat("a", 40) + "\tmain.gd\x00" +
		"100755 blob " + strings.Repeat("b", 40) + "\trun.sh\x00" +
		"120000 blob " + strings.Repeat("c", 40) + "\tlink.gd\x00" +
		"160000 commit " + strings.Repeat("d", 40) + "\tsub\x00"

	got := parseTree([]byte(data))
	if len(got) != 4 {
		t.Fatalf("entries = %d, want 4", len(got))
	}
	if got[0].path != "main.gd" || got[0].sha != strings.Repeat("a", 40) {
		t.Errorf("entry 0 = %+v", got[0])
	}
	if !regularFile(got[0].mode) || !regularFile(got[1].mode) {
		t.Errorf("regular files not recognized: %+v", got)
	}
	if regularFile(got[2].mode) || regularFile(got[3].mode) {
		t.Errorf("symlink/submodule must not be regular files: %+v", got)
	}
	if regularFile("") || regularFile("000000") {
		t.Error("missing modes must not be regular files")
	}
}

func TestSplitZ(t *testing.T) {
	got := splitZ([]byte("a.gd\x00sub/b.gd\x00"))
	if len(got) != 2 || got[0] != "a.gd" || got[1] != "sub/b.gd" {
		t.Errorf("splitZ = %q", got)
	}
	if got := splitZ(nil); len(got) != 0 {
		t.Errorf("splitZ(nil) = %q", got)
	}
}
