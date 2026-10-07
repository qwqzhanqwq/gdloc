package history

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// testTZ 是测试用的固定时区（+08:00），避免结果依赖机器所在时区。
const testTZ = 8 * 3600

func testLoc() *time.Location { return time.FixedZone("test", testTZ) }

// testNow 是测试里假定的"今天"。
func testNow() time.Time { return time.Date(2024, 2, 1, 12, 0, 0, 0, testLoc()) }

// testDay 构造测试时区下的某一天。
func testDay(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, testLoc())
}

// testRepo 在 t.TempDir() 里现场建一个 git 仓库，并隔离本机 git 配置。
type testRepo struct {
	t   *testing.T
	dir string
}

func newTestRepo(t *testing.T) *testRepo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	dir := t.TempDir()
	// 空的全局/系统配置：测试不受本机 core.autocrlf、diff.renames 等设置影响。
	cfg := filepath.Join(dir, "empty-gitconfig")
	if err := os.WriteFile(cfg, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	t.Setenv("GIT_CONFIG_SYSTEM", cfg)
	t.Setenv("GIT_AUTHOR_NAME", "gdloc test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "gdloc test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	r := &testRepo{t: t, dir: dir}
	r.git("init", "-q")
	r.git("config", "core.autocrlf", "false")
	return r
}

// git 在仓库里执行 git 命令，失败即终止测试。
func (r *testRepo) git(args ...string) string {
	r.t.Helper()
	return r.gitEnv(nil, args...)
}

// gitEnv 额外带上环境变量（如固定的提交日期）。
func (r *testRepo) gitEnv(env []string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// write 写入工作区文件，父目录自动创建。
func (r *testRepo) write(rel, content string) {
	r.t.Helper()
	p := filepath.Join(r.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *testRepo) remove(rel string) {
	r.t.Helper()
	if err := os.Remove(filepath.Join(r.dir, filepath.FromSlash(rel))); err != nil {
		r.t.Fatal(err)
	}
}

// commit 暂存全部改动并提交，作者与提交日期都用固定值。
func (r *testRepo) commit(date, msg string) string {
	r.t.Helper()
	r.git("add", "-A")
	r.gitEnv([]string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}, "commit", "-q", "-m", msg)
	return strings.TrimSpace(r.git("rev-parse", "HEAD"))
}

// period 是便于断言的时段快照。
type period struct {
	date                     string
	commits                  int
	addCode, delCode         int
	addComments, delComments int
	addBlanks, delBlanks     int
	code                     int
}

// counts 是便于断言的新增/删除快照。
type counts struct {
	addCode, delCode         int
	addComments, delComments int
	addBlanks, delBlanks     int
	code                     int
}

func periodsOf(res *Result) []period {
	out := make([]period, 0, len(res.Periods))
	for _, p := range res.Periods {
		out = append(out, period{
			date: p.Date.Format(dateLayout), commits: p.Commits,
			addCode: p.AddedCode, delCode: p.DeletedCode,
			addComments: p.AddedComments, delComments: p.DeletedComments,
			addBlanks: p.AddedBlanks, delBlanks: p.DeletedBlanks,
			code: p.Code,
		})
	}
	return out
}

func changesOf(d Delta, code int) counts {
	return counts{
		addCode: d.AddedCode, delCode: d.DeletedCode,
		addComments: d.AddedComments, delComments: d.DeletedComments,
		addBlanks: d.AddedBlanks, delBlanks: d.DeletedBlanks,
		code: code,
	}
}

// collect 以固定的时区与"今天"运行一次统计。
func collect(t *testing.T, root string, opts Options) *Result {
	t.Helper()
	opts.Root = root
	if opts.Location == nil {
		opts.Location = testLoc()
	}
	if opts.Now.IsZero() {
		opts.Now = testNow()
	}
	res, err := Collect(opts)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	return res
}

// checkResult 校验时段、Total 与 (uncommitted)；wantUncommitted 为 nil 表示期望无未提交改动。
func checkResult(t *testing.T, res *Result, wantPeriods []period, wantUncommitted *counts, headCode, totalCode int) {
	t.Helper()
	if got := periodsOf(res); !reflect.DeepEqual(got, wantPeriods) {
		t.Errorf("periods mismatch\n got: %s\nwant: %s", formatPeriods(got), formatPeriods(wantPeriods))
	}
	want := counts{code: headCode}
	if wantUncommitted != nil {
		want = *wantUncommitted
	}
	if got := changesOf(res.Uncommitted, res.UncommittedCode); got != want {
		t.Errorf("(uncommitted) = %+v, want %+v", got, want)
	}
	if res.HeadCode != headCode {
		t.Errorf("HeadCode = %d, want %d", res.HeadCode, headCode)
	}
	if res.TotalCode != totalCode {
		t.Errorf("TotalCode = %d, want %d", res.TotalCode, totalCode)
	}
	// Total 只汇总范围内时段，不含 (uncommitted)。
	var sum Delta
	for _, p := range res.Periods {
		sum.add(p.Delta)
	}
	if !reflect.DeepEqual(res.Total, sum) {
		t.Errorf("Total = %+v, want %+v", res.Total, sum)
	}
	if len(res.Periods) > 0 && res.TotalCode != res.Periods[len(res.Periods)-1].Code {
		t.Errorf("TotalCode = %d, want last period code %d", res.TotalCode, res.Periods[len(res.Periods)-1].Code)
	}
}

func formatPeriods(ps []period) string {
	var b strings.Builder
	for i, p := range ps {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s{c%d +%d/-%d m+%d/-%d b+%d/-%d =%d}",
			p.date, p.commits, p.addCode, p.delCode, p.addComments, p.delComments, p.addBlanks, p.delBlanks, p.code)
	}
	return b.String()
}

// tscnScript 生成一段内嵌 GDScript 的 tscn 片段，source 用转义写法。
func tscnScript(id, source string) string {
	return fmt.Sprintf("[sub_resource type=\"GDScript\" id=%q]\nscript/source = %q\n", id, source)
}

// historyCase 是一个历史统计用例：现场建仓库、跑一次 Collect、比对整张表。
type historyCase struct {
	name            string
	subdir          string // 非空时扫描根取仓库下的该子目录
	setup           func(r *testRepo)
	opts            Options
	want            []period
	wantUncommitted *counts
	headCode        int
	totalCode       int
}

func runHistoryCases(t *testing.T, cases []historyCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRepo(t)
			root := r.dir
			if tc.subdir != "" {
				root = filepath.Join(r.dir, tc.subdir)
			}
			// 与 CLI 一致：addons 目录按扫描根推算。
			tc.opts.AddonsDir = filepath.Join(root, "addons")
			tc.setup(r)
			res := collect(t, root, tc.opts)
			checkResult(t, res, tc.want, tc.wantUncommitted, tc.headCode, tc.totalCode)
		})
	}
}
