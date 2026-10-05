// Command gdloc 是 Godot 4 项目代码行数统计工具的入口。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gdloc/internal/godot"
	"gdloc/internal/report"
	"gdloc/internal/scan"
)

// version 默认版本号，可通过 -ldflags "-X main.version=..." 覆盖。
var version = "0.0.1-dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(argv []string, stdout, stderr io.Writer) int {
	// --version 与位置参数顺序无关，先单独识别。
	for _, a := range argv {
		if a == "--version" || a == "-version" {
			fmt.Fprintf(stdout, "gdloc %s\n", version)
			return 0
		}
	}

	fs := flag.NewFlagSet("gdloc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		byFile     bool
		sortKey    string
		top        int
		jsonOut    bool
		excludeDir string
		noIgnore   bool
	)
	fs.BoolVar(&byFile, "by-file", false, "list one row per file")
	fs.StringVar(&sortKey, "sort", "code", "sort key: code|comments|blanks|lines|files")
	fs.IntVar(&top, "top", 0, "show only the first N rows")
	fs.BoolVar(&jsonOut, "json", false, "output JSON")
	fs.StringVar(&excludeDir, "exclude-dir", "", "comma-separated directory names to exclude")
	fs.BoolVar(&noIgnore, "no-ignore", false, "do not read .gitignore (not yet implemented)")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: gdloc [path] [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	// flag 会在首个位置参数处停止解析，这里循环解析以支持参数与路径任意顺序。
	var positionals []string
	args := argv
	for {
		if err := fs.Parse(args); err != nil {
			if err == flag.ErrHelp {
				return 0
			}
			return 1
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		positionals = append(positionals, rest[0])
		args = rest[1:]
	}

	if len(positionals) > 1 {
		fmt.Fprintln(stderr, "error: too many arguments")
		fs.Usage()
		return 1
	}
	if !report.ValidSortKey(sortKey) {
		fmt.Fprintf(stderr, "error: invalid --sort value %q\n", sortKey)
		return 1
	}
	if top < 0 {
		fmt.Fprintln(stderr, "error: --top must not be negative")
		return 1
	}

	root := "."
	if len(positionals) == 1 {
		root = positionals[0]
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot resolve path %q: %v\n", root, err)
		return 2
	}
	info, err := os.Stat(rootAbs)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "error: path %q does not exist or is not readable\n", root)
		return 2
	}

	entries, err := scan.Scan(rootAbs, scan.Options{ExcludeDirs: splitList(excludeDir)})
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot scan %q: %v\n", root, err)
		return 2
	}

	rep := report.Build(rootAbs, entries, stderr)
	if name, ok := godot.FindProjectName(rootAbs); ok {
		rep.ProjectName = name
	}
	rep.SortBy(sortKey)

	if jsonOut {
		report.WriteJSON(stdout, rep, byFile, top)
	} else {
		report.WriteTable(stdout, rep, byFile, top)
	}
	return 0
}

// splitList 拆分逗号分隔的参数，去掉空白与空项。
func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
