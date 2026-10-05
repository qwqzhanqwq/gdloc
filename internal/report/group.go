package report

import (
	"path/filepath"
	"sort"
	"strings"

	"gdloc/internal/godot"
)

// GroupByDir 按扫描根目录下的顶层目录分组，根目录下的文件归入 "(root)"。
func GroupByDir(rep Report) []GroupStat {
	return buildGroups(rep.Files, func(source string) GroupStat {
		if i := strings.IndexByte(source, '/'); i >= 0 {
			return GroupStat{Name: source[:i]}
		}
		return GroupStat{Name: "(root)"}
	}, "")
}

// GroupByAddon 按项目 addons/ 下的插件分组，其余文件归入 "(project)"。
func GroupByAddon(rep Report, root, addonsDir string, plugins []godot.Plugin) []GroupStat {
	type pinfo struct {
		rel string
		p   godot.Plugin
	}
	var infos []pinfo
	for _, p := range plugins {
		rel, err := filepath.Rel(root, p.Dir)
		if err != nil {
			continue
		}
		infos = append(infos, pinfo{rel: filepath.ToSlash(rel), p: p})
	}
	// 长前缀优先，避免误匹配。
	sort.Slice(infos, func(i, j int) bool { return len(infos[i].rel) > len(infos[j].rel) })

	return buildGroups(rep.Files, func(source string) GroupStat {
		for _, in := range infos {
			if in.rel == "." || in.rel == ".." || strings.HasPrefix(in.rel, "../") {
				continue
			}
			if source == in.rel || strings.HasPrefix(source, in.rel+"/") {
				return GroupStat{Name: in.p.Name, Version: in.p.Version, Dir: in.rel, HasPluginCfg: in.p.HasPluginCfg}
			}
		}
		return GroupStat{Name: "(project)"}
	}, "(project)")
}

// buildGroups 把每个已统计单元归入分组；Files 按 (语言, 源文件) 去重，
// 保证各分组 Files 之和等于 Total.Files。ensure 非空时确保该名称的分组存在。
func buildGroups(files []FileStat, keyFn func(source string) GroupStat, ensure string) []GroupStat {
	type agg struct {
		stat GroupStat
		seen map[string]bool
	}
	aggs := map[string]*agg{}
	for _, f := range files {
		source := f.Path
		if i := strings.Index(source, "::"); i >= 0 {
			source = source[:i]
		}
		k := keyFn(source)
		a := aggs[k.Name]
		if a == nil {
			a = &agg{stat: k, seen: map[string]bool{}}
			aggs[k.Name] = a
		}
		if dk := f.Language + "|" + source; !a.seen[dk] {
			a.seen[dk] = true
			a.stat.Files++
		}
		a.stat.Result.Lines += f.Result.Lines
		a.stat.Result.Code += f.Result.Code
		a.stat.Result.Comments += f.Result.Comments
		a.stat.Result.Doc += f.Result.Doc
		a.stat.Result.Blanks += f.Result.Blanks
	}
	if ensure != "" {
		if _, ok := aggs[ensure]; !ok {
			aggs[ensure] = &agg{stat: GroupStat{Name: ensure}, seen: map[string]bool{}}
		}
	}
	out := make([]GroupStat, 0, len(aggs))
	for _, a := range aggs {
		out = append(out, a.stat)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
