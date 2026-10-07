package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// cliRepo 是一个隔离的临时 git 仓库，供 CLI 测试使用。
type cliRepo struct {
	t   *testing.T
	dir string
}

func newCLIRepo(t *testing.T) *cliRepo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	dir := t.TempDir()
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
	r := &cliRepo{t: t, dir: dir}
	r.git("init", "-q")
	r.git("config", "core.autocrlf", "false")
	return r
}

func (r *cliRepo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func (r *cliRepo) write(rel, content string) {
	r.t.Helper()
	p := filepath.Join(r.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// commit 用固定的作者日期提交全部改动。
func (r *cliRepo) commit(date, msg string) {
	r.t.Helper()
	r.git("add", "-A")
	cmd := exec.Command("git", "commit", "-q", "-m", msg)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		r.t.Fatalf("git commit: %v\n%s", err, out)
	}
}

// dayAt 把某一时刻换算成 time.Local 下的日期字符串（CLI 用本地时区分组）。
func dayAt(instant time.Time) string { return instant.In(time.Local).Format("2006-01-02") }

// daysBetween 列出 from 到 to（含）之间的日期，用于推算期望行数。
func daysBetween(from, to time.Time) []string {
	var out []string
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format("2006-01-02"))
	}
	return out
}

// historyRepo 建一个有两个提交（相隔 30 小时）的仓库，返回两个提交所在的本地日期。
func historyRepo(t *testing.T) (*cliRepo, string, string) {
	t.Helper()
	r := newCLIRepo(t)
	t1 := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.Add(30 * time.Hour)
	r.write("main.gd", "var a = 1\nvar b = 2\n")
	r.commit(t1.Format(time.RFC3339), "c1")
	r.write("extra.gd", "var c = 1\nvar d = 2\nvar e = 3\n")
	r.commit(t2.Format(time.RFC3339), "c2")
	return r, dayAt(t1), dayAt(t2)
}

func TestHistoryTableOutput(t *testing.T) {
	r, d1, d2 := historyRepo(t)
	var out, errOut bytes.Buffer
	if code := run([]string{r.dir, "--daily", "--since", d1, "--until", d2}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr=%s", code, errOut.String())
	}
	got := out.String()
	for _, want := range []string{"Date", "Commits", "+Code", "-Code", "Net", "+Comments", "Code", "(uncommitted)", "Total", d1, d2} {
		if !strings.Contains(got, want) {
			t.Errorf("table output missing %q:\n%s", want, got)
		}
	}
	// Total 一行汇总两次提交：2 + 3 行代码。
	if !strings.Contains(got, "Total") || !strings.Contains(got, " 5 ") {
		t.Errorf("table output missing the total of 5 code lines:\n%s", got)
	}
}

func TestHistoryJSONOutput(t *testing.T) {
	r, d1, d2 := historyRepo(t)
	var out, errOut bytes.Buffer
	if code := run([]string{r.dir, "--weekly", "--since", d1, "--until", d2, "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr=%s", code, errOut.String())
	}
	var got struct {
		Root    string `json:"root"`
		Mode    string `json:"mode"`
		Since   string `json:"since"`
		Until   string `json:"until"`
		Periods []struct {
			Date      string `json:"date"`
			Commits   int    `json:"commits"`
			AddedCode int    `json:"added_code"`
			NetCode   int    `json:"net_code"`
			Code      int    `json:"code"`
		} `json:"periods"`
		Uncommitted struct {
			AddedCode int `json:"added_code"`
			Code      int `json:"code"`
		} `json:"uncommitted"`
		Total struct {
			Commits   int `json:"commits"`
			AddedCode int `json:"added_code"`
			Code      int `json:"code"`
		} `json:"total"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if got.Mode != "weekly" || got.Since != d1 || got.Until != d2 {
		t.Errorf("header = %+v", got)
	}
	if len(got.Periods) == 0 || got.Total.Commits != 2 || got.Total.AddedCode != 5 {
		t.Errorf("periods/total = %+v / %+v", got.Periods, got.Total)
	}
	// 周标签始终是周一。
	for _, p := range got.Periods {
		day, err := time.ParseInLocation("2006-01-02", p.Date, time.Local)
		if err != nil {
			t.Fatalf("bad period date %q", p.Date)
		}
		if day.Weekday() != time.Monday {
			t.Errorf("week label %s is a %s, want Monday", p.Date, day.Weekday())
		}
	}
	if got.Uncommitted.Code != got.Total.Code {
		t.Errorf("(uncommitted) code = %d, want %d for a clean worktree", got.Uncommitted.Code, got.Total.Code)
	}
}

// TestHistoryCodeMatchesHeadCount 交叉校验期末 Code 总量与直接统计 HEAD 的结果。
func TestHistoryCodeMatchesHeadCount(t *testing.T) {
	r, d1, d2 := historyRepo(t)

	var histOut, histErr bytes.Buffer
	if code := run([]string{r.dir, "--daily", "--since", d1, "--until", d2, "--json"}, &histOut, &histErr); code != 0 {
		t.Fatalf("history run = %d, stderr=%s", code, histErr.String())
	}
	var hist struct {
		Periods []struct {
			Code int `json:"code"`
		} `json:"periods"`
		Total struct {
			Code int `json:"code"`
		} `json:"total"`
	}
	if err := json.Unmarshal(histOut.Bytes(), &hist); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	var headOut, headErr bytes.Buffer
	if code := run([]string{r.dir, "--json"}, &headOut, &headErr); code != 0 {
		t.Fatalf("default run = %d, stderr=%s", code, headErr.String())
	}
	var head struct {
		Total struct {
			Code int `json:"code"`
		} `json:"total"`
	}
	if err := json.Unmarshal(headOut.Bytes(), &head); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(hist.Periods) == 0 {
		t.Fatal("no periods")
	}
	if last := hist.Periods[len(hist.Periods)-1].Code; last != head.Total.Code {
		t.Errorf("last period code = %d, direct HEAD count = %d", last, head.Total.Code)
	}
	if hist.Total.Code != head.Total.Code {
		t.Errorf("total code = %d, direct HEAD count = %d", hist.Total.Code, head.Total.Code)
	}
}

func TestHistoryTopKeepsRecentRows(t *testing.T) {
	r, d1, d2 := historyRepo(t)
	var out, errOut bytes.Buffer
	if code := run([]string{r.dir, "--daily", "--since", d1, "--until", d2, "--top", "1"}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr=%s", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, d2) {
		t.Errorf("--top 1 should keep the most recent period %s:\n%s", d2, got)
	}
	if d1 != d2 && strings.Contains(got, d1+" ") {
		t.Errorf("--top 1 should drop the oldest period %s:\n%s", d1, got)
	}
	for _, want := range []string{"(uncommitted)", "Total"} {
		if !strings.Contains(got, want) {
			t.Errorf("--top should keep the %s row:\n%s", want, got)
		}
	}
}

// TestHistoryDefaultRanges 覆盖默认的 14 天 / 12 周。
func TestHistoryDefaultRanges(t *testing.T) {
	r, _, _ := historyRepo(t)
	for _, tc := range []struct {
		flag string
		want int
	}{
		{"--daily", 14},
		{"--weekly", 12},
	} {
		var out, errOut bytes.Buffer
		if code := run([]string{r.dir, tc.flag, "--json"}, &out, &errOut); code != 0 {
			t.Fatalf("run %s = %d, stderr=%s", tc.flag, code, errOut.String())
		}
		var got struct {
			Periods []struct {
				Date string `json:"date"`
			} `json:"periods"`
		}
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if len(got.Periods) != tc.want {
			t.Errorf("%s periods = %d, want %d", tc.flag, len(got.Periods), tc.want)
		}
	}
}

// TestHistoryUncommittedRow 覆盖未提交改动单独成行、不并入任何日期。
func TestHistoryUncommittedRow(t *testing.T) {
	r, d1, d2 := historyRepo(t)
	r.write("main.gd", "var a = 1\nvar b = 2\nvar fresh = 3\n") // 未暂存的新增

	var out, errOut bytes.Buffer
	if code := run([]string{r.dir, "--daily", "--since", d1, "--until", d2, "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr=%s", code, errOut.String())
	}
	var got struct {
		Periods []struct {
			Date      string `json:"date"`
			AddedCode int    `json:"added_code"`
		} `json:"periods"`
		Uncommitted struct {
			AddedCode int `json:"added_code"`
			Code      int `json:"code"`
		} `json:"uncommitted"`
		Total struct {
			AddedCode int `json:"added_code"`
			Code      int `json:"code"`
		} `json:"total"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got.Uncommitted.AddedCode != 1 {
		t.Errorf("uncommitted added_code = %d, want 1", got.Uncommitted.AddedCode)
	}
	// 未提交改动只出现在 (uncommitted) 行，Total 不受影响。
	if got.Total.AddedCode != 5 {
		t.Errorf("total added_code = %d, want 5", got.Total.AddedCode)
	}
	for _, p := range got.Periods {
		if p.AddedCode > 3 {
			t.Errorf("period %s includes uncommitted lines: %+v", p.Date, p)
		}
	}
	if got.Uncommitted.Code != got.Total.Code+1 {
		t.Errorf("uncommitted code = %d, want %d", got.Uncommitted.Code, got.Total.Code+1)
	}
}
