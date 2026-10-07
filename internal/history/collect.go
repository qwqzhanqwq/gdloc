package history

import (
	"io"
	"path"
	"time"

	"github.com/qwqzhanqwq/gdloc/internal/scan"
)

// commitStat 是一个提交对某个日期桶的贡献。
type commitStat struct {
	date  time.Time // 本地时区的作者日期
	delta Delta
}

// collector 持有一次统计的 git 句柄与缓存。
type collector struct {
	git      *gitRunner
	cat      *catFile
	scope    *scope
	repo     repo
	loc      *time.Location
	noIgnore bool
	warn     io.Writer
	cache    map[string]*fileContent
	warned   map[string]bool
	head     bool  // 仓库是否已有提交
	fatal    error // git 管道级错误：无法继续统计
}

// commits 读取 HEAD 可达的非合并提交，返回各提交的增量与最早的作者日期。
func (c *collector) commits() ([]commitStat, time.Time, error) {
	if !c.head {
		return nil, time.Time{}, nil
	}
	data, err := c.git.run("log", "--no-merges", "-M", "--no-abbrev", "--raw", "-z", "--format="+logFormat)
	if err != nil {
		return nil, time.Time{}, err
	}
	parsed, err := parseLog(data)
	if err != nil {
		return nil, time.Time{}, err
	}

	var out []commitStat
	var earliest time.Time
	for _, cm := range parsed {
		day := dateOnly(cm.date.In(c.loc))
		if earliest.IsZero() || day.Before(earliest) {
			earliest = day
		}
		d, touched := c.commitDelta(cm)
		if !touched {
			continue
		}
		d.Commits = 1
		out = append(out, commitStat{date: day, delta: d})
	}
	return out, earliest, nil
}

// commitDelta 汇总一个提交里所有在范围内的改动；touched 表示它是否算一次提交。
func (c *collector) commitDelta(cm commit) (Delta, bool) {
	var d Delta
	touched := false
	for _, ch := range cm.changes {
		if !regularFile(ch.oldMode) && !regularFile(ch.newMode) {
			continue // 软链接、子模块不参与
		}
		dd, ok := c.delta(sideOf(ch.oldPath, ch.oldSHA, ch.oldMode), sideOf(ch.newPath, ch.newSHA, ch.newMode))
		if !ok {
			continue
		}
		touched = true
		d.add(dd)
	}
	return d, touched
}

// headCode 按同一口径统计 HEAD 树的代码总量。
func (c *collector) headCode() (int, error) {
	if !c.head {
		return 0, nil
	}
	data, err := c.git.run("ls-tree", "-r", "-z", "HEAD")
	if err != nil {
		return 0, err
	}
	total := 0
	for _, e := range parseTree(data) {
		if !regularFile(e.mode) {
			continue
		}
		t := scan.Classify(path.Base(e.path))
		if !c.scope.accept(e.path, t) {
			continue
		}
		fc, ok := c.content(side{path: e.path, typ: t, present: true, sha: e.sha})
		if !ok {
			continue
		}
		total += codeCount(fc, t)
	}
	if c.fatal != nil {
		return 0, c.fatal
	}
	return total, nil
}

// uncommitted 统计工作区（含暂存区）相对 HEAD 的变化，未跟踪文件整份算新增。
func (c *collector) uncommitted() (Delta, error) {
	var d Delta
	if c.head {
		data, err := c.git.run("diff", "-M", "--no-abbrev", "--raw", "-z", "HEAD")
		if err != nil {
			return Delta{}, err
		}
		changes, err := parseRawEntries(data)
		if err != nil {
			return Delta{}, err
		}
		for _, ch := range changes {
			if !regularFile(ch.oldMode) && !regularFile(ch.newMode) {
				continue
			}
			newSide := sideOf(ch.newPath, ch.newSHA, ch.newMode)
			if newSide.present {
				// 工作区内容不是 git 对象，newSHA 恒为全零，直接从磁盘读。
				text, ok := c.worktreeText(ch.newPath)
				if !ok {
					continue
				}
				newSide.text, newSide.isText = text, true
			}
			dd, ok := c.delta(sideOf(ch.oldPath, ch.oldSHA, ch.oldMode), newSide)
			if !ok {
				continue
			}
			d.add(dd)
		}
	} else {
		// 还没有提交：索引与工作区里的文件全部算新增。
		data, err := c.git.run(c.listArgs("--cached", "--others")...)
		if err != nil {
			return Delta{}, err
		}
		d.add(c.newFiles(splitZ(data)))
	}
	// 未跟踪文件不在这两种来源里，单独列一遍；空仓库时上面已经全部列出。
	if !c.head {
		return d, nil
	}
	data, err := c.git.run(c.listArgs("--others")...)
	if err != nil {
		return Delta{}, err
	}
	d.add(c.newFiles(splitZ(data)))
	if c.fatal != nil {
		return Delta{}, c.fatal
	}
	return d, nil
}

// newFiles 把若干路径当作全新文件整份计入新增。
func (c *collector) newFiles(paths []string) Delta {
	var d Delta
	for _, p := range paths {
		t := scan.Classify(path.Base(p))
		if !c.scope.accept(p, t) {
			continue
		}
		text, ok := c.worktreeText(p)
		if !ok {
			continue
		}
		d.add(c.wholeFile(side{path: p, typ: t, present: true, text: text, isText: true}, true))
	}
	return d
}

// listArgs 拼出 ls-files 的参数；--no-ignore 时不再套用 .gitignore。
func (c *collector) listArgs(extra ...string) []string {
	args := append([]string{"ls-files", "-z"}, extra...)
	if !c.noIgnore {
		args = append(args, "--exclude-standard")
	}
	return args
}
