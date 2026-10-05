package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
)

// WriteTable 输出表格；byFile 时按文件列出，top>0 时截断显示行数（Total 仍按全部计算）。
func WriteTable(w io.Writer, rep Report, byFile bool, top int) {
	if rep.ProjectName != "" {
		fmt.Fprintf(w, "Project: %s\n", rep.ProjectName)
		fmt.Fprintf(w, "Root:    %s\n", rep.Root)
	}
	if byFile {
		writeFileTable(w, rep, top)
	} else {
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
	rows = append(rows, langRow(rep.Total))
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
	widths := columnWidths(all)
	writeLines(w, all, widths, 2)
}

func langRow(ls LangStat) []string {
	return []string{
		ls.Language, comma(ls.Files), comma(ls.Result.Lines), comma(ls.Result.Code),
		comma(ls.Result.Comments), comma(ls.Result.Doc), comma(ls.Result.Blanks),
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

type jsonReport struct {
	ProjectName   string      `json:"project_name"`
	Root          string      `json:"root"`
	Languages     []jsonLang  `json:"languages"`
	Total         jsonLang    `json:"total"`
	Scenes        jsonSection `json:"scenes"`
	Resources     jsonSection `json:"resources"`
	VisualShaders int         `json:"visual_shaders"`
	Files         []jsonFile  `json:"files,omitempty"`
}

func toJSONLang(ls LangStat) jsonLang {
	return jsonLang{
		Language: ls.Language, Files: ls.Files, Blocks: ls.Blocks, Lines: ls.Result.Lines,
		Code: ls.Result.Code, Comments: ls.Result.Comments, Doc: ls.Result.Doc, Blanks: ls.Result.Blanks,
	}
}

// WriteJSON 以 JSON 输出；byFile 时额外包含 files，top>0 时截断数组但 total 保持全量。
func WriteJSON(w io.Writer, rep Report, byFile bool, top int) {
	out := jsonReport{
		ProjectName: rep.ProjectName,
		Root:        rep.Root,
		Scenes:      jsonSection{Files: rep.Scenes.Files, Lines: rep.Scenes.Result.Lines},
		Resources:   jsonSection{Files: rep.Resources.Files, Lines: rep.Resources.Result.Lines},
		Total:       toJSONLang(rep.Total),
	}
	out.Languages = []jsonLang{}
	for i, ls := range rep.Languages {
		if top > 0 && i >= top {
			break
		}
		out.Languages = append(out.Languages, toJSONLang(ls))
	}
	out.VisualShaders = rep.VisualShaders
	if byFile {
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
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
