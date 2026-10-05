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

// CountLines 只统计文本的总行数（用于 Scene/Resource），规则同 CountGDScript 的行数定义。
func CountLines(text string) int {
	return len(splitLines(strings.TrimPrefix(text, "\ufeff")))
}

// CountGDScript 统计 GDScript 文本，规则见 AGENTS.md 4.1、4.2。
func CountGDScript(text string) Result {
	text = strings.TrimPrefix(text, "\ufeff")
	lines := splitLines(text)
	var r Result
	r.Lines = len(lines)

	// 跨行状态只有三引号多行字符串。
	inTriple := false
	var tripleQuote byte

	for _, line := range lines {
		// 空行（含只含 \r 的 CRLF 行）优先归为空行，且不改变三引号状态。
		if strings.TrimSpace(line) == "" {
			r.Blanks++
			continue
		}
		code, doc, inTriple2, q2 := scanLine(line, inTriple, tripleQuote)
		inTriple, tripleQuote = inTriple2, q2
		if code {
			r.Code++
			continue
		}
		r.Comments++
		if doc {
			r.Doc++
		}
	}
	return r
}

// scanLine 扫描单行，返回该行是否含代码、是否为文档注释，以及行末的三引号状态。
// 单/双引号串不跨行，行末即结束；只有三引号串会把状态带到下一行。
func scanLine(line string, inTriple bool, tripleQuote byte) (code, doc, nextInTriple bool, nextQuote byte) {
	nextInTriple, nextQuote = inTriple, tripleQuote
	n := len(line)
	for i := 0; i < n; {
		if nextInTriple {
			code = true
			c := line[i]
			// 反斜杠转义下一字符；raw 与非 raw 在字符串边界上等价，无需区分。
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
			// 行首（忽略缩进）为 ## 的注释行计为文档注释。
			if !code && i+1 < n && line[i+1] == '#' {
				doc = true
			}
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
			code = true
			i++
		}
	}
	return code, doc, nextInTriple, nextQuote
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
