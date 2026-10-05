package counter

import "testing"

func TestCountGDScript(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Result
	}{
		{"comment only", "# hello\n", Result{Lines: 1, Comments: 1}},
		{"indented comment", "    # hello\n", Result{Lines: 1, Comments: 1}},
		{"doc comment", "## doc\n", Result{Lines: 1, Comments: 1, Doc: 1}},
		{"indented doc comment", "\t## doc\n", Result{Lines: 1, Comments: 1, Doc: 1}},
		{"triple hash is doc", "### doc\n", Result{Lines: 1, Comments: 1, Doc: 1}},
		{"single hash region", "#region\n#endregion\n", Result{Lines: 2, Comments: 2}},
		{"hash not doc", "#TODO\n", Result{Lines: 1, Comments: 1}},
		{"code trailing comment", "var x = 1 # set\n", Result{Lines: 1, Code: 1}},
		{"hash in double string", "var s = \"a#b\"\n", Result{Lines: 1, Code: 1}},
		{"hash in single string", "var s = 'a#b'\n", Result{Lines: 1, Code: 1}},
		{"hash in stringname", "var s = &\"a#b\"\n", Result{Lines: 1, Code: 1}},
		{"hash in nodepath", "var s = ^\"a#b\"\n", Result{Lines: 1, Code: 1}},
		{"hash in raw string", "var s = r\"a#b\"\n", Result{Lines: 1, Code: 1}},
		{"escaped quote double", "var s = \"a\\\"#b\"\n", Result{Lines: 1, Code: 1}},
		{"escaped quote single", "var s = 'it\\'s # x'\n", Result{Lines: 1, Code: 1}},
		{"single quote inside double", "var s = \"it's # x\"\n", Result{Lines: 1, Code: 1}},
		{"double quote inside single", "var s = 'say \"hi\" # x'\n", Result{Lines: 1, Code: 1}},
		{"raw escaped quote", "var s = r\"a\\\"#b\"\n", Result{Lines: 1, Code: 1}},
		{"triple same line double", "var s = \"\"\"a#b\"\"\"\n", Result{Lines: 1, Code: 1}},
		{"triple same line single", "var s = '''a#b'''\n", Result{Lines: 1, Code: 1}},
		{
			"triple multiline hash is code",
			"var s = \"\"\"\n# not comment\n\"\"\"\n",
			Result{Lines: 3, Code: 3},
		},
		{
			"triple internal blank",
			"var s = \"\"\"\nabc\n   \n\"\"\"\n",
			Result{Lines: 4, Code: 3, Blanks: 1},
		},
		{
			"triple close with trailing comment",
			"var s = \"\"\"\nabc\n\"\"\" # done\n",
			Result{Lines: 3, Code: 3},
		},
		{
			"triple blank then content",
			"\"\"\"\n\nabc\n\"\"\"\n",
			Result{Lines: 4, Code: 3, Blanks: 1},
		},
		{"empty file", "", Result{}},
		{"single newline", "\n", Result{Lines: 1, Blanks: 1}},
		{"whitespace only", "   \n", Result{Lines: 1, Blanks: 1}},
		{"bom only", "\ufeff", Result{}},
		{"bom then code", "\ufeffvar x = 1\n", Result{Lines: 1, Code: 1}},
		{"crlf", "var x = 1\r\n# c\r\n\r\n", Result{Lines: 3, Code: 1, Comments: 1, Blanks: 1}},
		{"no trailing newline code", "var x = 1", Result{Lines: 1, Code: 1}},
		{"no trailing newline comment", "# c", Result{Lines: 1, Comments: 1}},
		{"unclosed triple counts rest as code", "var s = \"\"\"\n# still code\n", Result{Lines: 2, Code: 2}},
		{"unclosed single does not carry", "var s = \"abc\ndef = 1\n", Result{Lines: 2, Code: 2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CountGDScript(c.in)
			if got != c.want {
				t.Errorf("CountGDScript(%q) = %+v, want %+v", c.in, got, c.want)
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
