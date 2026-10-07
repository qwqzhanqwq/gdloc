package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/qwqzhanqwq/gdloc/internal/history"
)

// WriteHistoryTable 输出 --daily / --weekly 的表格；top>0 时只保留最近的 N 个时段，
// (uncommitted) 与 Total 始终显示。
func WriteHistoryTable(w io.Writer, res history.Result, top int) {
	if res.ProjectName != "" {
		fmt.Fprintf(w, "Project: %s\n", res.ProjectName)
		fmt.Fprintf(w, "Root:    %s\n", res.Root)
	}
	label := "Date"
	if res.Weekly {
		label = "Week"
	}
	rows := [][]string{{
		label, "Commits", "+Code", "-Code", "Net", "+Comments", "-Comments", "+Blanks", "-Blanks", "Code",
	}}
	periods := res.Periods
	if top > 0 && len(periods) > top {
		periods = periods[len(periods)-top:]
	}
	for _, p := range periods {
		rows = append(rows, deltaRow(p.Date.Format("2006-01-02"), p.Delta, p.Code))
	}
	rows = append(rows, deltaRow("(uncommitted)", res.Uncommitted, res.UncommittedCode))
	rows = append(rows, deltaRow("Total", res.Total, res.TotalCode))
	writeLines(w, rows, columnWidths(rows), 1)
}

func deltaRow(name string, d history.Delta, code int) []string {
	return []string{
		name, comma(d.Commits), comma(d.AddedCode), comma(d.DeletedCode), comma(d.NetCode()),
		comma(d.AddedComments), comma(d.DeletedComments), comma(d.AddedBlanks), comma(d.DeletedBlanks), comma(code),
	}
}

type jsonHistoryDelta struct {
	Commits         int `json:"commits"`
	AddedCode       int `json:"added_code"`
	DeletedCode     int `json:"deleted_code"`
	NetCode         int `json:"net_code"`
	AddedComments   int `json:"added_comments"`
	DeletedComments int `json:"deleted_comments"`
	AddedBlanks     int `json:"added_blanks"`
	DeletedBlanks   int `json:"deleted_blanks"`
	Code            int `json:"code"`
}

type jsonHistoryPeriod struct {
	Date string `json:"date"`
	jsonHistoryDelta
}

type jsonHistory struct {
	ProjectName string              `json:"project_name"`
	Root        string              `json:"root"`
	Mode        string              `json:"mode"`
	Since       string              `json:"since"`
	Until       string              `json:"until"`
	HeadCode    int                 `json:"head_code"`
	Periods     []jsonHistoryPeriod `json:"periods"`
	Uncommitted jsonHistoryDelta    `json:"uncommitted"`
	Total       jsonHistoryDelta    `json:"total"`
}

// WriteHistoryJSON 以 JSON 输出历史统计；top>0 时截断 periods，total 仍覆盖整个范围。
func WriteHistoryJSON(w io.Writer, res history.Result, top int) {
	mode := "daily"
	if res.Weekly {
		mode = "weekly"
	}
	out := jsonHistory{
		ProjectName: res.ProjectName,
		Root:        res.Root,
		Mode:        mode,
		Since:       res.Since.Format("2006-01-02"),
		Until:       res.Until.Format("2006-01-02"),
		HeadCode:    res.HeadCode,
		Periods:     []jsonHistoryPeriod{},
		Uncommitted: toJSONHistoryDelta(res.Uncommitted, res.UncommittedCode),
		Total:       toJSONHistoryDelta(res.Total, res.TotalCode),
	}
	periods := res.Periods
	if top > 0 && len(periods) > top {
		periods = periods[len(periods)-top:]
	}
	for _, p := range periods {
		out.Periods = append(out.Periods, jsonHistoryPeriod{
			Date:             p.Date.Format("2006-01-02"),
			jsonHistoryDelta: toJSONHistoryDelta(p.Delta, p.Code),
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func toJSONHistoryDelta(d history.Delta, code int) jsonHistoryDelta {
	return jsonHistoryDelta{
		Commits: d.Commits, AddedCode: d.AddedCode, DeletedCode: d.DeletedCode, NetCode: d.NetCode(),
		AddedComments: d.AddedComments, DeletedComments: d.DeletedComments,
		AddedBlanks: d.AddedBlanks, DeletedBlanks: d.DeletedBlanks, Code: code,
	}
}
