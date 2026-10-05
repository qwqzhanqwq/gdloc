package counter

import "strings"

// CountShader 统计 Godot Shader 文本，规则见 AGENTS.md 4.1、4.3。
func CountShader(text string) Result {
	return summarize(ShaderLines(text))
}

// ShaderLines 逐行扫描 Shader，返回每行的归类与屏蔽字符串后的代码文本。
// 跨行状态只有块注释（含文档块注释标记）。
func ShaderLines(text string) []Line {
	raw := splitLines(strings.TrimPrefix(text, "\ufeff"))
	out := make([]Line, len(raw))
	inBlock := false
	inDoc := false
	for i, line := range raw {
		if strings.TrimSpace(line) == "" {
			out[i] = Line{Kind: LineBlank, Indent: indentWidth(line)}
			continue
		}
		code, doc, masked, comment, nb, nd := scanShaderLine(line, inBlock, inDoc)
		inBlock, inDoc = nb, nd
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

// scanShaderLine 扫描单行 Shader，返回是否含代码、是否文档注释行、屏蔽后的代码、注释原文，以及行末块注释状态。
func scanShaderLine(line string, inBlock, inDoc bool) (code, doc bool, masked, comment string, nextInBlock, nextInDoc bool) {
	out := make([]byte, len(line))
	for i := range out {
		out[i] = ' '
	}
	nextInBlock, nextInDoc = inBlock, inDoc
	lineDoc := inDoc
	if inBlock {
		comment = line
	}
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
			comment = line[i:]
			i = n
		case c == '/' && i+1 < n && line[i+1] == '*':
			isDoc := i+2 < n && line[i+2] == '*' && !(i+3 < n && line[i+3] == '/')
			nextInBlock = true
			nextInDoc = isDoc
			if isDoc {
				lineDoc = true
			}
			comment = line[i:]
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
			out[i] = line[i]
			code = true
			i++
		}
	}
	if !code {
		doc = lineDoc
	}
	return code, doc, string(out), comment, nextInBlock, nextInDoc
}
