package history

import (
	"strings"

	"github.com/qwqzhanqwq/gdloc/internal/counter"
	"github.com/qwqzhanqwq/gdloc/internal/godot"
	"github.com/qwqzhanqwq/gdloc/internal/scan"
)

// Delta 是一组新增/删除行数，按分类拆分；Doc 不单列，计入注释。
type Delta struct {
	AddedCode       int
	DeletedCode     int
	AddedComments   int
	DeletedComments int
	AddedBlanks     int
	DeletedBlanks   int
	Commits         int // 只在提交与汇总层级有意义
}

// NetCode 是代码行的净变化。
func (d Delta) NetCode() int { return d.AddedCode - d.DeletedCode }

// add 累加另一个 Delta。
func (d *Delta) add(o Delta) {
	d.AddedCode += o.AddedCode
	d.DeletedCode += o.DeletedCode
	d.AddedComments += o.AddedComments
	d.DeletedComments += o.DeletedComments
	d.AddedBlanks += o.AddedBlanks
	d.DeletedBlanks += o.DeletedBlanks
	d.Commits += o.Commits
}

// addLines 把 idx 指定的行按分类累加到新增侧或删除侧。
func (d *Delta) addLines(kinds []counter.LineKind, idx []int, added bool) {
	var code, comments, blanks int
	for _, i := range idx {
		if i < 0 || i >= len(kinds) {
			continue
		}
		switch kinds[i] {
		case counter.LineBlank:
			blanks++
		case counter.LineCode:
			code++
		default: // 普通注释与文档注释都算注释
			comments++
		}
	}
	if added {
		d.AddedCode += code
		d.AddedComments += comments
		d.AddedBlanks += blanks
		return
	}
	d.DeletedCode += code
	d.DeletedComments += comments
	d.DeletedBlanks += blanks
}

// addAll 把整份行按分类累加到新增侧或删除侧。
func (d *Delta) addAll(kinds []counter.LineKind, added bool) {
	for _, k := range kinds {
		switch k {
		case counter.LineBlank:
			if added {
				d.AddedBlanks++
			} else {
				d.DeletedBlanks++
			}
		case counter.LineCode:
			if added {
				d.AddedCode++
			} else {
				d.DeletedCode++
			}
		default: // 普通注释与文档注释都算注释
			if added {
				d.AddedComments++
			} else {
				d.DeletedComments++
			}
		}
	}
}

// codeOf 汇总若干行的代码行数。
func codeOf(kinds []counter.LineKind) int {
	n := 0
	for _, k := range kinds {
		if k == counter.LineCode {
			n++
		}
	}
	return n
}

// countable 判断文件类型是否参与历史统计。
func countable(t scan.FileType) bool {
	switch t {
	case scan.TypeGDScript, scan.TypeShader, scan.TypeScene, scan.TypeResource:
		return true
	default:
		return false
	}
}

// isCode 判断是否为逐行统计的代码语言（Scene/Resource 只统计内嵌块）。
func isCode(t scan.FileType) bool {
	return t == scan.TypeGDScript || t == scan.TypeShader
}

// splitContent 按行切分文本，规则与 counter 内部一致：去掉 BOM，末尾单个换行不算新行。
func splitContent(text string) []string {
	text = strings.TrimPrefix(text, "\ufeff")
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// lineKinds 逐行归类，复用与总量统计相同的计数器。
func lineKinds(text string, t scan.FileType) []counter.LineKind {
	var lines []counter.Line
	switch t {
	case scan.TypeGDScript:
		lines = counter.GDScriptLines(text)
	case scan.TypeShader:
		lines = counter.ShaderLines(text)
	}
	out := make([]counter.LineKind, len(lines))
	for i, ln := range lines {
		out[i] = ln.Kind
	}
	return out
}

// embeddedBlock 是 tscn/tres 里的一段内嵌代码，key 由语言与块 id 组成，用于前后版本配对。
type embeddedBlock struct {
	key   string
	lines []string
	kinds []counter.LineKind
}

// blocksOf 提取内嵌 GDScript/Shader 并逐行归类；Scene/Resource 自身的行数不参与。
func blocksOf(text string) []embeddedBlock {
	parsed, _ := godot.ParseResource(text)
	out := make([]embeddedBlock, 0, len(parsed))
	for _, b := range parsed {
		var t scan.FileType
		switch b.Language {
		case "GDScript":
			t = scan.TypeGDScript
		case "Shader":
			t = scan.TypeShader
		default:
			continue
		}
		out = append(out, embeddedBlock{
			key:   b.Language + "\x00" + b.ID,
			lines: splitContent(b.Source),
			kinds: lineKinds(b.Source, t),
		})
	}
	return out
}

// fileContent 是一个版本的分类结果：整文件逐行归类，或 tscn/tres 的内嵌块。
type fileContent struct {
	kinds  []counter.LineKind
	blocks []embeddedBlock
}

// blockDelta 按块 id 配对做逐行 diff：新增的块全部算新增，消失的块全部算删除。
func blockDelta(oldBlocks, newBlocks []embeddedBlock) Delta {
	oldByKey := groupBlocks(oldBlocks)
	newByKey := groupBlocks(newBlocks)

	var d Delta
	seen := map[string]bool{}
	for _, b := range newBlocks {
		if seen[b.key] {
			continue
		}
		seen[b.key] = true
		olds, news := oldByKey[b.key], newByKey[b.key]
		n := len(olds)
		if len(news) < n {
			n = len(news)
		}
		for i := 0; i < n; i++ {
			added, deleted := diffLines(olds[i].lines, news[i].lines)
			d.addLines(news[i].kinds, added, true)
			d.addLines(olds[i].kinds, deleted, false)
		}
		for i := n; i < len(news); i++ {
			d.addAll(news[i].kinds, true)
		}
		for i := n; i < len(olds); i++ {
			d.addAll(olds[i].kinds, false)
		}
	}
	for _, b := range oldBlocks {
		if seen[b.key] {
			continue
		}
		seen[b.key] = true
		for _, o := range oldByKey[b.key] {
			d.addAll(o.kinds, false)
		}
	}
	return d
}

func groupBlocks(blocks []embeddedBlock) map[string][]embeddedBlock {
	out := make(map[string][]embeddedBlock, len(blocks))
	for _, b := range blocks {
		out[b.key] = append(out[b.key], b)
	}
	return out
}

// alignLineEndings 只在工作区版本与 blob 版本比较时对齐行尾：
// Windows 上 blob 是 LF、工作区是 CRLF 时，不能把每一行都算成改动。
func alignLineEndings(oldText, newText string, oldIsText, newIsText bool) (string, string) {
	if oldIsText == newIsText {
		return oldText, newText
	}
	blob, work := oldText, newText
	if oldIsText {
		blob, work = newText, oldText
	}
	if strings.Contains(blob, "\r\n") || !strings.Contains(work, "\r\n") {
		return oldText, newText
	}
	work = strings.ReplaceAll(work, "\r\n", "\n")
	if newIsText {
		return oldText, work
	}
	return work, newText
}
