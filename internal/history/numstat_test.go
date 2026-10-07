package history

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// codeLines 生成 n 行纯代码行；用例里所有文件都只含代码行，
// 因此 +Code / -Code 就等于 git numstat 的新增 / 删除行数。
func codeLines(prefix string, n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "var %s_%d = %d\n", prefix, i, i)
	}
	return b.String()
}

// numstatOf 返回单个提交的新增/删除行数（开启重命名检测）。
// 用 --minimal：gdloc 的逐行 diff 是最小编辑脚本，而 git 默认的启发式在
// 大段改写时可能多报几行；在真实项目上核对过，--minimal 的结果与 gdloc 一致。
func numstatOf(t *testing.T, r *testRepo, sha string) (int, int) {
	t.Helper()
	out := r.git("diff-tree", "-r", "-M", "--minimal", "--numstat", "--no-commit-id", "--root", sha)
	added, deleted := 0, 0
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			continue
		}
		a, err1 := strconv.Atoi(fields[0])
		d, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil {
			t.Fatalf("unexpected numstat line %q", line)
		}
		added += a
		deleted += d
	}
	return added, deleted
}

// TestDeltaMatchesGitNumstat 校验逐行分类用到的行差异与 git 自己的 diff 等价：
// 由于所有行都是代码行，每个提交的 +Code / -Code 应当等于 numstat 的新增 / 删除行数。
func TestDeltaMatchesGitNumstat(t *testing.T) {
	r := newTestRepo(t)

	a20 := codeLines("a", 20)
	r.write("a.gd", a20)
	r.write("b.gd", codeLines("b", 10))
	r.write("c.gd", codeLines("c", 5))
	first := r.commit("2024-01-01T10:00:00+08:00", "c1")

	// 删除、插入、替换、重命名、整份删除混在一起。
	mid := strings.Split(strings.TrimSuffix(a20, "\n"), "\n")
	edited := append([]string{"var head = 1", "var head2 = 2", "var head3 = 3"}, mid[:5]...)
	edited = append(edited, "var replaced = 99")
	edited = append(edited, mid[6:11]...)
	edited = append(edited, mid[14:]...)
	edited = append(edited, "var tail = 1", "var tail2 = 2")
	r.write("a.gd", strings.Join(edited, "\n")+"\n")
	r.remove("b.gd")
	r.write("b2.gd", codeLines("b", 10)+"var extra = 1\n")
	r.remove("c.gd")
	second := r.commit("2024-01-02T10:00:00+08:00", "c2")

	// 整份重写。
	r.write("a.gd", codeLines("rewritten", 20))
	third := r.commit("2024-01-03T10:00:00+08:00", "c3")

	// 插入一行与已有行完全相同的行。
	lines := strings.Split(strings.TrimSuffix(codeLines("rewritten", 20), "\n"), "\n")
	lines = append(lines[:10], append([]string{lines[9]}, lines[10:]...)...)
	r.write("a.gd", strings.Join(lines, "\n")+"\n")
	fourth := r.commit("2024-01-04T10:00:00+08:00", "c4")

	res := collect(t, r.dir, Options{Since: testDay(2024, 1, 1), Until: testDay(2024, 1, 4)})
	shas := []string{first, second, third, fourth}
	if len(res.Periods) != len(shas) {
		t.Fatalf("periods = %d, want %d", len(res.Periods), len(shas))
	}
	for i, sha := range shas {
		wantAdded, wantDeleted := numstatOf(t, r, sha)
		p := res.Periods[i]
		if p.AddedCode != wantAdded || p.DeletedCode != wantDeleted {
			t.Errorf("%s: gdloc +%d/-%d, git numstat +%d/-%d",
				p.Date.Format(dateLayout), p.AddedCode, p.DeletedCode, wantAdded, wantDeleted)
		}
	}
}
