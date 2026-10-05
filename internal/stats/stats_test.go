package stats

import "testing"

func TestIsSuspectedCommentGDScript(t *testing.T) {
	cases := []struct {
		comment string
		want    bool
	}{
		{"# func foo():", true},
		{"# var x = 1", true},
		{"# const A = 2", true},
		{"# if cond:", true},
		{"# else:", true},
		{"# for i in range(3):", true},
		{"# while true:", true},
		{"# match x:", true},
		{"# return value", true},
		{"# await signal", true},
		{"# pass", true},
		{"# extends Node", true},
		{"# class_name Foo", true},
		{"# signal hit", true},
		{"# @export var speed = 1", true},
		{"# @onready var n = $Node", true},
		{"# do_something()", true},
		{"# x = y", true},
		{"# simple note", false},
		{"# TODO: implement", false},
		{"# FIXME later", false},
		{"# NOTE something", false},
		{"# HACK around", false},
		{"# 这是中文注释 func", false},
		{"#region", false},
		{"#endregion", false},
		{"# ---------------------------------------------------------------------------", false},
		{"#", false},
	}
	for _, c := range cases {
		if got := isSuspectedComment("GDScript", c.comment); got != c.want {
			t.Errorf("isSuspectedComment(GDScript, %q) = %v, want %v", c.comment, got, c.want)
		}
	}
}

func TestIsSuspectedCommentShader(t *testing.T) {
	cases := []struct {
		comment string
		want    bool
	}{
		{"// uniform float x;", true},
		{"// void fragment() {", true},
		{"// float y = 1.0;", true},
		{"// vec3 c;", true},
		{"// if (x) {", true},
		{"// return;", true},
		{"// x = 1.0;", true},
		{"//", false},
		{"// a simple note", false},
		{"// TODO", false},
		{"// 中文注释", false},
		{" * vec2 uv; ", true}, // 块注释内部的普通注释行同样参与判断
	}
	for _, c := range cases {
		if got := isSuspectedComment("Shader", c.comment); got != c.want {
			t.Errorf("isSuspectedComment(Shader, %q) = %v, want %v", c.comment, got, c.want)
		}
	}
}

const structureSrc = `extends Node

signal hit

class_name Demo

@export var a = 1
@export_range(0, 10) var b = 2

var s = "func fake()"
# func fake2():

func foo():
	var x = 1
	if x:
		pass

static func bar(v):
	return v

var cb = func():
	pass

func tail():
	pass
`

func TestStructureAndFuncs(t *testing.T) {
	st := Analyze([]Unit{{Path: "a.gd", Language: "GDScript", Text: structureSrc}}, 10)
	if st.Structure != (Structure{Funcs: 3, Signals: 1, ClassNames: 1, Exports: 2}) {
		t.Errorf("structure = %+v", st.Structure)
	}
	byName := map[string]FuncInfo{}
	for _, f := range st.LongestFuncs {
		byName[f.Name] = f
	}
	if len(byName) != 3 {
		t.Fatalf("funcs = %+v", st.LongestFuncs)
	}
	if f := byName["foo"]; f.Length != 4 {
		t.Errorf("foo length = %d, want 4", f.Length)
	}
	if f := byName["bar"]; f.Length != 2 {
		t.Errorf("bar length = %d, want 2", f.Length)
	}
	if f := byName["tail"]; f.Length != 2 {
		t.Errorf("tail length = %d, want 2", f.Length)
	}
	if _, ok := byName["fake"]; ok {
		t.Error("string/comment func should not be counted")
	}
}

func TestFuncLengthSkipsBlankAndComment(t *testing.T) {
	src := "func big():\n\tvar a = 1\n\t# comment\n\n\tvar b = 2\n"
	st := Analyze([]Unit{{Path: "a.gd", Language: "GDScript", Text: src}}, 10)
	if len(st.LongestFuncs) != 1 || st.LongestFuncs[0].Length != 3 {
		t.Errorf("funcs = %+v, want single length 3", st.LongestFuncs)
	}
}

func TestInnerClassFunc(t *testing.T) {
	src := "class Inner:\n\tfunc m():\n\t\tpass\n\nfunc top():\n\tpass\n"
	st := Analyze([]Unit{{Path: "a.gd", Language: "GDScript", Text: src}}, 10)
	if st.Structure.Funcs != 2 {
		t.Errorf("funcs = %d, want 2", st.Structure.Funcs)
	}
	byName := map[string]int{}
	for _, f := range st.LongestFuncs {
		byName[f.Name] = f.Length
	}
	if byName["m"] != 2 || byName["top"] != 2 {
		t.Errorf("lengths = %+v", byName)
	}
}

func TestTopTruncation(t *testing.T) {
	units := []Unit{
		{Path: "b.gd", Language: "GDScript", Text: "func f():\n\tpass\n"},
		{Path: "a.gd", Language: "GDScript", Text: "func g():\n\tvar x = 1\n\treturn x\n"},
	}
	st := Analyze(units, 1)
	if len(st.LongestFiles) != 1 || st.LongestFiles[0].Path != "a.gd" {
		t.Errorf("longest files = %+v", st.LongestFiles)
	}
	if len(st.LongestFuncs) != 1 || st.LongestFuncs[0].Name != "g" {
		t.Errorf("longest funcs = %+v", st.LongestFuncs)
	}
}

func TestLanguageStats(t *testing.T) {
	src := "# func foo():\n# a plain note\n# TODO fix\nvar x = 1\n"
	shader := "// uniform float y;\n// a note\nvoid main() {}\n"
	st := Analyze([]Unit{
		{Path: "a.gd", Language: "GDScript", Text: src},
		{Path: "b.gdshader", Language: "Shader", Text: shader},
	}, 10)
	m := map[string]LangStat{}
	for _, l := range st.Languages {
		m[l.Language] = l
	}
	if gd := m["GDScript"]; gd.Comments != 3 || gd.Suspected != 1 {
		t.Errorf("GDScript = %+v", gd)
	}
	if sh := m["Shader"]; sh.Comments != 2 || sh.Suspected != 1 {
		t.Errorf("Shader = %+v", sh)
	}
	if r := m["GDScript"].Ratio(); r <= 0 || r >= 1 {
		t.Errorf("ratio = %v", r)
	}
}
