// Package stats 在计数器逐行结果之上做进阶统计（阶段 6）。
// 其中"疑似被注释掉的代码"是启发式估算，不是精确判断。
package stats

import (
	"sort"
	"strings"
	"unicode"

	"gdloc/internal/counter"
)

// Unit 是一段待统计的源码（普通文件或 tscn/tres 内嵌块）。
type Unit struct {
	Path     string
	Language string // "GDScript" 或 "Shader"
	Text     string
}

// LangStat 是按语言的注释统计。
type LangStat struct {
	Language  string
	Comments  int
	Suspected int
}

// Ratio 是疑似代码占注释行数的比例。
func (l LangStat) Ratio() float64 {
	if l.Comments == 0 {
		return 0
	}
	return float64(l.Suspected) / float64(l.Comments)
}

// Structure 是 GDScript 的结构计数。
type Structure struct {
	Funcs      int
	Signals    int
	ClassNames int
	Exports    int
}

// FileCode 是一个文件的代码行数。
type FileCode struct {
	Path     string
	Language string
	Code     int
}

// FuncInfo 是一个 GDScript 函数的位置与长度（代码行数）。
type FuncInfo struct {
	Path   string
	Line   int
	Name   string
	Length int
}

// Stats 是进阶统计结果。
type Stats struct {
	Languages    []LangStat
	Structure    Structure
	LongestFiles []FileCode
	LongestFuncs []FuncInfo
}

// Analyze 统计若干源码单元。top<=0 时取默认 10。
func Analyze(units []Unit, top int) Stats {
	if top <= 0 {
		top = 10
	}
	langs := map[string]*LangStat{}
	var files []FileCode
	var funcs []FuncInfo
	var structure Structure

	for _, u := range units {
		var lines []counter.Line
		switch u.Language {
		case "GDScript":
			lines = counter.GDScriptLines(u.Text)
		case "Shader":
			lines = counter.ShaderLines(u.Text)
		default:
			continue
		}
		ls := langs[u.Language]
		if ls == nil {
			ls = &LangStat{Language: u.Language}
			langs[u.Language] = ls
		}

		codeLines := 0
		for _, ln := range lines {
			switch ln.Kind {
			case counter.LineCode:
				codeLines++
			case counter.LineComment:
				ls.Comments++
				if isSuspectedComment(u.Language, ln.Comment) {
					ls.Suspected++
				}
			case counter.LineDoc:
				ls.Comments++
			}
		}
		files = append(files, FileCode{Path: u.Path, Language: u.Language, Code: codeLines})

		if u.Language != "GDScript" {
			continue
		}
		for i, ln := range lines {
			if ln.Kind != counter.LineCode {
				continue
			}
			s := strings.TrimSpace(ln.Code)
			if s == "" {
				continue
			}
			switch firstWord(s) {
			case "signal":
				structure.Signals++
			case "class_name":
				structure.ClassNames++
			}
			if strings.HasPrefix(s, "@export") {
				structure.Exports++
			}
			if name, ok := parseFuncDecl(s); ok {
				structure.Funcs++
				funcs = append(funcs, FuncInfo{Path: u.Path, Line: i + 1, Name: name, Length: funcLength(lines, i)})
			}
		}
	}

	out := Stats{Structure: structure}
	for _, ls := range langs {
		out.Languages = append(out.Languages, *ls)
	}
	sort.Slice(out.Languages, func(i, j int) bool { return out.Languages[i].Language < out.Languages[j].Language })

	sort.Slice(files, func(i, j int) bool {
		if files[i].Code != files[j].Code {
			return files[i].Code > files[j].Code
		}
		return files[i].Path < files[j].Path
	})
	out.LongestFiles = truncateFiles(files, top)

	sort.Slice(funcs, func(i, j int) bool {
		if funcs[i].Length != funcs[j].Length {
			return funcs[i].Length > funcs[j].Length
		}
		if funcs[i].Path != funcs[j].Path {
			return funcs[i].Path < funcs[j].Path
		}
		return funcs[i].Line < funcs[j].Line
	})
	out.LongestFuncs = truncateFuncs(funcs, top)
	return out
}

func truncateFiles(in []FileCode, top int) []FileCode {
	if top > 0 && len(in) > top {
		return in[:top]
	}
	return in
}

func truncateFuncs(in []FuncInfo, top int) []FuncInfo {
	if top > 0 && len(in) > top {
		return in[:top]
	}
	return in
}

// funcLength 返回从 func 行到函数体结束的代码行数；以缩进判断结束，空行/注释行不计入。
func funcLength(lines []counter.Line, idx int) int {
	base := lines[idx].Indent
	length := 1
	for j := idx + 1; j < len(lines); j++ {
		ln := lines[j]
		switch ln.Kind {
		case counter.LineBlank, counter.LineComment, counter.LineDoc:
			continue
		}
		if ln.Indent <= base {
			break
		}
		length++
	}
	return length
}

// parseFuncDecl 解析具名函数声明（含 static func），匿名 func(...) 返回 false。
func parseFuncDecl(s string) (string, bool) {
	if strings.HasPrefix(s, "static ") {
		s = strings.TrimSpace(s[len("static "):])
	}
	if !strings.HasPrefix(s, "func") {
		return "", false
	}
	rest := s[len("func"):]
	if rest == "" || (rest[0] != ' ' && rest[0] != '\t' && rest[0] != '(') {
		return "", false
	}
	rest = strings.TrimSpace(rest)
	if rest == "" || rest[0] == '(' {
		return "", false
	}
	end := strings.IndexAny(rest, " \t(")
	if end < 0 {
		return "", false
	}
	name := rest[:end]
	if name == "" {
		return "", false
	}
	return name, true
}

var gdKeywords = map[string]bool{
	"func": true, "var": true, "const": true, "if": true, "elif": true, "else": true,
	"for": true, "while": true, "match": true, "return": true, "await": true, "pass": true,
	"extends": true, "class_name": true, "signal": true,
}

var shaderKeywords = map[string]bool{
	"uniform": true, "void": true, "float": true, "int": true,
	"vec2": true, "vec3": true, "vec4": true, "if": true, "return": true,
}

var excludedPrefixes = []string{"TODO", "FIXME", "NOTE", "HACK"}

// isSuspectedComment 判断一行普通注释是否疑似被注释掉的代码。
func isSuspectedComment(language, raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if language == "GDScript" && (strings.HasPrefix(trimmed, "#region") || strings.HasPrefix(trimmed, "#endregion")) {
		return false
	}
	s := normalizeComment(language, raw)
	if s == "" || hasCJK(s) {
		return false
	}
	for _, p := range excludedPrefixes {
		if strings.HasPrefix(s, p) {
			return false
		}
	}
	if language == "GDScript" {
		if gdKeywords[firstWord(s)] {
			return true
		}
		if strings.HasPrefix(s, "@export") || strings.HasPrefix(s, "@onready") {
			return true
		}
		return strings.HasSuffix(s, ")") || strings.HasSuffix(s, ":") || strings.Contains(s, " = ")
	}
	if shaderKeywords[firstWord(s)] {
		return true
	}
	return strings.HasSuffix(s, ";")
}

// normalizeComment 去掉注释符号与首尾空白。
func normalizeComment(language, raw string) string {
	s := strings.TrimSpace(raw)
	if language == "Shader" {
		s = strings.TrimSpace(strings.TrimPrefix(s, "/*"))
		s = strings.TrimSpace(strings.TrimSuffix(s, "*/"))
		s = strings.TrimSpace(strings.TrimPrefix(s, "//"))
		s = strings.TrimSpace(strings.TrimPrefix(s, "*"))
		return s
	}
	return strings.TrimSpace(strings.TrimPrefix(s, "#"))
}

func firstWord(s string) string {
	if i := strings.IndexAny(s, " \t(:=;,."); i >= 0 {
		return s[:i]
	}
	return s
}

func hasCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
