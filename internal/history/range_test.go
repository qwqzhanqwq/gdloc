package history

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// testDayIn 在指定时区构造某一天。
func testDayIn(loc *time.Location, year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

// TestRangeResolution 覆盖 --since / --until 的默认值与边界（4.7、AGENTS.md 第 6 节）。
func TestRangeResolution(t *testing.T) {
	cases := []struct {
		name      string
		opts      Options
		earliest  time.Time
		wantSince string
		wantUntil string
		wantCount int
		wantErr   bool
	}{
		{
			name: "default daily is the last 14 days", opts: Options{},
			wantSince: "2024-01-19", wantUntil: "2024-02-01", wantCount: 14,
		},
		{
			name: "default weekly is the last 12 weeks", opts: Options{Weekly: true},
			wantSince: "2023-11-13", wantUntil: "2024-02-01", wantCount: 12,
		},
		{
			name: "since only ends today", opts: Options{Since: testDay(2024, 1, 20)},
			wantSince: "2024-01-20", wantUntil: "2024-02-01", wantCount: 13,
		},
		{
			name: "until only starts at the earliest commit", earliest: testDay(2023, 12, 30),
			opts:      Options{Until: testDay(2024, 1, 2)},
			wantSince: "2023-12-30", wantUntil: "2024-01-02", wantCount: 4,
		},
		{
			name: "empty repository with until only degenerates to a single day", earliest: time.Time{},
			opts:      Options{Until: testDay(2024, 1, 2)},
			wantSince: "2024-01-02", wantUntil: "2024-01-02", wantCount: 1,
		},
		{
			name: "since after until is an error",
			opts: Options{Since: testDay(2024, 2, 2), Until: testDay(2024, 2, 1)}, wantErr: true,
		},
		{
			name: "earliest commit after until is an error", earliest: testDay(2024, 1, 1),
			opts: Options{Until: testDay(2020, 1, 1)}, wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			since, until, err := resolveRange(tc.opts, tc.earliest, testLoc(), testNow())
			if tc.wantErr {
				if !errors.Is(err, ErrUsage) {
					t.Fatalf("resolveRange error = %v, want ErrUsage", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveRange: %v", err)
			}
			if got := since.Format(dateLayout); got != tc.wantSince {
				t.Errorf("since = %s, want %s", got, tc.wantSince)
			}
			if got := until.Format(dateLayout); got != tc.wantUntil {
				t.Errorf("until = %s, want %s", got, tc.wantUntil)
			}
			opts := tc.opts
			res := build(opts, nil, 0, Delta{}, since, until)
			if len(res.Periods) != tc.wantCount {
				t.Errorf("periods = %d, want %d", len(res.Periods), tc.wantCount)
			}
		})
	}
}

// TestEmptyRepository 覆盖没有提交的仓库：不报错，数值全为 0。
func TestEmptyRepository(t *testing.T) {
	r := newTestRepo(t)
	res := collect(t, r.dir, Options{Since: testDay(2024, 1, 1), Until: testDay(2024, 1, 3)})
	want := []period{{date: "2024-01-01"}, {date: "2024-01-02"}, {date: "2024-01-03"}}
	checkResult(t, res, want, nil, 0, 0)

	// 未跟踪文件在还没有 HEAD 的仓库里也算新增。
	r.write("main.gd", "var a = 1\n# c\n\n")
	res = collect(t, r.dir, Options{Since: testDay(2024, 1, 1), Until: testDay(2024, 1, 3)})
	checkResult(t, res, want, &counts{addCode: 1, addComments: 1, addBlanks: 1, code: 1}, 0, 0)
}

// TestNotAGitRepository 覆盖扫描根不在 git 仓库内的情况。
func TestNotAGitRepository(t *testing.T) {
	newTestRepo(t) // 只为了复用 git 存在性检查与隔离的 git 配置
	dir := t.TempDir()
	_, err := Collect(Options{Root: dir, Location: testLoc(), Now: testNow()})
	if err == nil {
		t.Fatal("Collect outside a git repository should fail")
	}
	if !strings.Contains(err.Error(), "git") {
		t.Errorf("error = %v, want a git related message", err)
	}
}

// TestTimezoneDecidesBucket 验证同一个提交在不同时区下落到不同日期。
func TestTimezoneDecidesBucket(t *testing.T) {
	r := newTestRepo(t)
	// 2024-01-01 02:00 +08:00 在 UTC-08:00 下是 2023-12-31 10:00。
	r.write("main.gd", "var a = 1\n")
	r.commit("2024-01-01T02:00:00+08:00", "c1")

	east := testLoc()
	west := time.FixedZone("west", -8*3600)

	eastRes := collect(t, r.dir, Options{
		Location: east, Since: testDayIn(east, 2024, 1, 1), Until: testDayIn(east, 2024, 1, 1),
	})
	checkResult(t, eastRes, []period{{date: "2024-01-01", commits: 1, addCode: 1, code: 1}}, nil, 1, 1)

	westRes := collect(t, r.dir, Options{
		Location: west, Since: testDayIn(west, 2023, 12, 31), Until: testDayIn(west, 2023, 12, 31),
	})
	checkResult(t, westRes, []period{{date: "2023-12-31", commits: 1, addCode: 1, code: 1}}, nil, 1, 1)
}
