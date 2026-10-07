package history

import "testing"

// TestHistoryRules 覆盖 4.7 的排除规则与内容口径：内嵌代码、多行结构、排除目录、.gdignore。
func TestHistoryRules(t *testing.T) {
	runHistoryCases(t, []historyCase{
		{
			name: "whitespace-only line added inside a multi-line string counts as blank",
			setup: func(r *testRepo) {
				r.write("main.gd", "var s = \"\"\"\nline\nline2\n\"\"\"\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
				r.write("main.gd", "var s = \"\"\"\nline\n   \nline2\n\"\"\"\n")
				r.commit("2024-01-03T10:00:00+08:00", "c2")
			},
			opts: Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 3)},
			want: []period{
				{date: "2024-01-02", commits: 1, addCode: 4, code: 4},
				{date: "2024-01-03", commits: 1, addBlanks: 1, code: 4},
			},
			headCode: 4, totalCode: 4,
		},
		{
			name: "shader block and doc comments",
			setup: func(r *testRepo) {
				r.write("player.gdshader", "shader_type canvas_item;\n/*\n block\n*/\nvoid fragment() {}\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
				r.write("player.gdshader", "shader_type canvas_item;\n/*\n block\n*/\n/**\n doc\n*/\nvoid fragment() {}\n")
				r.commit("2024-01-03T10:00:00+08:00", "c2")
			},
			opts: Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 3)},
			want: []period{
				{date: "2024-01-02", commits: 1, addCode: 2, addComments: 3, code: 2},
				{date: "2024-01-03", commits: 1, addComments: 3, code: 2},
			},
			headCode: 2, totalCode: 2,
		},
		{
			name: "embedded GDScript blocks are modified, added and removed",
			setup: func(r *testRepo) {
				const header = "[gd_scene load_steps=2 format=3]\n\n"
				r.write("scene.tscn", header+tscnScript("GDScript_a", "extends Node\nvar a = 1\n"))
				r.commit("2024-01-02T10:00:00+08:00", "c1")
				r.write("scene.tscn", header+tscnScript("GDScript_a", "extends Node\nvar a = 1\nvar b = 2\n"))
				r.commit("2024-01-03T10:00:00+08:00", "c2")
				r.write("scene.tscn", header+tscnScript("GDScript_a", "extends Node\nvar a = 1\nvar b = 2\n")+
					tscnScript("GDScript_b", "var c = 1\n"))
				r.commit("2024-01-04T10:00:00+08:00", "c3")
				r.write("scene.tscn", header+tscnScript("GDScript_b", "var c = 1\n"))
				r.commit("2024-01-05T10:00:00+08:00", "c4")
			},
			opts: Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 5)},
			want: []period{
				{date: "2024-01-02", commits: 1, addCode: 2, code: 2},
				{date: "2024-01-03", commits: 1, addCode: 1, code: 3},
				{date: "2024-01-04", commits: 1, addCode: 1, code: 4},
				{date: "2024-01-05", commits: 1, delCode: 3, code: 1},
			},
			headCode: 1, totalCode: 1,
		},
		{
			name: "embedded shader in a main resource",
			setup: func(r *testRepo) {
				const header = "[gd_resource type=\"Shader\" format=3]\n\n[resource]\n"
				r.write("res.tres", header+"code = \"shader_type canvas_item;\\n// c\\n\"\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
				r.write("res.tres", header+"code = \"shader_type canvas_item;\\n// c\\nvoid fragment() {}\\n\"\n")
				r.commit("2024-01-03T10:00:00+08:00", "c2")
			},
			opts: Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 3)},
			want: []period{
				{date: "2024-01-02", commits: 1, addCode: 1, addComments: 1, code: 1},
				{date: "2024-01-03", commits: 1, addCode: 1, code: 2},
			},
			headCode: 2, totalCode: 2,
		},
		{
			name:   "scan root inside a repository subdirectory",
			subdir: "src",
			setup: func(r *testRepo) {
				r.write("src/main.gd", "var m = 1\nvar n = 2\n")
				r.write("tools/tool.gd", "var t = 1\nvar u = 2\nvar v = 3\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
				r.write("tools/other.gd", "var w = 1\n")
				r.commit("2024-01-03T10:00:00+08:00", "c2")
			},
			opts: Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 3)},
			want: []period{
				{date: "2024-01-02", commits: 1, addCode: 2, code: 2},
				// 只改动扫描根之外的文件：不计入 Commits，也不产生增量。
				{date: "2024-01-03", code: 2},
			},
			headCode: 2, totalCode: 2,
		},
		{
			name:   "rename leaving the scan root counts as a full delete",
			subdir: "src",
			setup: func(r *testRepo) {
				r.write("src/a.gd", "var a = 1\nvar b = 2\nvar c = 3\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
				// 内容不变、只是移出扫描根：不做逐行 diff，旧版本整份算删除。
				r.remove("src/a.gd")
				r.write("moved.gd", "var a = 1\nvar b = 2\nvar c = 3\n")
				r.commit("2024-01-03T10:00:00+08:00", "c2")
			},
			opts: Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 3)},
			want: []period{
				{date: "2024-01-02", commits: 1, addCode: 3, code: 3},
				{date: "2024-01-03", commits: 1, delCode: 3, code: 0},
			},
			headCode: 0, totalCode: 0,
		},
		{
			name: "hidden directories are excluded",
			setup: func(r *testRepo) {
				r.write("main.gd", "var m = 1\n")
				r.write(".hidden/secret.gd", "var s = 1\nvar t = 2\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
			},
			opts:     Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 2)},
			want:     []period{{date: "2024-01-02", commits: 1, addCode: 1, code: 1}},
			headCode: 1, totalCode: 1,
		},
		{
			name: "exclude-dir and exclude-addons",
			setup: func(r *testRepo) {
				r.write("main.gd", "var m = 1\n")
				r.write("addons/plug/plug.gd", "var p = 1\nvar q = 2\n")
				r.write("tools/t.gd", "var t = 1\nvar u = 2\nvar v = 3\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
			},
			opts: Options{
				Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 2),
				ExcludeDirs: []string{"tools"}, ExcludeAddons: true,
			},
			want:     []period{{date: "2024-01-02", commits: 1, addCode: 1, code: 1}},
			headCode: 1, totalCode: 1,
		},
		{
			name:   "exclude-addons works when the scan root is a subdirectory",
			subdir: "src",
			setup: func(r *testRepo) {
				r.write("src/main.gd", "var m = 1\n")
				r.write("src/addons/plug/plug.gd", "var p = 1\nvar q = 2\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
			},
			opts: Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 2), ExcludeAddons: true},
			want: []period{{date: "2024-01-02", commits: 1, addCode: 1, code: 1}},
			// 扫描根是 src/，addons 路径必须先换算到扫描根再比较。
			headCode: 1, totalCode: 1,
		},
		{
			name: "gdignore is applied from the current workspace",
			setup: func(r *testRepo) {
				r.write("secret/a.gd", "var s = 1\n")
				r.write("public/b.gd", "var p = 1\n")
				r.commit("2024-01-02T10:00:00+08:00", "c1")
				r.write("secret/a.gd", "var s = 1\nvar s2 = 2\n")
				r.commit("2024-01-03T10:00:00+08:00", "c2")
				// 只存在于当前工作区、没有提交：历史里的 secret/ 也要排除。
				r.write("secret/.gdignore", "")
			},
			opts: Options{Since: testDay(2024, 1, 2), Until: testDay(2024, 1, 3)},
			want: []period{
				{date: "2024-01-02", commits: 1, addCode: 1, code: 1},
				{date: "2024-01-03", code: 1},
			},
			headCode: 1, totalCode: 1,
		},
	})
}
