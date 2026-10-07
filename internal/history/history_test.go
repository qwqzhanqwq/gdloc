package history

import (
	"strings"
	"testing"
)

// TestHistoryBuckets 覆盖分桶本身：根提交、同日多次提交、跨 ISO 周、空日期、范围边界。
func TestHistoryBuckets(t *testing.T) {
	runHistoryCases(t, []historyCase{
		{
			name: "root commit is compared against the empty tree",
			setup: func(r *testRepo) {
				r.write("main.gd", "extends Node\n# comment\n\nvar a = 1\n")
				r.commit("2024-01-03T10:00:00+08:00", "c1")
			},
			opts: Options{Since: testDay(2024, 1, 3), Until: testDay(2024, 1, 3)},
			want: []period{{
				date: "2024-01-03", commits: 1, addCode: 2, addComments: 1, addBlanks: 1, code: 2,
			}},
			headCode: 2, totalCode: 2,
		},
		{
			name: "several commits on the same day",
			setup: func(r *testRepo) {
				r.write("main.gd", "var a = 1\n")
				r.commit("2024-01-03T09:00:00+08:00", "c1")
				r.write("main.gd", "var a = 1\nvar b = 2\n")
				r.commit("2024-01-03T10:00:00+08:00", "c2")
				r.write("main.gd", "var a = 1\n")
				r.commit("2024-01-03T11:00:00+08:00", "c3")
			},
			opts: Options{Since: testDay(2024, 1, 3), Until: testDay(2024, 1, 3)},
			want: []period{{
				date: "2024-01-03", commits: 3, addCode: 2, delCode: 1, code: 1,
			}},
			headCode: 1, totalCode: 1,
		},
		{
			name: "days without commits keep the running code total",
			setup: func(r *testRepo) {
				r.write("main.gd", "var a = 1\nvar b = 2\n")
				r.commit("2024-01-01T10:00:00+08:00", "c1")
				r.write("main.gd", "var a = 1\n")
				r.commit("2024-01-05T10:00:00+08:00", "c2")
			},
			opts: Options{Since: testDay(2024, 1, 1), Until: testDay(2024, 1, 7)},
			want: []period{
				{date: "2024-01-01", commits: 1, addCode: 2, code: 2},
				{date: "2024-01-02", code: 2},
				{date: "2024-01-03", code: 2},
				{date: "2024-01-04", code: 2},
				{date: "2024-01-05", commits: 1, delCode: 1, code: 1},
				{date: "2024-01-06", code: 1},
				{date: "2024-01-07", code: 1},
			},
			headCode: 1, totalCode: 1,
		},
		{
			name: "the same commit lands in different date buckets per timezone",
			setup: func(r *testRepo) {
				// 2024-01-01 02:00 +08:00 在 UTC-08:00 下是 2023-12-31 10:00。
				r.write("main.gd", "var a = 1\n")
				r.commit("2024-01-01T02:00:00+08:00", "c1")
			},
			opts: Options{Since: testDay(2024, 1, 1), Until: testDay(2024, 1, 1)},
			want: []period{{date: "2024-01-01", commits: 1, addCode: 1, code: 1}},
			// 见 TestTimezoneDecidesBucket：换成 UTC-08:00 会落到 2023-12-31。
			headCode: 1, totalCode: 1,
		},
		{
			name: "since and until are inclusive and bound the commit sums",
			setup: func(r *testRepo) {
				r.write("main.gd", "var a = 1\n")
				r.commit("2024-01-05T10:00:00+08:00", "c1")
				r.write("main.gd", "var a = 1\nvar b = 2\n")
				r.commit("2024-01-06T10:00:00+08:00", "c2")
			},
			opts: Options{Since: testDay(2024, 1, 5), Until: testDay(2024, 1, 5)},
			want: []period{{date: "2024-01-05", commits: 1, addCode: 1, code: 1}},
			// Code 是该时段末的总量：1 月 6 日的提交更晚，因此要从基准里减掉。
			headCode: 2, totalCode: 1,
		},
		{
			name: "week containing commits after until still reports the week end total",
			setup: func(r *testRepo) {
				r.write("main.gd", "var a = 1\n")
				r.commit("2024-01-05T10:00:00+08:00", "c1")
				r.write("main.gd", "var a = 1\nvar b = 2\n")
				r.commit("2024-01-06T10:00:00+08:00", "c2")
			},
			opts:     Options{Weekly: true, Since: testDay(2024, 1, 1), Until: testDay(2024, 1, 5)},
			want:     []period{{date: "2024-01-01", commits: 1, addCode: 1, code: 2}},
			headCode: 2, totalCode: 2,
		},
		{
			name: "ISO weeks start on Monday",
			setup: func(r *testRepo) {
				r.write("main.gd", "var a = 1\nvar b = 2\n")
				r.commit("2024-01-07T23:00:00+08:00", "sunday")
				r.write("main.gd", "var a = 1\nvar b = 2\nvar c = 3\n")
				r.commit("2024-01-08T00:30:00+08:00", "monday")
			},
			opts: Options{Weekly: true, Since: testDay(2024, 1, 1), Until: testDay(2024, 1, 14)},
			want: []period{
				{date: "2024-01-01", commits: 1, addCode: 2, code: 2},
				{date: "2024-01-08", commits: 1, addCode: 1, code: 3},
			},
			headCode: 3, totalCode: 3,
		},
		{
			name: "add, modify, delete and rename",
			setup: func(r *testRepo) {
				r.write("a.gd", "var a1 = 1\nvar a2 = 2\n")
				r.write("b.gd", "var b1 = 1\nvar b2 = 2\nvar b3 = 3\n")
				r.write("c.gd", "var c1 = 1\nvar c2 = 2\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
				r.write("a.gd", "var a1 = 1\nvar a2 = 2\nvar a3 = 3\n")
				r.remove("b.gd")
				r.remove("c.gd")
				r.write("c2.gd", "var c1 = 1\nvar c2 = 2\nvar c3 = 3\n")
				r.commit("2024-01-03T10:00:00+08:00", "c2")
			},
			opts: Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 3)},
			want: []period{
				{date: "2024-01-02", commits: 1, addCode: 7, code: 7},
				// 重命名只算内容差异（+1），不做整份删除 + 新增。
				{date: "2024-01-03", commits: 1, addCode: 2, delCode: 3, code: 6},
			},
			headCode: 6, totalCode: 6,
		},
		{
			name: "merge commits are skipped",
			setup: func(r *testRepo) {
				r.write("main.gd", "var m = 1\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
				base := strings.TrimSpace(r.git("rev-parse", "--abbrev-ref", "HEAD"))
				r.git("checkout", "-q", "-b", "feature")
				r.write("feat.gd", "var f = 1\nvar g = 2\n")
				r.commit("2024-01-02T11:00:00+08:00", "c2")
				r.git("checkout", "-q", base)
				r.write("other.gd", "var o = 1\n")
				r.commit("2024-01-03T10:00:00+08:00", "c3")
				r.gitEnv([]string{
					"GIT_AUTHOR_DATE=2024-01-04T10:00:00+08:00",
					"GIT_COMMITTER_DATE=2024-01-04T10:00:00+08:00",
				}, "merge", "-q", "--no-ff", "-m", "merge feature", "feature")
			},
			opts: Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 4)},
			want: []period{
				{date: "2024-01-02", commits: 2, addCode: 3, code: 3},
				{date: "2024-01-03", commits: 1, addCode: 1, code: 4},
				// 合并提交不计入 Commits，也不产生增量。
				{date: "2024-01-04", code: 4},
			},
			headCode: 4, totalCode: 4,
		},
		{
			name: "commits that do not touch counted files are not counted",
			setup: func(r *testRepo) {
				r.write("main.gd", "var a = 1\n")
				r.write("notes.md", "# not counted\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
				r.write("notes.md", "# still not counted\n")
				r.commit("2024-01-03T10:00:00+08:00", "c2")
			},
			opts: Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 3)},
			want: []period{
				{date: "2024-01-02", commits: 1, addCode: 1, code: 1},
				{date: "2024-01-03", code: 1},
			},
			headCode: 1, totalCode: 1,
		},
	})
}
