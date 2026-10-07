package history

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"time"
)

// ErrUsage 表示日期范围本身矛盾；调用方按参数错误（退出码 1）处理。
var ErrUsage = errors.New("invalid date range")

// dateLayout 是 --since / --until 与表格里的日期格式。
const dateLayout = "2006-01-02"

// Options 控制一次历史统计。
type Options struct {
	Root          string         // 扫描根目录（绝对路径）
	AddonsDir     string         // 项目 addons 目录（绝对路径，可空）
	ExcludeDirs   []string       // 与 --exclude-dir 一致
	ExcludeAddons bool           // 与 --exclude-addons 一致
	NoIgnore      bool           // 与 --no-ignore 一致：未跟踪文件不再套用 .gitignore
	Weekly        bool           // 按 ISO 周而不是按天
	Since, Until  time.Time      // 零值表示未指定
	Location      *time.Location // 分组与日期解释用的时区，nil 表示 time.Local
	Now           time.Time      // 计算"今天"用的时刻，零值表示 time.Now()
	Warn          io.Writer      // 警告输出，可空
}

// Period 是一个时段（一天或一个 ISO 周）的统计。
type Period struct {
	Date time.Time // 时段起点：按天是当天，按周是周一
	Delta
	Code int // 时段末的代码总量（不含未提交改动）
}

// Result 是一次历史统计的完整结果。
type Result struct {
	ProjectName     string
	Root            string
	Weekly          bool
	Since, Until    time.Time
	Periods         []Period
	Uncommitted     Delta
	UncommittedCode int
	Total           Delta
	TotalCode       int
	HeadCode        int
}

// Collect 读取 git 历史，按时段汇总新增/删除，并给出每个时段末的代码总量。
func Collect(opts Options) (*Result, error) {
	loc := opts.Location
	if loc == nil {
		loc = time.Local
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	rp, err := detectRepo(opts.Root)
	if err != nil {
		return nil, err
	}
	g := &gitRunner{dir: rp.root}
	cat, err := startCatFile(rp.root)
	if err != nil {
		return nil, err
	}
	defer cat.close()

	c := &collector{
		git: g, cat: cat, repo: rp, loc: loc, noIgnore: opts.NoIgnore, warn: opts.Warn,
		scope: newScope(opts.Root, rp.prefix, opts.AddonsDir, opts.ExcludeDirs, opts.ExcludeAddons),
		cache: map[string]*fileContent{}, warned: map[string]bool{}, head: g.hasHead(),
	}

	stats, earliest, err := c.commits()
	if err != nil {
		return nil, err
	}
	headCode, err := c.headCode()
	if err != nil {
		return nil, err
	}
	uncommitted, err := c.uncommitted()
	if err != nil {
		return nil, err
	}
	if c.fatal != nil {
		return nil, c.fatal
	}

	since, until, err := resolveRange(opts, earliest, loc, now)
	if err != nil {
		return nil, err
	}
	return build(opts, stats, headCode, uncommitted, since, until), nil
}

// resolveRange 决定统计范围：只给一端时，另一端分别取最早提交与今天。
func resolveRange(opts Options, earliest time.Time, loc *time.Location, now time.Time) (time.Time, time.Time, error) {
	since, until := opts.Since, opts.Until
	if !since.IsZero() {
		since = dateOnly(since.In(loc))
	}
	if !until.IsZero() {
		until = dateOnly(until.In(loc))
	}
	today := dateOnly(now.In(loc))

	switch {
	case since.IsZero() && until.IsZero():
		until = today
		if opts.Weekly {
			since = mondayOf(today).AddDate(0, 0, -11*7)
		} else {
			since = today.AddDate(0, 0, -13)
		}
	case since.IsZero():
		since = earliest
		if since.IsZero() {
			since = until // 仓库还没有提交：退化成单日
		}
	case until.IsZero():
		until = today
	}
	if since.After(until) {
		return since, until, fmt.Errorf("%w: --since %s is later than --until %s",
			ErrUsage, since.Format(dateLayout), until.Format(dateLayout))
	}
	return since, until, nil
}

// build 生成时段列表、各时段的增量与期末代码总量。
func build(opts Options, stats []commitStat, headCode int, uncommitted Delta, since, until time.Time) *Result {
	res := &Result{
		Root: opts.Root, Weekly: opts.Weekly, Since: since, Until: until,
		HeadCode: headCode, Uncommitted: uncommitted,
	}

	// 范围内没有提交的时段也要占一行，因此先生成完整的时段起点列表。
	var starts []time.Time
	if opts.Weekly {
		for d := mondayOf(since); !d.After(until); d = d.AddDate(0, 0, 7) {
			starts = append(starts, d)
		}
	} else {
		for d := since; !d.After(until); d = d.AddDate(0, 0, 1) {
			starts = append(starts, d)
		}
	}
	index := make(map[string]int, len(starts))
	periods := make([]Period, len(starts))
	for i, d := range starts {
		periods[i].Date = d
		index[d.Format(dateLayout)] = i
	}

	// 逐提交归桶：作者日期不单调也不影响结果。日期用字符串比较，避免夏令时干扰。
	sinceKey, untilKey := since.Format(dateLayout), until.Format(dateLayout)
	for _, s := range stats {
		key := s.date.Format(dateLayout)
		if key < sinceKey || key > untilKey {
			continue
		}
		if opts.Weekly {
			key = mondayOf(s.date).Format(dateLayout)
		}
		i, ok := index[key]
		if !ok {
			continue
		}
		periods[i].Delta.add(s.delta)
	}

	// 时段末代码总量 = HEAD 树总量 − 作者日期晚于该时段末的提交的净变化。
	sorted := append([]commitStat(nil), stats...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].date.Before(sorted[j].date) })
	suffix := make([]int, len(sorted)+1)
	for i := len(sorted) - 1; i >= 0; i-- {
		suffix[i] = suffix[i+1] + sorted[i].delta.NetCode()
	}
	netAfter := func(end time.Time) int {
		key := end.Format(dateLayout)
		i := sort.Search(len(sorted), func(i int) bool { return sorted[i].date.Format(dateLayout) > key })
		return suffix[i]
	}
	for i := range periods {
		end := periods[i].Date
		if opts.Weekly {
			end = end.AddDate(0, 0, 6)
		}
		periods[i].Code = headCode - netAfter(end)
	}

	res.Periods = periods
	for _, p := range periods {
		res.Total.add(p.Delta)
	}
	if len(periods) > 0 {
		res.TotalCode = periods[len(periods)-1].Code
	}
	// (uncommitted) 的 Code 是当前工作区的代码总量。
	res.UncommittedCode = headCode + uncommitted.NetCode()
	return res
}

// dateOnly 取某时刻所在时区的当天零点。
func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// mondayOf 返回该时刻所在 ISO 周的周一（周为一周之始）。
func mondayOf(d time.Time) time.Time {
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}
