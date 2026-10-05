package scan

import (
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// loadGitignore 读取 dir 下的 .gitignore（若存在），把其中的模式重写为相对扫描根的
// 形式后追加到 inherited，返回新的模式列表。prefix 是 dir 相对根的路径（根为空串）。
func loadGitignore(dir, prefix string, inherited []string) []string {
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		return inherited
	}
	lines := strings.Split(string(data), "\n")
	patterns := make([]string, 0, len(inherited)+len(lines))
	patterns = append(patterns, inherited...)
	for _, line := range lines {
		patterns = append(patterns, rewritePatternLine(line, prefix))
	}
	return patterns
}

// rewritePatternLine 把某目录下 .gitignore 的一行重写为相对扫描根的模式：
// 含斜杠的锚定模式加上目录前缀；不含斜杠的模式在子目录中加 "/prefix/**/" 前缀，
// 使其只在该子目录内匹配；根目录的模式保持原样。
func rewritePatternLine(line, prefix string) string {
	line = strings.TrimRight(line, "\r")
	if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" || prefix == "" {
		return line
	}
	neg := strings.HasPrefix(line, "!")
	body := line
	if neg {
		body = body[1:]
	}
	dirOnly := strings.HasSuffix(body, "/")
	core := body
	if dirOnly {
		core = core[:len(core)-1]
	}
	anchored := strings.HasPrefix(core, "/") || strings.Contains(core, "/")

	var out string
	if anchored {
		core = strings.TrimPrefix(core, "/")
		out = "/" + prefix + "/" + core
	} else {
		out = "/" + prefix + "/**/" + core
	}
	if dirOnly {
		out += "/"
	}
	if neg {
		out = "!" + out
	}
	return out
}

// newIgnoreMatcher 编译累积的模式；无模式时也返回可用（永不匹配）的 matcher。
func newIgnoreMatcher(patterns []string) ignore.IgnoreParser {
	return ignore.CompileIgnoreLines(patterns...)
}
