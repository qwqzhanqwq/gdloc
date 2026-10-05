// Package report 汇总扫描结果，负责排序、表格与 JSON 输出。
package report

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"gdloc/internal/counter"
	"gdloc/internal/scan"
)

// FileStat 是单个已统计文件的结果。
type FileStat struct {
	Path     string
	Language string
	Result   counter.Result
}

// LangStat 是单个语言的汇总结果。
type LangStat struct {
	Language string
	Files    int
	Result   counter.Result
}

// Report 是完整的汇总结果。
type Report struct {
	ProjectName string
	Root        string
	Languages   []LangStat
	Total       LangStat
	Files       []FileStat
	CSharpFiles int
}

// Build 读取并统计各文件的代码行数，读取失败时写入 warn 并继续。
// 目前只有 GDScript 与 Shader 有计数器；C# 只计数文件数。
func Build(root string, entries []scan.FileEntry, warn io.Writer) Report {
	rep := Report{Root: root}
	langs := map[string]*LangStat{}

	add := func(language string, r counter.Result) {
		ls := langs[language]
		if ls == nil {
			ls = &LangStat{Language: language}
			langs[language] = ls
		}
		ls.Files++
		ls.Result.Lines += r.Lines
		ls.Result.Code += r.Code
		ls.Result.Comments += r.Comments
		ls.Result.Doc += r.Doc
		ls.Result.Blanks += r.Blanks
	}

	for _, e := range entries {
		var language string
		var count func(string) counter.Result
		switch e.Type {
		case scan.TypeGDScript:
			language, count = "GDScript", counter.CountGDScript
		case scan.TypeShader:
			language, count = "Shader", counter.CountShader
		case scan.TypeCSharp:
			rep.CSharpFiles++
			continue
		default:
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.Path)))
		if err != nil {
			if warn != nil {
				fmt.Fprintf(warn, "warning: cannot read %q: %v\n", e.Path, err)
			}
			continue
		}
		r := count(string(data))
		add(language, r)
		rep.Files = append(rep.Files, FileStat{Path: e.Path, Language: language, Result: r})
	}

	for _, ls := range langs {
		rep.Languages = append(rep.Languages, *ls)
	}
	sort.Slice(rep.Languages, func(i, j int) bool { return rep.Languages[i].Language < rep.Languages[j].Language })
	sort.Slice(rep.Files, func(i, j int) bool { return rep.Files[i].Path < rep.Files[j].Path })

	for _, ls := range rep.Languages {
		rep.Total.Files += ls.Files
		rep.Total.Result.Lines += ls.Result.Lines
		rep.Total.Result.Code += ls.Result.Code
		rep.Total.Result.Comments += ls.Result.Comments
		rep.Total.Result.Doc += ls.Result.Doc
		rep.Total.Result.Blanks += ls.Result.Blanks
	}
	rep.Total.Language = "Total"
	return rep
}

// ValidSortKey 判断排序键是否合法。
func ValidSortKey(key string) bool {
	switch key {
	case "code", "comments", "blanks", "lines", "files":
		return true
	default:
		return false
	}
}

// SortBy 按 key 降序排列语言与文件，同值按名称/路径升序，保证稳定。
func (r *Report) SortBy(key string) {
	sort.Slice(r.Languages, func(i, j int) bool {
		a, b := langMetric(r.Languages[i], key), langMetric(r.Languages[j], key)
		if a != b {
			return a > b
		}
		return r.Languages[i].Language < r.Languages[j].Language
	})
	sort.Slice(r.Files, func(i, j int) bool {
		a, b := fileMetric(r.Files[i], key), fileMetric(r.Files[j], key)
		if a != b {
			return a > b
		}
		return r.Files[i].Path < r.Files[j].Path
	})
}

func langMetric(l LangStat, key string) int {
	switch key {
	case "lines":
		return l.Result.Lines
	case "comments":
		return l.Result.Comments
	case "blanks":
		return l.Result.Blanks
	case "files":
		return l.Files
	default:
		return l.Result.Code
	}
}

func fileMetric(f FileStat, key string) int {
	switch key {
	case "comments":
		return f.Result.Comments
	case "blanks":
		return f.Result.Blanks
	// 文件维度没有"文件数"，files 键退回按总行数排序。
	case "lines", "files":
		return f.Result.Lines
	default:
		return f.Result.Code
	}
}
