package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/qwqzhanqwq/gdloc/internal/godot"
	"github.com/qwqzhanqwq/gdloc/internal/history"
	"github.com/qwqzhanqwq/gdloc/internal/report"
)

// historyArgs 是从命令行收集到的历史统计配置。
type historyArgs struct {
	root          string
	addonsDir     string
	excludeDirs   []string
	excludeAddons bool
	noIgnore      bool
	weekly        bool
	since         time.Time
	until         time.Time
	jsonOut       bool
	top           int
}

// runHistory 执行 --daily / --weekly，返回进程退出码。
// 日期范围矛盾按参数错误（1）处理；不在 git 仓库内或找不到 git 按环境错误（2）处理。
func runHistory(a historyArgs, stdout, stderr io.Writer) int {
	res, err := history.Collect(history.Options{
		Root:          a.root,
		AddonsDir:     a.addonsDir,
		ExcludeDirs:   a.excludeDirs,
		ExcludeAddons: a.excludeAddons,
		NoIgnore:      a.noIgnore,
		Weekly:        a.weekly,
		Since:         a.since,
		Until:         a.until,
		Location:      time.Local,
		Warn:          stderr,
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		if errors.Is(err, history.ErrUsage) {
			return 1
		}
		return 2
	}
	if name, ok := godot.FindProjectName(a.root); ok {
		res.ProjectName = name
	}
	if a.jsonOut {
		report.WriteHistoryJSON(stdout, *res, a.top)
	} else {
		report.WriteHistoryTable(stdout, *res, a.top)
	}
	return 0
}

// parseRange 解析 --since / --until，空串表示未指定；日期按本地时区解释。
func parseRange(since, until string) (time.Time, time.Time, error) {
	s, err := parseDate("--since", since)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	u, err := parseDate("--until", until)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return s, u, nil
}

func parseDate(name, value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	t, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s value %q, want YYYY-MM-DD", name, value)
	}
	return t, nil
}
