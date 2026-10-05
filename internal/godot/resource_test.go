package godot

import (
	"os"
	"path/filepath"
	"testing"
)

func parseSample(t *testing.T, name string) ([]EmbeddedBlock, int) {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "embedded", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", path, err)
	}
	return ParseResource(string(data))
}

func TestParseEmbeddedScript(t *testing.T) {
	blocks, vs := parseSample(t, "embedded_script.tscn")
	if vs != 0 {
		t.Errorf("visualShaders = %d, want 0", vs)
	}
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(blocks))
	}
	b := blocks[0]
	if b.Language != "GDScript" || b.ID != "GDScript_qr1jj" {
		t.Fatalf("block = %+v", b)
	}
	want := "## Doc comment for the embedded class.\n" +
		"# A regular comment.\n" +
		"extends Node\n" +
		"\n" +
		"const GREETING := \"say \\\"hi\\\" C:\\\\path\"\n" +
		"\n" +
		"\n" +
		"func _ready() -> void:\n" +
		"\tvar text := \"\"\"\n" +
		"\tmulti line one\n" +
		"\tmulti line two\n" +
		"\t\"\"\"\n" +
		"\tprint(GREETING)\n" +
		"\tprint(text)\n"
	if b.Source != want {
		t.Errorf("source mismatch:\n got %q\nwant %q", b.Source, want)
	}
}

func TestParseEmbeddedShader(t *testing.T) {
	blocks, _ := parseSample(t, "embedded_shader.tscn")
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(blocks))
	}
	b := blocks[0]
	if b.Language != "Shader" || b.ID != "Shader_4rye0" {
		t.Fatalf("block = %+v", b)
	}
	if b.Source != shaderSampleSource {
		t.Errorf("source mismatch:\n got %q\nwant %q", b.Source, shaderSampleSource)
	}
}

const shaderSampleSource = "/**\n" +
	" * Doc comment for the shader.\n" +
	" */\n" +
	"shader_type canvas_item;\n" +
	"\n" +
	"// A line comment.\n" +
	"uniform int mode : hint_enum(\"A\", \"B\") = 0;\n" +
	"\n" +
	"void fragment() {\n" +
	"\tCOLOR = vec4(1.0, 0.0, 0.0, 1.0);\n" +
	"}\n"

func TestParseShaderResource(t *testing.T) {
	blocks, vs := parseSample(t, "shader_resource.tres")
	if vs != 0 {
		t.Errorf("visualShaders = %d, want 0", vs)
	}
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(blocks))
	}
	b := blocks[0]
	if b.Language != "Shader" || b.ID != "[resource]" {
		t.Fatalf("block = %+v", b)
	}
	if b.Source != shaderSampleSource {
		t.Errorf("source mismatch:\n got %q\nwant %q", b.Source, shaderSampleSource)
	}
}

func TestParseEscapedString(t *testing.T) {
	blocks, _ := parseSample(t, "embedded_escaped.tscn")
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(blocks))
	}
	want := "## doc\nextends Node\nvar s = \"a\\\"b\\\\c\"\n"
	if blocks[0].Source != want {
		t.Errorf("source = %q, want %q", blocks[0].Source, want)
	}
}

func TestParseMultiBlocks(t *testing.T) {
	blocks, _ := parseSample(t, "multi_blocks.tscn")
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want 2", len(blocks))
	}
	if blocks[0].Language != "GDScript" || blocks[0].ID != "GDScript_m1" {
		t.Errorf("block0 = %+v", blocks[0])
	}
	if blocks[1].Language != "Shader" || blocks[1].ID != "Shader_m2" {
		t.Errorf("block1 = %+v", blocks[1])
	}
}

func TestParseVisualShader(t *testing.T) {
	blocks, vs := parseSample(t, "visual_shader.tscn")
	if len(blocks) != 0 {
		t.Errorf("blocks = %d, want 0", len(blocks))
	}
	if vs != 1 {
		t.Errorf("visualShaders = %d, want 1", vs)
	}
}

func TestParsePlainScene(t *testing.T) {
	blocks, vs := parseSample(t, "plain.tscn")
	if len(blocks) != 0 || vs != 0 {
		t.Errorf("blocks=%d vs=%d, want 0,0", len(blocks), vs)
	}
}

func TestParseBroken(t *testing.T) {
	// 不得 panic；能取到未闭合的块或取不到都可接受。
	blocks, vs := parseSample(t, "broken.tscn")
	_ = blocks
	if vs != 0 {
		t.Errorf("visualShaders = %d, want 0", vs)
	}
}
