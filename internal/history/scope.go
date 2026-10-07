package history

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/qwqzhanqwq/gdloc/internal/scan"
)

// scope 判断"某个提交中的某个路径"是否参与统计。
// 排除规则与增量统计一致：以 . 开头的目录、--exclude-dir、--exclude-addons，
// 以及 .gdignore（按当前工作区判断，不回溯历史）。
type scope struct {
	root          string // 扫描根绝对路径
	prefix        string // 扫描根相对仓库根的前缀，以 / 结尾；仓库根处为空串
	excludeDirs   map[string]bool
	excludeAddons bool
	addonsRel     string // addons 相对扫描根的路径，空表示不在扫描根内
	gdignore      map[string]bool
}

func newScope(root, prefix, addonsDir string, excludeDirs []string, excludeAddons bool) *scope {
	s := &scope{
		root:          root,
		prefix:        prefix,
		excludeDirs:   make(map[string]bool, len(excludeDirs)),
		excludeAddons: excludeAddons,
		gdignore:      map[string]bool{},
	}
	for _, d := range excludeDirs {
		if d != "" {
			s.excludeDirs[d] = true
		}
	}
	if addonsDir != "" {
		// addonsRel 相对扫描根，与 accept 里比较的路径同一坐标系；
		// 项目根位于扫描根之上时 addons 不在统计范围内，relToRepo 会失败。
		if rel, ok := relToRepo(root, addonsDir); ok {
			s.addonsRel = rel
		}
	}
	return s
}

// accept 判断仓库根相对路径（/ 分隔）是否参与统计。
func (s *scope) accept(repoRel string, t scan.FileType) bool {
	if !countable(t) {
		return false
	}
	sub, ok := s.sub(repoRel)
	if !ok || sub == "" {
		return false
	}
	if s.excludeAddons && s.addonsRel != "" &&
		(sub == s.addonsRel || strings.HasPrefix(sub, s.addonsRel+"/")) {
		return false
	}
	dir := path.Dir(sub)
	if dir == "." {
		return true // 扫描根下的文件，根目录本身不参与 .gdignore 判定
	}
	for _, name := range strings.Split(dir, "/") {
		if strings.HasPrefix(name, ".") || s.excludeDirs[name] {
			return false
		}
	}
	for _, anc := range ancestors(dir) {
		if s.hasGdignore(anc) {
			return false
		}
	}
	return true
}

// sub 把仓库根相对路径换算成扫描根相对路径；不在扫描根下时 ok 为 false。
func (s *scope) sub(repoRel string) (string, bool) {
	if s.prefix == "" {
		return repoRel, true
	}
	if !strings.HasPrefix(repoRel, s.prefix) {
		return "", false
	}
	return repoRel[len(s.prefix):], true
}

// hasGdignore 判断扫描根下的某个目录在当前工作区里是否有 .gdignore。
func (s *scope) hasGdignore(relDir string) bool {
	if v, ok := s.gdignore[relDir]; ok {
		return v
	}
	_, err := os.Stat(filepath.Join(s.root, filepath.FromSlash(relDir), ".gdignore"))
	v := err == nil
	s.gdignore[relDir] = v
	return v
}

// ancestors 返回目录自身及其全部上级目录，从最深层往上。
func ancestors(dir string) []string {
	var out []string
	for dir != "" && dir != "." && dir != "/" {
		out = append(out, dir)
		dir = path.Dir(dir)
		if dir == "." || dir == "/" {
			break
		}
	}
	return out
}

// relToRepo 求 target 相对 base 的路径；不在 base 下时 ok 为 false。
func relToRepo(base, target string) (string, bool) {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}
