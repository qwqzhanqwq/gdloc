// Package counter 提供各语言的逐行计数器。
// 计数器只接收文本内容，不读磁盘，便于复用 tscn 中提取的内嵌代码。
package counter

import "strings"

// Result 是一次计数的汇总结果。Doc 是 Comments 的子集。
type Result struct {
	Lines    int
	Code     int
	Comments int
	Doc      int
	Blanks   int
}

// LineKind 是单行的归类。
type LineKind int

const (
	// LineBlank 空行。
	LineBlank LineKind = iota
	// LineCode 代码行。
	LineCode
	// LineComment 普通注释行。
	LineComment
	// LineDoc 文档注释行。
	LineDoc
)

// Line 是逐行扫描结果。Code 为屏蔽字符串与注释后的代码文本；
// Comment 为原始注释文本（含注释符号），仅注释/文档行有意义；Indent 为前导空白字符数。
type Line struct {
	Kind    LineKind
	Code    string
	Comment string
	Indent  int
}

// CountLines 只统计文本的总行数（用于 Scene/Resource），规则同 CountGDScript 的行数定义。
func CountLines(text string) int {
	return len(splitLines(strings.TrimPrefix(text, "\ufeff")))
}

// CountGDScript 统计 GDScript 文本，规则见 AGENTS.md 4.1、4.2。
func CountGDScript(text string) Result {
	return summarize(GDScriptLines(text))
}

// GDScriptLines 逐行扫描 GDScript，返回每行的归类与屏蔽字符串后的代码文本。
func GDScriptLines(text string) []Line {
	raw := splitLines(strings.TrimPrefix(text, "\ufeff"))
	out := make([]Line, len(raw))
	inTriple := false
	var quote byte
	for i, line := range raw {
		if strings.TrimSpace(line) == "" {
			out[i] = Line{Kind: LineBlank, Indent: indentWidth(line)}
			continue
		}
		code, doc, masked, comment, ni, nq := scanGDScriptLine(line, inTriple, quote)
		inTriple, quote = ni, nq
		kind := LineComment
		if code {
			kind = LineCode
		} else if doc {
			kind = LineDoc
		}
		out[i] = Line{Kind: kind, Code: masked, Comment: comment, Indent: indentWidth(line)}
	}
	return out
}

func summarize(lines []Line) Result {
	var r Result
	r.Lines = len(lines)
	for _, ln := range lines {
		switch ln.Kind {
		case LineBlank:
			r.Blanks++
		case LineCode:
			r.Code++
		case LineDoc:
			r.Comments++
			r.Doc++
		default:
			r.Comments++
		}
	}
	return r
}

// scanGDScriptLine 扫描单行，返回是否含代码、是否文档注释、屏蔽后的代码、注释原文，以及行末三引号状态。
func scanGDScriptLine(line string, inTriple bool, tripleQuote byte) (code, doc bool, masked, comment string, nextInTriple bool, nextQuote byte) {
	out := make([]byte, len(line))
	for i := range out {
		out[i] = ' '
	}
	nextInTriple, nextQuote = inTriple, tripleQuote
	n := len(line)
	for i := 0; i < n; {
		if nextInTriple {
			code = true
			c := line[i]
			if c == '\\' {
				i += 2
				continue
			}
			if i+3 <= n && line[i] == nextQuote && line[i+1] == nextQuote && line[i+2] == nextQuote {
				nextInTriple = false
				i += 3
				continue
			}
			i++
			continue
		}
		switch c := line[i]; {
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case c == '#':
			if !code && i+1 < n && line[i+1] == '#' {
				doc = true
			}
			comment = line[i:]
			i = n
		case c == '"' || c == '\'':
			code = true
			if i+2 < n && line[i+1] == c && line[i+2] == c {
				nextInTriple = true
				nextQuote = c
				i += 3
				continue
			}
			i++
			for i < n {
				if line[i] == '\\' {
					i += 2
					continue
				}
				if line[i] == c {
					i++
					break
				}
				i++
			}
		default:
			out[i] = line[i]
			code = true
			i++
		}
	}
	return code, doc, string(out), comment, nextInTriple, nextQuote
}

// indentWidth 统计前导空格与制表符数量。
func indentWidth(line string) int {
	n := 0
	for n < len(line) && (line[n] == ' ' || line[n] == '\t') {
		n++
	}
	return n
}

// splitLines 按 \n 切分并兼容 CRLF；末尾单个换行不产生额外一行，空文本为 0 行。
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	if strings.HasSuffix(text, "\n") {
		text = text[:len(text)-1]
	}
	return strings.Split(text, "\n")
}
