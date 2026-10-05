// Package report 汇总扫描结果，负责排序、表格与 JSON 输出。
package report

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/qwqzhanqwq/gdloc/internal/counter"
	"github.com/qwqzhanqwq/gdloc/internal/godot"
	"github.com/qwqzhanqwq/gdloc/internal/scan"
	"github.com/qwqzhanqwq/gdloc/internal/stats"
)

// FileStat 是单个已统计单元的结果；内嵌代码的 Path 形如 "scene.tscn::id"。
type FileStat struct {
	Path     string
	Language string
	Result   counter.Result
}

// LangStat 是单个语言的汇总结果。Blocks 仅对内嵌语言有意义。
type LangStat struct {
	Language string
	Files    int
	Blocks   int
	Result   counter.Result
}

// GroupStat 是 --by-addon / --by-dir 的一行；Version/Dir/HasPluginCfg 仅插件分组使用。
type GroupStat struct {
	Name         string
	Version      string
	Dir          string
	HasPluginCfg bool
	Files        int
	Result       counter.Result
}

// Report 是完整的汇总结果。
type Report struct {
	ProjectName   string
	Root          string
	Languages     []LangStat // 代码类语言（含内嵌）
	Total         LangStat   // 只汇总代码类语言
	Scenes        LangStat   // 分隔线下方，仅 Files 与 Lines 有意义
	Resources     LangStat
	Groups        []GroupStat // --by-addon / --by-dir 分组结果
	Files         []FileStat
	CSharpFiles   int
	VisualShaders int
	Stats         *stats.Stats // --stats 视图
}

// Build 读取并统计各文件的代码行数，读取失败时写入 warn 并继续。
// GDScript/Shader 单独统计；.tscn/.tres 统计总行数并提取内嵌 GDScript/Shader；C# 只计数文件数。
// wantStats 为真时额外做进阶统计（top 为最长列表条数）。
func Build(root string, entries []scan.FileEntry, warn io.Writer, wantStats bool, statsTop int) Report {
	rep := Report{
		Root:      root,
		Scenes:    LangStat{Language: "Scene"},
		Resources: LangStat{Language: "Resource"},
	}
	langs := map[string]*LangStat{}
	embeddedParents := map[string]map[string]bool{}
	var units []stats.Unit

	read := func(e scan.FileEntry) (string, bool) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.Path)))
		if err != nil {
			if warn != nil {
				fmt.Fprintf(warn, "warning: cannot read %q: %v\n", e.Path, err)
			}
			return "", false
		}
		return string(data), true
	}
	addResult := func(language string, r counter.Result) *LangStat {
		ls := langs[language]
		if ls == nil {
			ls = &LangStat{Language: language}
			langs[language] = ls
		}
		ls.Result.Lines += r.Lines
		ls.Result.Code += r.Code
		ls.Result.Comments += r.Comments
		ls.Result.Doc += r.Doc
		ls.Result.Blanks += r.Blanks
		return ls
	}
	addEmbedded := func(language, parent string, r counter.Result) {
		ls := addResult(language, r)
		ls.Blocks++
		set := embeddedParents[language]
		if set == nil {
			set = map[string]bool{}
			embeddedParents[language] = set
		}
		set[parent] = true
	}

	for _, e := range entries {
		switch e.Type {
		case scan.TypeGDScript:
			text, ok := read(e)
			if !ok {
				continue
			}
			r := counter.CountGDScript(text)
			addResult("GDScript", r).Files++
			rep.Files = append(rep.Files, FileStat{Path: e.Path, Language: "GDScript", Result: r})
			if wantStats {
				units = append(units, stats.Unit{Path: e.Path, Language: "GDScript", Text: text})
			}
		case scan.TypeShader:
			text, ok := read(e)
			if !ok {
				continue
			}
			r := counter.CountShader(text)
			addResult("Shader", r).Files++
			rep.Files = append(rep.Files, FileStat{Path: e.Path, Language: "Shader", Result: r})
			if wantStats {
				units = append(units, stats.Unit{Path: e.Path, Language: "Shader", Text: text})
			}
		case scan.TypeCSharp:
			rep.CSharpFiles++
		case scan.TypeScene, scan.TypeResource:
			text, ok := read(e)
			if !ok {
				continue
			}
			target := &rep.Scenes
			if e.Type == scan.TypeResource {
				target = &rep.Resources
			}
			target.Files++
			target.Result.Lines += counter.CountLines(text)

			blocks, visualShaders := godot.ParseResource(text)
			rep.VisualShaders += visualShaders
			for _, b := range blocks {
				var r counter.Result
				var language string
				switch b.Language {
				case "GDScript":
					r = counter.CountGDScript(b.Source)
					language = "GDScript (embedded)"
				case "Shader":
					r = counter.CountShader(b.Source)
					language = "Shader (embedded)"
				default:
					continue
				}
				addEmbedded(language, e.Path, r)
				rep.Files = append(rep.Files, FileStat{Path: e.Path + "::" + b.ID, Language: language, Result: r})
				if wantStats {
					units = append(units, stats.Unit{Path: e.Path + "::" + b.ID, Language: b.Language, Text: b.Source})
				}
			}
		}
	}

	for language, set := range embeddedParents {
		if ls := langs[language]; ls != nil {
			ls.Files = len(set)
		}
	}
	for _, ls := range langs {
		rep.Languages = append(rep.Languages, *ls)
	}
	sort.Slice(rep.Languages, func(i, j int) bool { return rep.Languages[i].Language < rep.Languages[j].Language })
	sort.Slice(rep.Files, func(i, j int) bool { return rep.Files[i].Path < rep.Files[j].Path })

	for _, ls := range rep.Languages {
		rep.Total.Files += ls.Files
		rep.Total.Blocks += ls.Blocks
		rep.Total.Result.Lines += ls.Result.Lines
		rep.Total.Result.Code += ls.Result.Code
		rep.Total.Result.Comments += ls.Result.Comments
		rep.Total.Result.Doc += ls.Result.Doc
		rep.Total.Result.Blanks += ls.Result.Blanks
	}
	rep.Total.Language = "Total"
	if wantStats {
		s := stats.Analyze(units, statsTop)
		rep.Stats = &s
	}
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
	sort.Slice(r.Groups, func(i, j int) bool {
		a, b := groupMetric(r.Groups[i], key), groupMetric(r.Groups[j], key)
		if a != b {
			return a > b
		}
		return r.Groups[i].Name < r.Groups[j].Name
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

func groupMetric(g GroupStat, key string) int {
	switch key {
	case "lines":
		return g.Result.Lines
	case "comments":
		return g.Result.Comments
	case "blanks":
		return g.Result.Blanks
	case "files":
		return g.Files
	default:
		return g.Result.Code
	}
}
