// Package scan 递归遍历目标目录，按扩展名对文件分类。
package scan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileType 是文件识别类型，未能识别的归为 TypeUnknown。
type FileType string

const (
	TypeGDScript FileType = "GDScript"
	TypeShader   FileType = "Shader"
	TypeCSharp   FileType = "CSharp"
	TypeScene    FileType = "Scene"
	TypeResource FileType = "Resource"
	TypeUnknown  FileType = "Unknown"
)

// Options 控制遍历时的排除行为。
type Options struct {
	// ExcludeDirs 是按目录名匹配的排除列表，任意层级命中即跳过。
	ExcludeDirs []string
	// NoIgnore 为真时不读取 .gitignore。
	NoIgnore bool
}

// FileEntry 是一条分类结果，Path 为相对根目录的路径，统一用 / 分隔。
type FileEntry struct {
	Path string
	Type FileType
}

// Classify 按文件扩展名分类，匹配不区分大小写。
func Classify(name string) FileType {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".gd":
		return TypeGDScript
	case ".gdshader", ".gdshaderinc":
		return TypeShader
	case ".cs":
		return TypeCSharp
	case ".tscn":
		return TypeScene
	case ".tres":
		return TypeResource
	default:
		return TypeUnknown
	}
}

// Scan 递归遍历 root，返回全部分类结果（含未识别文件）。
// 跳过所有以 . 开头的目录（覆盖 .godot/、.git/）、包含 .gdignore 的目录、
// opts.ExcludeDirs 命中的目录，以及被 .gitignore 忽略的路径（除非 opts.NoIgnore）。
// 根目录本身不参与跳过判定；.gitignore 只在 root 及子目录中查找。
// 返回的相对路径统一用 / 分隔，并按路径排序，保证输出稳定。
func Scan(root string, opts Options) ([]FileEntry, error) {
	exclude := make(map[string]bool, len(opts.ExcludeDirs))
	for _, name := range opts.ExcludeDirs {
		if name != "" {
			exclude[name] = true
		}
	}
	var out []FileEntry
	if err := walkDir(root, "", nil, opts, exclude, &out); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func walkDir(dir, rel string, inherited []string, opts Options, exclude map[string]bool, out *[]FileEntry) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	patterns := inherited
	if !opts.NoIgnore {
		patterns = loadGitignore(dir, rel, inherited)
	}
	matcher := newIgnoreMatcher(patterns)

	for _, e := range entries {
		name := e.Name()
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		if e.IsDir() {
			if strings.HasPrefix(name, ".") || exclude[name] {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, name, ".gdignore")); err == nil {
				continue
			}
			if !opts.NoIgnore && matcher.MatchesPath(childRel+"/") {
				continue
			}
			if err := walkDir(filepath.Join(dir, name), childRel, patterns, opts, exclude, out); err != nil {
				return err
			}
			continue
		}
		if !opts.NoIgnore && matcher.MatchesPath(childRel) {
			continue
		}
		*out = append(*out, FileEntry{Path: childRel, Type: Classify(name)})
	}
	return nil
}
