// Command gdloc 是 Godot 4 项目代码行数统计工具的入口。
// 当前为临时输出：目录遍历 + 文件分类 + GDScript 汇总，阶段 3 会替换为正式表格。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gdloc/internal/counter"
	"gdloc/internal/scan"
)

// version 默认版本号，可通过 -ldflags "-X main.version=..." 覆盖。
var version = "0.0.1-dev"

// groupOrder 是临时输出的分组顺序。
var groupOrder = []scan.FileType{
	scan.TypeGDScript,
	scan.TypeShader,
	scan.TypeCSharp,
	scan.TypeScene,
	scan.TypeResource,
}

func main() {
	fs := flag.NewFlagSet("gdloc", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	// 先手动识别 --version，使其与路径参数顺序无关（flag 遇到位置参数后会停止解析）。
	for _, a := range os.Args[1:] {
		if a == "--version" || a == "-version" {
			*showVersion = true
			fmt.Printf("gdloc %s\n", version)
			return
		}
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		// flag 已向 stderr 输出具体错误，这里按约定统一退出码 1。
		os.Exit(1)
	}
	if *showVersion {
		fmt.Printf("gdloc %s\n", version)
		return
	}
	args := fs.Args()
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "error: too many arguments")
		fs.Usage()
		os.Exit(1)
	}
	root := "."
	if len(args) == 1 {
		root = args[0]
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "error: path %q does not exist or is not readable\n", root)
		os.Exit(2)
	}
	entries, err := scan.Scan(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot scan %q: %v\n", root, err)
		os.Exit(2)
	}
	printGrouped(entries)
	printLangSummary("GDScript", scan.TypeGDScript, root, entries, counter.CountGDScript)
	printLangSummary("Shader", scan.TypeShader, root, entries, counter.CountShader)
}

// printLangSummary 汇总某一类型文件的计数，读取失败只警告不中断。
func printLangSummary(label string, t scan.FileType, root string, entries []scan.FileEntry, count func(string) counter.Result) {
	var total counter.Result
	files := 0
	for _, e := range entries {
		if e.Type != t {
			continue
		}
		files++
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.Path)))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot read %q: %v\n", e.Path, err)
			continue
		}
		r := count(string(data))
		total.Lines += r.Lines
		total.Code += r.Code
		total.Comments += r.Comments
		total.Doc += r.Doc
		total.Blanks += r.Blanks
	}
	fmt.Printf("%s summary: files=%d lines=%d code=%d comments=%d doc=%d blanks=%d\n",
		label, files, total.Lines, total.Code, total.Comments, total.Doc, total.Blanks)
}

// printGrouped 按类型分组列出识别到的文件，末尾输出未识别文件数。
func printGrouped(entries []scan.FileEntry) {
	groups := make(map[scan.FileType][]string, len(groupOrder))
	unknown := 0
	for _, e := range entries {
		if e.Type == scan.TypeUnknown {
			unknown++
			continue
		}
		groups[e.Type] = append(groups[e.Type], e.Path)
	}
	for _, t := range groupOrder {
		files := groups[t]
		if len(files) == 0 {
			continue
		}
		fmt.Printf("%s (%d files):\n", t, len(files))
		for _, p := range files {
			fmt.Printf("  %s\n", p)
		}
	}
	fmt.Printf("Unrecognized files: %d\n", unknown)
}
