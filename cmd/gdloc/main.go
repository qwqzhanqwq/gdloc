// Command gdloc 是 Godot 4 项目代码行数统计工具的入口。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/qwqzhanqwq/gdloc/internal/godot"
	"github.com/qwqzhanqwq/gdloc/internal/report"
	"github.com/qwqzhanqwq/gdloc/internal/scan"
)

// version 默认版本号，可通过 -ldflags "-X main.version=..." 覆盖。
var version = "0.0.1-dev"

func init() {
	// go install ...@vX.Y.Z 不经过 ldflags，从模块信息里取版本号。
	if info, ok := debug.ReadBuildInfo(); ok && version == "0.0.1-dev" {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			version = strings.TrimPrefix(v, "v")
		}
	}
}

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
		byFile        bool
		byDir         bool
		byAddon       bool
		statsView     bool
		excludeAddons bool
		sortKey       string
		top           int
		jsonOut       bool
		excludeDir    string
		noIgnore      bool
	)
	fs.BoolVar(&byFile, "by-file", false, "list one row per file")
	fs.BoolVar(&byDir, "by-dir", false, "group by top-level directory")
	fs.BoolVar(&byAddon, "by-addon", false, "group by addon plugin")
	fs.BoolVar(&statsView, "stats", false, "advanced statistics view")
	fs.BoolVar(&excludeAddons, "exclude-addons", false, "do not count the addons/ directory")
	fs.StringVar(&sortKey, "sort", "code", "sort key: code|comments|blanks|lines|files")
	fs.IntVar(&top, "top", 0, "show only the first N rows")
	fs.BoolVar(&jsonOut, "json", false, "output JSON")
	fs.StringVar(&excludeDir, "exclude-dir", "", "comma-separated directory names to exclude")
	fs.BoolVar(&noIgnore, "no-ignore", false, "do not read .gitignore")
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
	modes := 0
	for _, b := range []bool{byFile, byDir, byAddon} {
		if b {
			modes++
		}
	}
	if modes > 1 {
		fmt.Fprintln(stderr, "error: --by-file, --by-dir and --by-addon are mutually exclusive")
		return 1
	}
	if statsView && modes > 0 {
		fmt.Fprintln(stderr, "error: --stats cannot be combined with --by-file, --by-dir or --by-addon")
		return 1
	}
	if excludeAddons && byAddon {
		fmt.Fprintln(stderr, "error: --exclude-addons cannot be used with --by-addon")
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

	projectRoot, ok := godot.FindProjectRoot(rootAbs)
	if !ok {
		projectRoot = rootAbs
	}
	addonsDir := filepath.Join(projectRoot, "addons")

	scanOpts := scan.Options{ExcludeDirs: splitList(excludeDir), NoIgnore: noIgnore}
	if excludeAddons {
		scanOpts.ExcludePaths = []string{addonsDir}
	}
	entries, err := scan.Scan(rootAbs, scanOpts)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot scan %q: %v\n", root, err)
		return 2
	}

	statsTop := top
	if statsTop == 0 {
		statsTop = 10
	}
	rep := report.Build(rootAbs, entries, stderr, statsView, statsTop)
	if name, ok := godot.FindProjectName(rootAbs); ok {
		rep.ProjectName = name
	}

	mode := report.ModeLanguage
	switch {
	case byFile:
		mode = report.ModeFile
	case byDir:
		mode = report.ModeDir
		rep.Groups = report.GroupByDir(rep)
	case byAddon:
		mode = report.ModeAddon
		plugins := godot.ScanPlugins(addonsDir, stderr)
		rep.Groups = report.GroupByAddon(rep, rootAbs, addonsDir, plugins)
	case statsView:
		mode = report.ModeStats
	}
	rep.SortBy(sortKey)

	if jsonOut {
		report.WriteJSON(stdout, rep, mode, top)
	} else {
		report.WriteTable(stdout, rep, mode, top)
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
