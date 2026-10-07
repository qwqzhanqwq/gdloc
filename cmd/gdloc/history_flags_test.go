package main

import (
	"bytes"
	"testing"
)

// TestHistoryFlagValidation 覆盖参数互斥、--since/--until 依赖与日期格式（退出码 1）。
func TestHistoryFlagValidation(t *testing.T) {
	r, d1, d2 := historyRepo(t)
	cases := []struct {
		name string
		argv []string
	}{
		{"daily and weekly", []string{r.dir, "--daily", "--weekly"}},
		{"daily with by-file", []string{r.dir, "--daily", "--by-file"}},
		{"weekly with by-dir", []string{r.dir, "--weekly", "--by-dir"}},
		{"daily with by-addon", []string{r.dir, "--daily", "--by-addon"}},
		{"weekly with stats", []string{r.dir, "--weekly", "--stats"}},
		{"since without a mode", []string{r.dir, "--since", d1}},
		{"until without a mode", []string{r.dir, "--until", d2}},
		{"bad since format", []string{r.dir, "--daily", "--since", "2024-3-1"}},
		{"bad until format", []string{r.dir, "--daily", "--until", "20240301"}},
		{"since after until", []string{r.dir, "--daily", "--since", d2, "--until", d1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run(tc.argv, &out, &errOut); code != 1 {
				t.Errorf("run(%v) = %d, want 1 (stderr=%s)", tc.argv, code, errOut.String())
			}
		})
	}
}

// TestHistoryNotAGitRepository 覆盖不在 git 仓库内时的退出码 2。
func TestHistoryNotAGitRepository(t *testing.T) {
	newCLIRepo(t) // 复用 git 存在性检查与隔离配置
	var out, errOut bytes.Buffer
	if code := run([]string{t.TempDir(), "--daily"}, &out, &errOut); code != 2 {
		t.Errorf("run outside a repository = %d, want 2 (stderr=%s)", code, errOut.String())
	}
}
