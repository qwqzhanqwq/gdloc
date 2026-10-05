// Command gdloc 是 Godot 4 项目代码行数统计工具的入口。
// 阶段 0 只做目录遍历与文件分类展示，不做任何计数逻辑。
package main

import (
	"flag"
	"fmt"
	"os"

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
