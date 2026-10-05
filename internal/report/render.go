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
	renderRows(w, header, rows, 1)
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
	renderRows(w, header, rows, 2)
}

func langRow(ls LangStat) []string {
	return []string{
		ls.Language, comma(ls.Files), comma(ls.Result.Lines), comma(ls.Result.Code),
		comma(ls.Result.Comments), comma(ls.Result.Doc), comma(ls.Result.Blanks),
	}
}

// renderRows 按列宽预对齐后再交给 tabwriter 拼接：numericFrom 之前的列左对齐，其余右对齐。
func renderRows(w io.Writer, header []string, rows [][]string, numericFrom int) {
	widths := make([]int, len(header))
	for i, h := range header {
		widths[i] = len(h)
	}
	for _, r := range rows {
		for i, c := range r {
			if len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	write := func(cells []string) {
		pad := make([]string, len(cells))
		for i, c := range cells {
			if i >= numericFrom {
				pad[i] = fmt.Sprintf("%*s", widths[i], c)
			} else {
				pad[i] = fmt.Sprintf("%-*s", widths[i], c)
			}
		}
		fmt.Fprintln(tw, strings.Join(pad, "\t"))
	}
	write(header)
	for _, r := range rows {
		write(r)
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

type jsonReport struct {
	ProjectName string     `json:"project_name"`
	Root        string     `json:"root"`
	Languages   []jsonLang `json:"languages"`
	Total       jsonLang   `json:"total"`
	Files       []jsonFile `json:"files,omitempty"`
}

// WriteJSON 以 JSON 输出；byFile 时额外包含 files，top>0 时截断数组但 total 保持全量。
func WriteJSON(w io.Writer, rep Report, byFile bool, top int) {
	out := jsonReport{ProjectName: rep.ProjectName, Root: rep.Root}
	for i, ls := range rep.Languages {
		if top > 0 && i >= top {
			break
		}
		out.Languages = append(out.Languages, jsonLang{
			Language: ls.Language, Files: ls.Files, Lines: ls.Result.Lines, Code: ls.Result.Code,
			Comments: ls.Result.Comments, Doc: ls.Result.Doc, Blanks: ls.Result.Blanks,
		})
	}
	if out.Languages == nil {
		out.Languages = []jsonLang{}
	}
	t := rep.Total
	out.Total = jsonLang{
		Language: t.Language, Files: t.Files, Lines: t.Result.Lines, Code: t.Result.Code,
		Comments: t.Result.Comments, Doc: t.Result.Doc, Blanks: t.Result.Blanks,
	}
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
