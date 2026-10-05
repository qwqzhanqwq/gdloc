// Package scan 递归遍历目标目录，按扩展名对文件分类。
package scan

import (
	"io/fs"
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
// 跳过所有以 . 开头的目录（覆盖 .godot/、.git/）。
// 返回的相对路径统一用 / 分隔，并按路径排序，保证输出稳定。
func Scan(root string) ([]FileEntry, error) {
	var out []FileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// 根目录本身不跳过，只跳过其下的点目录。
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, FileEntry{
			Path: filepath.ToSlash(rel),
			Type: Classify(d.Name()),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}
