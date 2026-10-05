package counter

import "strings"

// CountShader 统计 Godot Shader 文本，规则见 AGENTS.md 4.1、4.3。
// 跨行状态只有块注释（含文档块注释标记）。
func CountShader(text string) Result {
	text = strings.TrimPrefix(text, "\ufeff")
	lines := splitLines(text)
	var r Result
	r.Lines = len(lines)

	inBlock := false
	inDoc := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			r.Blanks++
			continue
		}
		code, doc, nb, nd := scanShaderLine(line, inBlock, inDoc)
		inBlock, inDoc = nb, nd
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

// scanShaderLine 扫描单行 Shader，返回是否含代码、是否为文档注释行，以及行末块注释状态。
func scanShaderLine(line string, inBlock, inDoc bool) (code, doc, nextInBlock, nextInDoc bool) {
	nextInBlock, nextInDoc = inBlock, inDoc
	lineDoc := inDoc // 行首已处于文档块注释中
	n := len(line)

	for i := 0; i < n; {
		if nextInBlock {
			if line[i] == '*' && i+1 < n && line[i+1] == '/' {
				nextInBlock = false
				nextInDoc = false
				i += 2
				continue
			}
			i++
			continue
		}

		switch c := line[i]; {
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case c == '/' && i+1 < n && line[i+1] == '/':
			// 行注释到行尾，其中出现的 /* 不开启块注释。
			i = n
		case c == '/' && i+1 < n && line[i+1] == '*':
			// /** 开头且不是空的 /**/ 才算文档注释。
			isDoc := i+2 < n && line[i+2] == '*' && !(i+3 < n && line[i+3] == '/')
			nextInBlock = true
			nextInDoc = isDoc
			if isDoc {
				lineDoc = true
			}
			i += 2
		case c == '"':
			code = true
			i++
			for i < n {
				if line[i] == '\\' {
					i += 2
					continue
				}
				if line[i] == '"' {
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
	if !code {
		doc = lineDoc
	}
	return code, doc, nextInBlock, nextInDoc
}
