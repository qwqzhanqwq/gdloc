package godot

import (
	"strconv"
	"strings"
)

// EmbeddedBlock 是 tscn/tres 中一段内嵌代码。ID 为 sub_resource 的 id；
// 主资源（[resource] 段）使用 "[resource]"。
type EmbeddedBlock struct {
	Language string // "GDScript" 或 "Shader"
	ID       string
	Source   string // 已反转义
}

// ParseResource 解析 .tscn/.tres 文本，取出内嵌 GDScript/Shader，并统计 VisualShader 数量。
// 解析宽容：格式损坏不报错，能取多少取多少。
func ParseResource(text string) (blocks []EmbeddedBlock, visualShaders int) {
	text = strings.TrimPrefix(text, "\ufeff")
	i, n := 0, len(text)
	atLineStart := true

	sectionKind := "" // sub_resource / gd_resource / resource / node / ...
	subType := ""     // 当前 sub/gd_resource 的 type
	subID := ""       // 当前 sub_resource 的 id
	mainType := ""    // gd_resource 的 type（主资源类型）

	for i < n {
		c := text[i]
		if c == '\n' {
			i++
			atLineStart = true
			continue
		}
		if c == '\r' {
			i++
			continue
		}

		if atLineStart && c == '[' {
			end := strings.IndexByte(text[i:], ']')
			if end < 0 {
				break
			}
			kind, attrs := parseHeader(text[i : i+end+1])
			i += end + 1
			atLineStart = false
			sectionKind = kind
			subType = attrs["type"]
			subID = attrs["id"]
			if kind == "gd_resource" {
				mainType = subType
			}
			if (kind == "sub_resource" || kind == "gd_resource") && subType == "VisualShader" {
				visualShaders++
			}
			continue
		}

		if atLineStart {
			lineEnd := n
			if nl := strings.IndexByte(text[i:], '\n'); nl >= 0 {
				lineEnd = i + nl
			}
			eq := strings.IndexByte(text[i:lineEnd], '=')
			if eq < 0 {
				i = lineEnd
				continue
			}
			key := strings.TrimSpace(text[i : i+eq])
			i += eq + 1
			for i < n && (text[i] == ' ' || text[i] == '\t') {
				i++
			}
			if i < n && text[i] == '"' {
				var value string
				value, i = parseQuoted(text, i)
				switch {
				case sectionKind == "sub_resource" && subType == "GDScript" && key == "script/source":
					blocks = append(blocks, EmbeddedBlock{Language: "GDScript", ID: subID, Source: value})
				case sectionKind == "sub_resource" && subType == "Shader" && key == "code":
					blocks = append(blocks, EmbeddedBlock{Language: "Shader", ID: subID, Source: value})
				case sectionKind == "resource" && mainType == "Shader" && key == "code":
					blocks = append(blocks, EmbeddedBlock{Language: "Shader", ID: "[resource]", Source: value})
				}
			}
			atLineStart = false
			continue
		}

		i++
	}
	return blocks, visualShaders
}

// parseHeader 解析 "[sub_resource type=\"X\" id=\"Y\"]"，返回段名与属性。
func parseHeader(header string) (kind string, attrs map[string]string) {
	attrs = map[string]string{}
	inner := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(header), "["), "]")
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return "", attrs
	}
	sp := strings.IndexAny(inner, " \t")
	if sp < 0 {
		return inner, attrs
	}
	kind = inner[:sp]
	rest := inner[sp:]
	for {
		rest = strings.TrimSpace(rest)
		if rest == "" {
			break
		}
		eq := strings.IndexByte(rest, '=')
		if eq < 0 {
			break
		}
		k := strings.TrimSpace(rest[:eq])
		rest = strings.TrimSpace(rest[eq+1:])
		if k == "" {
			break
		}
		if strings.HasPrefix(rest, "\"") {
			var v string
			var ni int
			v, ni = parseQuoted(rest, 0)
			attrs[k] = v
			rest = rest[ni:]
			continue
		}
		end := strings.IndexAny(rest, " \t")
		if end < 0 {
			attrs[k] = rest
			break
		}
		attrs[k] = rest[:end]
		rest = rest[end:]
	}
	return kind, attrs
}

// parseQuoted 从 i（text[i] 为开引号）解析一个 Godot 字符串，返回反转义结果与结束后的下标。
// 支持真实换行与 \n 等转义、\" 与 \\。未闭合时读到文本末尾。
func parseQuoted(text string, i int) (string, int) {
	i++ // 跳过开引号
	var b strings.Builder
	for i < len(text) {
		c := text[i]
		if c == '"' {
			return b.String(), i + 1
		}
		if c != '\\' {
			b.WriteByte(c)
			i++
			continue
		}
		if i+1 >= len(text) {
			b.WriteByte('\\')
			i++
			break
		}
		e := text[i+1]
		switch e {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case 'a':
			b.WriteByte('\a')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'v':
			b.WriteByte('\v')
		case '"':
			b.WriteByte('"')
		case '\'':
			b.WriteByte('\'')
		case '\\':
			b.WriteByte('\\')
		case 'u':
			if r, ok := parseHexRune(text, i+2, 4); ok {
				b.WriteRune(r)
				i += 6
				continue
			}
			b.WriteString(`\u`)
		case 'U':
			if r, ok := parseHexRune(text, i+2, 6); ok {
				b.WriteRune(r)
				i += 8
				continue
			}
			b.WriteString(`\U`)
		default:
			b.WriteByte('\\')
			b.WriteByte(e)
		}
		i += 2
	}
	return b.String(), i
}

// parseHexRune 解析 off 处 width 个十六进制字符；越界或非法时返回 false。
func parseHexRune(text string, off, width int) (rune, bool) {
	if off+width > len(text) {
		return 0, false
	}
	v, err := strconv.ParseUint(text[off:off+width], 16, 32)
	if err != nil {
		return 0, false
	}
	return rune(v), true
}
