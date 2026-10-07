package history

import "testing"

// TestUncommitted 覆盖工作区（含暂存区）相对 HEAD 的变化：已暂存、未暂存、未跟踪、.gitignore、CRLF。
func TestUncommitted(t *testing.T) {
	// 所有用例共用的基准提交：main.gd 有 2 行代码。
	base := func(r *testRepo) {
		r.write("main.gd", "var a = 1\nvar b = 2\n")
		r.commit("2024-01-02T10:00:00+08:00", "c1")
	}
	basePeriods := []period{{date: "2024-01-02", commits: 1, addCode: 2, code: 2}}

	cases := []struct {
		name  string
		setup func(r *testRepo)
		opts  Options
		want  counts
	}{
		{
			name:  "clean worktree",
			setup: func(r *testRepo) {},
			want:  counts{code: 2},
		},
		{
			name: "unstaged modification",
			setup: func(r *testRepo) {
				r.write("main.gd", "var a = 1\nvar b = 2\nvar c = 3\n# note\n")
			},
			want: counts{addCode: 1, addComments: 1, code: 3},
		},
		{
			name: "staged modification",
			setup: func(r *testRepo) {
				r.write("main.gd", "var a = 1\nvar b = 2\nvar c = 3\n# note\n")
				r.git("add", "-A")
			},
			want: counts{addCode: 1, addComments: 1, code: 3},
		},
		{
			name: "new file staged",
			setup: func(r *testRepo) {
				r.write("staged.gd", "var s = 1\nvar t = 2\n")
				r.git("add", "-A")
			},
			want: counts{addCode: 2, code: 4},
		},
		{
			name: "untracked file counts as a whole new file",
			setup: func(r *testRepo) {
				r.write("new.gd", "var n = 1\n\n# c\n")
			},
			want: counts{addCode: 1, addComments: 1, addBlanks: 1, code: 3},
		},
		{
			name: "untracked scene only contributes its embedded code",
			setup: func(r *testRepo) {
				r.write("scene.tscn", "[gd_scene format=3]\n\n"+tscnScript("GDScript_a", "extends Node\nvar a = 1\n"))
			},
			want: counts{addCode: 2, code: 4},
		},
		{
			name: "untracked file ignored by .gitignore",
			setup: func(r *testRepo) {
				r.write(".gitignore", "ignored.gd\n")
				r.write("ignored.gd", "var i = 1\nvar j = 2\n")
			},
			want: counts{code: 2},
		},
		{
			name: "no-ignore includes gitignored untracked files",
			setup: func(r *testRepo) {
				r.write(".gitignore", "ignored.gd\n")
				r.write("ignored.gd", "var i = 1\nvar j = 2\n")
			},
			opts: Options{NoIgnore: true},
			want: counts{addCode: 2, code: 4},
		},
		{
			name: "untracked file inside an excluded directory",
			setup: func(r *testRepo) {
				r.write("vendor/v.gd", "var v = 1\nvar w = 2\n")
			},
			opts: Options{ExcludeDirs: []string{"vendor"}},
			want: counts{code: 2},
		},
		{
			name: "deleted in the worktree",
			setup: func(r *testRepo) {
				r.remove("main.gd")
			},
			want: counts{delCode: 2, code: 0},
		},
		{
			name: "CRLF worktree file with LF blobs is not a change",
			setup: func(r *testRepo) {
				r.write("main.gd", "var a = 1\r\nvar b = 2\r\n")
			},
			want: counts{code: 2},
		},
		{
			name: "staged rename keeps only the content difference",
			setup: func(r *testRepo) {
				r.git("mv", "main.gd", "renamed.gd")
				r.write("renamed.gd", "var a = 1\nvar b = 2\nvar c = 3\n")
			},
			want: counts{addCode: 1, code: 3},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRepo(t)
			base(r)
			tc.setup(r)
			opts := tc.opts
			opts.Since, opts.Until = testDay(2024, 1, 2), testDay(2024, 1, 2)
			opts.AddonsDir = r.dir + "/addons"
			res := collect(t, r.dir, opts)
			checkResult(t, res, basePeriods, &tc.want, 2, 2)
		})
	}
}
