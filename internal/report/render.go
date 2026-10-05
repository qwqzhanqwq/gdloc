package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"gdloc/internal/counter"
)

// Mode 决定输出形态。
type Mode int

const (
	// ModeLanguage 按语言汇总（默认）。
	ModeLanguage Mode = iota
	// ModeFile 按文件列出（含内嵌块）。
	ModeFile
	// ModeAddon 按插件分组。
	ModeAddon
	// ModeDir 按顶层目录分组。
	ModeDir
	// ModeStats 进阶统计。
	ModeStats
)

// WriteTable 输出表格；top>0 时截断显示行数（Total 仍按全部计算）。
func WriteTable(w io.Writer, rep Report, mode Mode, top int) {
	if rep.ProjectName != "" {
		fmt.Fprintf(w, "Project: %s\n", rep.ProjectName)
		fmt.Fprintf(w, "Root:    %s\n", rep.Root)
	}
	switch mode {
	case ModeFile:
		writeFileTable(w, rep, top)
	case ModeAddon:
		writeAddonTable(w, rep, top)
	case ModeDir:
		writeDirTable(w, rep, top)
	case ModeStats:
		writeStatsTable(w, rep)
	default:
		writeLangTable(w, rep, top)
	}
	if rep.VisualShaders > 0 {
		fmt.Fprintf(w, "Note: %d VisualShader resource(s) not counted.\n", rep.VisualShaders)
	}
	if rep.CSharpFiles > 0 {
		fmt.Fprintf(w, "Note: %d C# file(s) not counted.\n", rep.CSharpFiles)
	}
}

func writeLangTable(w io.Writer, rep Report, top int) {
	header := []string{"Language", "Files", "Lines", "Code", "Comments", "Doc", "Blanks"}
	var rows [][]string
	for i, ls := range rep.Languages {
		if top > 0 && i >= top {
			break
		}
		rows = append(rows, langRow(ls))
	}
	rows = append(rows, totalRow(rep.Total.Language, rep.Total.Files, rep.Total.Result))

	var tail [][]string
	if rep.Scenes.Files > 0 {
		tail = append(tail, sectionRow(rep.Scenes))
	}
	if rep.Resources.Files > 0 {
		tail = append(tail, sectionRow(rep.Resources))
	}

	widths := columnWidths(append(append([][]string{header}, rows...), tail...))
	writeLines(w, append([][]string{header}, rows...), widths, 1)
	if len(tail) > 0 {
		fmt.Fprintln(w, strings.Repeat("-", totalWidth(widths)))
		writeLines(w, tail, widths, 1)
	}
}

func writeFileTable(w io.Writer, rep Report, top int) {
	header := []string{"Path", "Language", "Lines", "Code", "Comments", "Doc", "Blanks"}
	var rows [][]string
	for i, f := range rep.Files {
		if top > 0 && i >= top {
			break
		}
		rows = append(rows, []string{
			f.Path, f.Language, comma(f.Result.Lines), comma(f.Result.Code),
			comma(f.Result.Comments), comma(f.Result.Doc), comma(f.Result.Blanks),
		})
	}
	all := append([][]string{header}, rows...)
	writeLines(w, all, columnWidths(all), 2)
}

func writeAddonTable(w io.Writer, rep Report, top int) {
	header := []string{"Addon", "Version", "Files", "Lines", "Code", "Comments", "Doc", "Blanks"}
	var rows [][]string
	for i, g := range rep.Groups {
		if top > 0 && i >= top {
			break
		}
		rows = append(rows, []string{
			g.Name, g.Version, comma(g.Files), comma(g.Result.Lines), comma(g.Result.Code),
			comma(g.Result.Comments), comma(g.Result.Doc), comma(g.Result.Blanks),
		})
	}
	rows = append(rows, []string{
		rep.Total.Language, "", comma(rep.Total.Files), comma(rep.Total.Result.Lines), comma(rep.Total.Result.Code),
		comma(rep.Total.Result.Comments), comma(rep.Total.Result.Doc), comma(rep.Total.Result.Blanks),
	})
	all := append([][]string{header}, rows...)
	writeLines(w, all, columnWidths(all), 2)
}

func writeDirTable(w io.Writer, rep Report, top int) {
	header := []string{"Group", "Files", "Lines", "Code", "Comments", "Doc", "Blanks"}
	var rows [][]string
	for i, g := range rep.Groups {
		if top > 0 && i >= top {
			break
		}
		rows = append(rows, []string{
			g.Name, comma(g.Files), comma(g.Result.Lines), comma(g.Result.Code),
			comma(g.Result.Comments), comma(g.Result.Doc), comma(g.Result.Blanks),
		})
	}
	rows = append(rows, totalRow(rep.Total.Language, rep.Total.Files, rep.Total.Result))
	all := append([][]string{header}, rows...)
	writeLines(w, all, columnWidths(all), 1)
}

func writeStatsTable(w io.Writer, rep Report) {
	if rep.Stats == nil {
		return
	}
	st := rep.Stats

	fmt.Fprintln(w, "Comment analysis (suspected commented-out code is a heuristic estimate):")
	rows := [][]string{{"Language", "Comments", "Suspected", "Ratio"}}
	for _, l := range st.Languages {
		rows = append(rows, []string{l.Language, comma(l.Comments), comma(l.Suspected), fmt.Sprintf("%.1f%%", l.Ratio()*100)})
	}
	writeLines(w, rows, columnWidths(rows), 1)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "GDScript structure:")
	fmt.Fprintf(w, "funcs: %d  signals: %d  class_names: %d  exports: %d\n\n",
		st.Structure.Funcs, st.Structure.Signals, st.Structure.ClassNames, st.Structure.Exports)

	fmt.Fprintln(w, "Longest files by code lines:")
	frows := [][]string{{"Path", "Language", "Code"}}
	for _, f := range st.LongestFiles {
		frows = append(frows, []string{f.Path, f.Language, comma(f.Code)})
	}
	writeLines(w, frows, columnWidths(frows), 2)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Longest GDScript functions:")
	fns := [][]string{{"Location", "Function", "Lines"}}
	for _, f := range st.LongestFuncs {
		fns = append(fns, []string{fmt.Sprintf("%s:%d", f.Path, f.Line), f.Name, comma(f.Length)})
	}
	writeLines(w, fns, columnWidths(fns), 2)
}

func langRow(ls LangStat) []string {
	return totalRow(ls.Language, ls.Files, ls.Result)
}

func totalRow(name string, files int, r counter.Result) []string {
	return []string{
		name, comma(files), comma(r.Lines), comma(r.Code),
		comma(r.Comments), comma(r.Doc), comma(r.Blanks),
	}
}

// sectionRow 只填 Language / Files / Lines，其余列留空。
func sectionRow(ls LangStat) []string {
	return []string{ls.Language, comma(ls.Files), comma(ls.Result.Lines), "", "", "", ""}
}

func columnWidths(rows [][]string) []int {
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, c := range row {
			if len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}
	return widths
}

func totalWidth(widths []int) int {
	sum := 0
	for _, w := range widths {
		sum += w
	}
	return sum + 2*(len(widths)-1)
}

// writeLines 预对齐后交给 tabwriter 拼接：numericFrom 之前的列左对齐，其余右对齐。
func writeLines(w io.Writer, rows [][]string, widths []int, numericFrom int) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		cells := make([]string, len(row))
		for i, c := range row {
			if i >= numericFrom {
				cells[i] = fmt.Sprintf("%*s", widths[i], c)
			} else {
				cells[i] = fmt.Sprintf("%-*s", widths[i], c)
			}
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	tw.Flush()
}

// comma 为数字添加千位分隔符。
func comma(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var groups []string
	for len(s) > 3 {
		groups = append([]string{s[len(s)-3:]}, groups...)
		s = s[:len(s)-3]
	}
	groups = append([]string{s}, groups...)
	out := strings.Join(groups, ",")
	if neg {
		return "-" + out
	}
	return out
}

type jsonLang struct {
	Language string `json:"language"`
	Files    int    `json:"files"`
	Blocks   int    `json:"blocks"`
	Lines    int    `json:"lines"`
	Code     int    `json:"code"`
	Comments int    `json:"comments"`
	Doc      int    `json:"doc"`
	Blanks   int    `json:"blanks"`
}

type jsonFile struct {
	Path     string `json:"path"`
	Language string `json:"language"`
	Lines    int    `json:"lines"`
	Code     int    `json:"code"`
	Comments int    `json:"comments"`
	Doc      int    `json:"doc"`
	Blanks   int    `json:"blanks"`
}

type jsonSection struct {
	Files int `json:"files"`
	Lines int `json:"lines"`
}

type jsonStats struct {
	Languages        []jsonStatsLang `json:"languages"`
	Structure        jsonStatsStruct `json:"structure"`
	LongestFiles     []jsonStatsFile `json:"longest_files"`
	LongestFunctions []jsonStatsFunc `json:"longest_functions"`
}

type jsonStatsLang struct {
	Language  string  `json:"language"`
	Comments  int     `json:"comments"`
	Suspected int     `json:"suspected"`
	Ratio     float64 `json:"ratio"`
}

type jsonStatsStruct struct {
	Funcs      int `json:"funcs"`
	Signals    int `json:"signals"`
	ClassNames int `json:"class_names"`
	Exports    int `json:"exports"`
}

type jsonStatsFile struct {
	Path     string `json:"path"`
	Language string `json:"language"`
	Code     int    `json:"code"`
}

type jsonStatsFunc struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Name   string `json:"name"`
	Length int    `json:"length"`
}

type jsonAddon struct {
	Name         string `json:"name"`
	Dir          string `json:"dir"`
	Version      string `json:"version"`
	HasPluginCfg bool   `json:"has_plugin_cfg"`
	Files        int    `json:"files"`
	Lines        int    `json:"lines"`
	Code         int    `json:"code"`
	Comments     int    `json:"comments"`
	Doc          int    `json:"doc"`
	Blanks       int    `json:"blanks"`
}

type jsonDir struct {
	Name     string `json:"name"`
	Files    int    `json:"files"`
	Lines    int    `json:"lines"`
	Code     int    `json:"code"`
	Comments int    `json:"comments"`
	Doc      int    `json:"doc"`
	Blanks   int    `json:"blanks"`
}

type jsonReport struct {
	ProjectName   string      `json:"project_name"`
	Root          string      `json:"root"`
	Languages     []jsonLang  `json:"languages"`
	Total         jsonLang    `json:"total"`
	Scenes        jsonSection `json:"scenes"`
	Resources     jsonSection `json:"resources"`
	VisualShaders int         `json:"visual_shaders"`
	Addons        []jsonAddon `json:"addons,omitempty"`
	Dirs          []jsonDir   `json:"dirs,omitempty"`
	Files         []jsonFile  `json:"files,omitempty"`
	Stats         *jsonStats  `json:"stats,omitempty"`
}

func toJSONLang(ls LangStat) jsonLang {
	return jsonLang{
		Language: ls.Language, Files: ls.Files, Blocks: ls.Blocks, Lines: ls.Result.Lines,
		Code: ls.Result.Code, Comments: ls.Result.Comments, Doc: ls.Result.Doc, Blanks: ls.Result.Blanks,
	}
}

// WriteJSON 以 JSON 输出；ModeFile 时含 files，ModeAddon 时含 addons，ModeDir 时含 dirs。
// top>0 时截断数组但 total 保持全量。
func WriteJSON(w io.Writer, rep Report, mode Mode, top int) {
	out := jsonReport{
		ProjectName:   rep.ProjectName,
		Root:          rep.Root,
		Scenes:        jsonSection{Files: rep.Scenes.Files, Lines: rep.Scenes.Result.Lines},
		Resources:     jsonSection{Files: rep.Resources.Files, Lines: rep.Resources.Result.Lines},
		Total:         toJSONLang(rep.Total),
		VisualShaders: rep.VisualShaders,
	}
	out.Languages = []jsonLang{}
	for i, ls := range rep.Languages {
		if top > 0 && i >= top {
			break
		}
		out.Languages = append(out.Languages, toJSONLang(ls))
	}

	switch mode {
	case ModeFile:
		out.Files = []jsonFile{}
		for i, f := range rep.Files {
			if top > 0 && i >= top {
				break
			}
			out.Files = append(out.Files, jsonFile{
				Path: f.Path, Language: f.Language, Lines: f.Result.Lines, Code: f.Result.Code,
				Comments: f.Result.Comments, Doc: f.Result.Doc, Blanks: f.Result.Blanks,
			})
		}
	case ModeAddon:
		out.Addons = []jsonAddon{}
		for i, g := range rep.Groups {
			if top > 0 && i >= top {
				break
			}
			out.Addons = append(out.Addons, jsonAddon{
				Name: g.Name, Dir: g.Dir, Version: g.Version, HasPluginCfg: g.HasPluginCfg,
				Files: g.Files, Lines: g.Result.Lines, Code: g.Result.Code,
				Comments: g.Result.Comments, Doc: g.Result.Doc, Blanks: g.Result.Blanks,
			})
		}
	case ModeDir:
		out.Dirs = []jsonDir{}
		for i, g := range rep.Groups {
			if top > 0 && i >= top {
				break
			}
			out.Dirs = append(out.Dirs, jsonDir{
				Name: g.Name, Files: g.Files, Lines: g.Result.Lines, Code: g.Result.Code,
				Comments: g.Result.Comments, Doc: g.Result.Doc, Blanks: g.Result.Blanks,
			})
		}
	}

	if rep.Stats != nil {
		js := &jsonStats{
			Structure: jsonStatsStruct{
				Funcs: rep.Stats.Structure.Funcs, Signals: rep.Stats.Structure.Signals,
				ClassNames: rep.Stats.Structure.ClassNames, Exports: rep.Stats.Structure.Exports,
			},
			Languages:        []jsonStatsLang{},
			LongestFiles:     []jsonStatsFile{},
			LongestFunctions: []jsonStatsFunc{},
		}
		for _, l := range rep.Stats.Languages {
			js.Languages = append(js.Languages, jsonStatsLang{
				Language: l.Language, Comments: l.Comments, Suspected: l.Suspected, Ratio: l.Ratio(),
			})
		}
		for _, f := range rep.Stats.LongestFiles {
			js.LongestFiles = append(js.LongestFiles, jsonStatsFile{Path: f.Path, Language: f.Language, Code: f.Code})
		}
		for _, f := range rep.Stats.LongestFuncs {
			js.LongestFunctions = append(js.LongestFunctions, jsonStatsFunc{Path: f.Path, Line: f.Line, Name: f.Name, Length: f.Length})
		}
		out.Stats = js
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
