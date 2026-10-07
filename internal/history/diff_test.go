package history

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
)

// minEdits 用 O(n*m) 的 DP 求最短编辑脚本长度（新增 + 删除的行数），作为最小性基准。
func minEdits(a, b []string) int {
	lcs := make([][]int, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] > lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	return len(a) + len(b) - 2*lcs[0][0]
}

// checkDiff 校验编辑脚本能把 a 变成 b，并且是最短的。
func checkDiff(t *testing.T, a, b []string, added, deleted []int) {
	t.Helper()
	del := map[int]bool{}
	for _, i := range deleted {
		del[i] = true
	}
	add := map[int]bool{}
	for _, i := range added {
		add[i] = true
	}

	var out []string
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case i < len(a) && del[i]:
			i++
		case j < len(b) && add[j]:
			out = append(out, b[j])
			j++
		case i < len(a) && j < len(b) && a[i] == b[j]:
			out = append(out, a[i])
			i++
			j++
		default:
			t.Fatalf("invalid diff at a[%d] b[%d]", i, j)
		}
	}
	if !slices.Equal(out, b) {
		t.Fatalf("diff does not transform a into b:\n got %q\nwant %q", out, b)
	}
	if got, want := len(added)+len(deleted), minEdits(a, b); got != want {
		t.Errorf("edit script length = %d, want minimal %d", got, want)
	}
}

func lines(texts ...string) []string { return texts }

func TestDiffLines(t *testing.T) {
	cases := []struct {
		name    string
		a, b    []string
		wantAdd int
		wantDel int
	}{
		{name: "identical", a: lines("a", "b"), b: lines("a", "b")},
		{name: "empty to empty"},
		{name: "empty to three lines", b: lines("a", "b", "c"), wantAdd: 3},
		{name: "three lines to empty", a: lines("a", "b", "c"), wantDel: 3},
		{name: "append at the end", a: lines("a", "b"), b: lines("a", "b", "c"), wantAdd: 1},
		{name: "insert in the middle", a: lines("a", "c"), b: lines("a", "b", "c"), wantAdd: 1},
		{name: "delete in the middle", a: lines("a", "b", "c"), b: lines("a", "c"), wantDel: 1},
		{name: "replace one line", a: lines("a", "b", "c"), b: lines("a", "x", "c"), wantAdd: 1, wantDel: 1},
		{name: "swap two lines", a: lines("a", "b"), b: lines("b", "a"), wantAdd: 1, wantDel: 1},
		{name: "change first and last", a: lines("x", "b", "y"), b: lines("p", "b", "q"), wantAdd: 2, wantDel: 2},
		{name: "duplicated lines", a: lines("a", "a", "b"), b: lines("a", "b", "b"), wantAdd: 1, wantDel: 1},
		{name: "blank line only", a: lines("a", "b"), b: lines("a", "", "b"), wantAdd: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			added, deleted := diffLines(tc.a, tc.b)
			if len(added) != tc.wantAdd || len(deleted) != tc.wantDel {
				t.Errorf("added/deleted = %d/%d, want %d/%d", len(added), len(deleted), tc.wantAdd, tc.wantDel)
			}
			checkDiff(t, tc.a, tc.b, added, deleted)
		})
	}
}

// TestDiffLinesLongRun 覆盖公共前后缀很长的常见情况。
func TestDiffLinesLongRun(t *testing.T) {
	a := make([]string, 500)
	for i := range a {
		a[i] = fmt.Sprintf("var line%d = %d", i, i)
	}
	b := append(append([]string{}, a[:250]...), "var inserted = 1")
	b = append(b, a[250:]...)

	added, deleted := diffLines(a, b)
	if len(added) != 1 || len(deleted) != 0 {
		t.Fatalf("added/deleted = %d/%d, want 1/0", len(added), len(deleted))
	}
	if added[0] != 250 {
		t.Errorf("added index = %d, want 250", added[0])
	}
	checkDiff(t, a, b, added, deleted)
}

// TestDiffLinesRandom 用随机序列验证脚本正确且最短。
func TestDiffLinesRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	alphabet := []string{"a", "b", "c", ""}
	for i := 0; i < 300; i++ {
		a := randomLines(rng, alphabet)
		b := randomLines(rng, alphabet)
		added, deleted := diffLines(a, b)
		checkDiff(t, a, b, added, deleted)
	}
}

// TestDiffLinesHugeRewrite 覆盖超过回溯上限时退化成整份替换。
func TestDiffLinesHugeRewrite(t *testing.T) {
	a := make([]string, 2500)
	b := make([]string, 2500)
	for i := range a {
		a[i] = fmt.Sprintf("old%d", i)
		b[i] = fmt.Sprintf("new%d", i)
	}
	a[0], b[0] = "start-a", "start-b" // 没有公共前后缀
	added, deleted := diffLines(a, b)
	if len(added) != len(b) || len(deleted) != len(a) {
		t.Errorf("added/deleted = %d/%d, want %d/%d", len(added), len(deleted), len(b), len(a))
	}
}

func randomLines(rng *rand.Rand, alphabet []string) []string {
	n := rng.Intn(9)
	out := make([]string, n)
	for i := range out {
		out[i] = alphabet[rng.Intn(len(alphabet))]
	}
	return out
}
