package counter

import "testing"

func TestCountShader(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Result
	}{
		{"line comment", "// hi\n", Result{Lines: 1, Comments: 1}},
		{"indented line comment", "\t// hi\n", Result{Lines: 1, Comments: 1}},
		{"block single line", "/* hi */\n", Result{Lines: 1, Comments: 1}},
		{"block multiline", "/*\nhi\n*/\n", Result{Lines: 3, Comments: 3}},
		{"block internal blank", "/*\na\n\nb\n*/\n", Result{Lines: 5, Comments: 4, Blanks: 1}},
		{"code then block open", "x = 1; /* c\nmore */\n", Result{Lines: 2, Code: 1, Comments: 1}},
		{"block close then code", "/* c\n*/ x = 1;\n", Result{Lines: 2, Comments: 1, Code: 1}},
		{"line comment hides block open", "// c /* not block\nx = 1;\n", Result{Lines: 2, Comments: 1, Code: 1}},
		{"block hides line comment", "/*\n// not line comment\n*/\n", Result{Lines: 3, Comments: 3}},
		{"block not nested", "/* /* */\n", Result{Lines: 1, Comments: 1}},
		{"block not nested then code", "/* /* */ code\n", Result{Lines: 1, Code: 1}},
		{"stray close is code", "*/ x = 1\n", Result{Lines: 1, Code: 1}},
		{"code trailing line comment", "x = 1; // c\n", Result{Lines: 1, Code: 1}},
		{"doc single line", "/** doc */\n", Result{Lines: 1, Comments: 1, Doc: 1}},
		{"doc indented", "   /** doc */\n", Result{Lines: 1, Comments: 1, Doc: 1}},
		{"doc multiline", "/**\n * doc\n */\n", Result{Lines: 3, Comments: 3, Doc: 3}},
		{"doc with internal blank", "/**\n\n*/\n", Result{Lines: 3, Comments: 2, Doc: 2, Blanks: 1}},
		{"empty doc is not doc", "/**/\n", Result{Lines: 1, Comments: 1}},
		{"normal block is not doc", "/* */\n", Result{Lines: 1, Comments: 1}},
		{"doc then code same line", "/** doc */ x = 1\n", Result{Lines: 1, Code: 1}},
		{"preprocessor is code", "#include \"res://a.gdshaderinc\"\n#define X 1\n#ifdef FOO\n#endif\n", Result{Lines: 4, Code: 4}},
		{"string hides line comment", "#include \"res://a//b.gdshaderinc\"\n", Result{Lines: 1, Code: 1}},
		{"string hides block comment", "uniform int x : hint_enum(\"a//b\", \"c/*d\");\n", Result{Lines: 1, Code: 1}},
		{"escaped quote in string", "foo(\"a\\\"//b\")\n", Result{Lines: 1, Code: 1}},
		{"unclosed block to eof", "/* comment\nstill comment\n", Result{Lines: 2, Comments: 2}},
		{"unclosed string", "foo(\"abc\n", Result{Lines: 1, Code: 1}},
		{"empty file", "", Result{}},
		{"crlf", "// c\r\nx = 1\r\n\r\n", Result{Lines: 3, Code: 1, Comments: 1, Blanks: 1}},
		{"no trailing newline", "x = 1", Result{Lines: 1, Code: 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CountShader(c.in)
			if got != c.want {
				t.Errorf("CountShader(%q) = %+v, want %+v", c.in, got, c.want)
			}
			if got.Lines != got.Code+got.Comments+got.Blanks {
				t.Errorf("invariant broken: Lines=%d != Code+Comments+Blanks=%d", got.Lines, got.Code+got.Comments+got.Blanks)
			}
			if got.Doc > got.Comments {
				t.Errorf("Doc=%d > Comments=%d", got.Doc, got.Comments)
			}
		})
	}
}
