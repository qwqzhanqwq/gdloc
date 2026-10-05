package godot

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Plugin 是 addons/ 下的一个插件。没有 plugin.cfg 时 HasPluginCfg 为 false，Name 取目录名。
type Plugin struct {
	Dir          string
	Name         string
	Version      string
	HasPluginCfg bool
}

// ScanPlugins 列出 addonsDir 的直接子目录作为插件，读取 plugin.cfg 的 name/version。
// addonsDir 不存在时返回 nil。解析失败时写入 warn 并退回目录名。
func ScanPlugins(addonsDir string, warn io.Writer) []Plugin {
	entries, err := os.ReadDir(addonsDir)
	if err != nil {
		return nil
	}
	var out []Plugin
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dir := filepath.Join(addonsDir, e.Name())
		p := Plugin{Dir: dir, Name: e.Name()}

		cfgPath := filepath.Join(dir, "plugin.cfg")
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			out = append(out, p)
			continue
		}
		name, version, ok := parsePluginCfg(string(data))
		if !ok {
			if warn != nil {
				fmt.Fprintf(warn, "warning: cannot parse %q, falling back to directory name\n", cfgPath)
			}
			out = append(out, p)
			continue
		}
		p.Name = name
		p.Version = version
		p.HasPluginCfg = true
		out = append(out, p)
	}
	return out
}

// parsePluginCfg 读取 plugin.cfg 中 [plugin] 段的 name 与 version。
func parsePluginCfg(content string) (name, version string, ok bool) {
	content = strings.TrimPrefix(content, "\ufeff")
	inPlugin := false
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inPlugin = strings.TrimSpace(line[1:len(line)-1]) == "plugin"
			continue
		}
		if !inPlugin {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch strings.TrimSpace(key) {
		case "name":
			name = unquoteGodotString(strings.TrimSpace(value))
		case "version":
			version = unquoteGodotString(strings.TrimSpace(value))
		}
	}
	return name, version, name != ""
}
