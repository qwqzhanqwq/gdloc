// Package godot 解析 Godot 项目文件：project.godot、plugin.cfg、tscn/tres 等。
package godot

import (
	"os"
	"path/filepath"
	"strings"
)

// FindProjectName 从 root 起逐层向上查找 project.godot，返回 [application] 下的
// config/name。找到非空项目名时 ok 为 true；否则返回空串和 false。
func FindProjectName(root string) (name string, ok bool) {
	dir, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, "project.godot"))
		if err == nil {
			if n := parseProjectName(string(data)); n != "" {
				return n, true
			}
			return "", false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// parseProjectName 在近似 INI 的内容中读取 [application] 段的 config/name。
func parseProjectName(content string) string {
	content = strings.TrimPrefix(content, "\ufeff")
	inApplication := false
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSpace(line[1 : len(line)-1])
			inApplication = section == "application"
			continue
		}
		if !inApplication {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(key) != "config/name" {
			continue
		}
		return unquoteGodotString(strings.TrimSpace(value))
	}
	return ""
}

// unquoteGodotString 去掉 Godot 字符串字面量的引号并处理常见转义。
func unquoteGodotString(s string) string {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return s
	}
	body := s[1 : len(s)-1]
	var b strings.Builder
	for i := 0; i < len(body); i++ {
		if body[i] == '\\' && i+1 < len(body) {
			i++
			switch body[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte('\\')
				b.WriteByte(body[i])
			}
			continue
		}
		b.WriteByte(body[i])
	}
	return b.String()
}
