package history

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/qwqzhanqwq/gdloc/internal/scan"
)

// side 是改动的一侧版本：可能来自仓库对象，也可能直接来自工作区。
type side struct {
	path    string // 仓库根相对路径，/ 分隔
	typ     scan.FileType
	present bool
	sha     string // 来自仓库时的 blob 对象名
	text    string // 工作区内容
	isText  bool
}

// textOf 取某一侧版本的文本；blob 通过常驻的 cat-file 进程读取。
func (c *collector) textOf(s side) (string, bool) {
	if s.isText {
		return s.text, true
	}
	if s.sha == "" || s.sha == zeroSHA {
		return "", false
	}
	data, ok, err := c.cat.get(s.sha)
	if err != nil {
		c.fail(fmt.Errorf("cannot read git object %s: %v", s.sha, err))
		return "", false
	}
	if !ok {
		c.warnOnce(s.sha, "cannot read git object %s", s.sha)
		return "", false
	}
	return string(data), true
}

// classify 按文件类型逐行归类；tscn/tres 另外提取内嵌块。
func (c *collector) classify(s side, text string) *fileContent {
	fc := &fileContent{kinds: lineKinds(text, s.typ)}
	if !isCode(s.typ) {
		fc.blocks = blocksOf(text)
	}
	return fc
}

// info 返回某一侧版本的分类结果，blob 版本按对象名缓存。
func (c *collector) info(s side, text string) *fileContent {
	if !s.isText {
		if fc, ok := c.cache[c.key(s)]; ok {
			return fc
		}
	}
	fc := c.classify(s, text)
	if !s.isText {
		c.cache[c.key(s)] = fc
	}
	return fc
}

// content 在不需要文本做 diff 时使用：命中缓存就不必再取内容。
func (c *collector) content(s side) (*fileContent, bool) {
	if !s.isText {
		if fc, ok := c.cache[c.key(s)]; ok {
			return fc, true
		}
	}
	text, ok := c.textOf(s)
	if !ok {
		return nil, false
	}
	return c.info(s, text), true
}

// wholeFile 把某一侧版本整体算作新增或删除。
func (c *collector) wholeFile(s side, added bool) Delta {
	fc, ok := c.content(s)
	if !ok {
		return Delta{}
	}
	var d Delta
	if isCode(s.typ) {
		d.addAll(fc.kinds, added)
		return d
	}
	for _, b := range fc.blocks {
		d.addAll(b.kinds, added)
	}
	return d
}

// delta 计算两侧版本之间的分类增量；ok 为 false 表示两侧都不参与统计。
func (c *collector) delta(old, new side) (Delta, bool) {
	oldOK := old.present && c.scope.accept(old.path, old.typ)
	newOK := new.present && c.scope.accept(new.path, new.typ)
	if !oldOK && !newOK {
		return Delta{}, false
	}
	// 只有新旧路径都在范围内且类型相同才按内容逐行比较；
	// 跨越扫描根/排除目录或改了扩展名时，旧版本整份算删除、新版本整份算新增。
	if oldOK && newOK && old.typ == new.typ {
		if d, ok := c.pairedDelta(old, new); ok {
			return d, true
		}
	}
	var d Delta
	if oldOK {
		d.add(c.wholeFile(old, false))
	}
	if newOK {
		d.add(c.wholeFile(new, true))
	}
	return d, true
}

// pairedDelta 对两个版本做逐行 diff：新增行按新版本的分类、删除行按旧版本的分类。
func (c *collector) pairedDelta(old, new side) (Delta, bool) {
	oldText, ok1 := c.textOf(old)
	newText, ok2 := c.textOf(new)
	if !ok1 || !ok2 {
		return Delta{}, false
	}
	oldText, newText = alignLineEndings(oldText, newText, old.isText, new.isText)

	var d Delta
	if isCode(old.typ) {
		added, deleted := diffLines(splitContent(oldText), splitContent(newText))
		d.addLines(c.info(old, oldText).kinds, deleted, false)
		d.addLines(c.info(new, newText).kinds, added, true)
		return d, true
	}
	d.add(blockDelta(c.info(old, oldText).blocks, c.info(new, newText).blocks))
	return d, true
}

// worktreeText 读取工作区文件内容。
func (c *collector) worktreeText(rel string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(c.repo.root, filepath.FromSlash(rel)))
	if err != nil {
		c.warnf("cannot read %q: %v", c.displayPath(rel), err)
		return "", false
	}
	return string(data), true
}

// displayPath 把仓库根相对路径换算成扫描根相对路径，与其它输出保持一致。
func (c *collector) displayPath(repoRel string) string {
	if sub, ok := c.scope.sub(repoRel); ok {
		return sub
	}
	return repoRel
}

// sideOf 由 raw 记录的一侧构造版本；存在性只由 git 模式决定。
func sideOf(p, sha, mode string) side {
	return side{path: p, typ: scan.Classify(path.Base(p)), present: regularFile(mode), sha: sha}
}

// codeCount 汇总一个版本的代码行数；Scene/Resource 只算内嵌块。
func codeCount(fc *fileContent, t scan.FileType) int {
	if isCode(t) {
		return codeOf(fc.kinds)
	}
	n := 0
	for _, b := range fc.blocks {
		n += codeOf(b.kinds)
	}
	return n
}

// warnOnce 按 key 去重，避免同一个坏对象刷屏。
func (c *collector) warnOnce(key, format string, args ...any) {
	if c.warned[key] {
		return
	}
	c.warned[key] = true
	c.warnf(format, args...)
}

func (c *collector) warnf(format string, args ...any) {
	if c.warn != nil {
		fmt.Fprintf(c.warn, "warning: "+format+"\n", args...)
	}
}

// fail 记录无法恢复的 git 错误（例如 cat-file 进程退出）。
func (c *collector) fail(err error) {
	if c.fatal == nil {
		c.fatal = err
	}
}

func (c *collector) key(s side) string { return string(s.typ) + "\x00" + s.sha }
