#!/usr/bin/env python3
"""Generate the fake Godot 4 projects used by the gdloc skill evals.

Deterministic (fixed seed), so eval results stay comparable between iterations.
The generated trees never enter this repo: the evals copy them into a temp dir
where the subagent under test can see them.

Layout produced under the target directory:

  windup/                     project.godot in src/ -> tests the "project root is
    src/                      a subdirectory" trap; project name WindupWonderland
      project.godot
      main.gd, ...            own GDScript
      shaders/*.gdshader
      scenes/*.tscn           one with embedded GDScript, one with embedded Shader
      resources/*.tres
      addons/
        ww_water/             plugin.cfg -> name "WW Water", version 0.1.0
        gdquest_lib/          no plugin.cfg -> falls back to the directory name
        broken_plugin/        plugin.cfg with no [plugin] section -> warns
        docs/                 addons/ child with no code at all
  astral/                     project.godot at the root, smaller, for comparisons

Usage: python make_fixtures.py <output-dir>
"""

import os
import random
import sys

SEED = 20240117


def write(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="\n") as fh:
        fh.write(text)


def gd_script(rng, class_name=None, extends="Node", funcs=3, doc=True, messy=False):
    """Build a plausible GDScript file with doc comments, signals and exports."""
    out = []
    if doc:
        out.append("## %s" % class_name if class_name else "## A gameplay helper.")
        out.append("##")
        out.append("## Handles state that the scene tree should not know about.")
    if class_name:
        out.append("class_name %s" % class_name)
    out.append("extends %s" % extends)
    out.append("")
    out.append("signal changed(value)")
    out.append("signal finished")
    out.append("")
    out.append("@export var speed: float = %.1f" % (rng.uniform(1, 40),))
    out.append("@export_range(0, 100) var weight: int = %d" % rng.randint(1, 99))
    out.append("")
    out.append("const MAX_RETRIES := %d" % rng.randint(2, 8))
    out.append("")
    out.append("var _state: int = 0")
    out.append("")
    out.append("")
    for i in range(funcs):
        name = "step_%d" % i if i else "_ready"
        out.append("func %s() -> void:" % name)
        if i == 0:
            out.append("\t_state = 0")
            out.append("\tset_process(true)")
        else:
            out.append("\tvar total := 0.0")
            out.append("\tfor node in get_children():")
            out.append("\t\tif node is Node2D:")
            out.append("\t\t\ttotal += node.position.length()")
            out.append("\tif total > MAX_RETRIES:")
            out.append("\t\t_state += 1")
            out.append("\t\tchanged.emit(_state)")
            out.append("\t\treturn")
            out.append("\t_state = int(total)")
        out.append("")
        out.append("")
    out.append("func describe() -> String:")
    out.append('\t## 文档字符串不是注释，这里用普通字符串演示引号里的 # 不算注释。')
    out.append('\treturn "state #%d at speed %.1f" % [_state, speed]')
    if messy:
        out.append("")
        out.append("")
        out.append("# TODO: 这段以后要拆出去")
        out.append("# var old_timer = 0.0")
        out.append("# if old_timer > 1.0:")
        out.append("# \tqueue_free()")
    out.append("")
    return "\n".join(out)


def shader(rng, kind="canvas_item"):
    return "\n".join(
        [
            "shader_type %s;" % kind,
            "",
            "// A small effect used by the prototype.",
            "",
            "uniform vec4 tint : source_color = vec4(1.0);",
            "uniform float strength : hint_range(0.0, 1.0) = %.2f;" % rng.uniform(0.1, 0.9),
            "",
            "/**",
            " * Darkens the edges of the sprite.",
            " */",
            "void fragment() {",
            "\tvec2 uv = UV - vec2(0.5);",
            "\tfloat d = length(uv) * strength;",
            "\tCOLOR = mix(COLOR, tint, d);",
            "}",
            "",
        ]
    )


def tscn_with_embedded_script(rng):
    return "\n".join(
        [
            '[gd_scene load_steps=2 format=3 uid="uid://bwindup0001"]',
            "",
            '[sub_resource type="GDScript" id="GDScript_embedded1"]',
            'script/source = "extends Node\\n\\n## 内嵌脚本\\nvar hp := 10\\n\\nfunc _ready():\\n\\tprint(hp)\\n"',
            "",
            '[node name="Embedded" type="Node"]',
            'script = SubResource("GDScript_embedded1")',
            "",
        ]
    )


def tscn_with_embedded_shader(rng):
    return "\n".join(
        [
            '[gd_scene load_steps=2 format=3 uid="uid://bwindup0002"]',
            "",
            '[sub_resource type="Shader" id="Shader_embedded1"]',
            'code = "shader_type canvas_item;\\n\\n// inline\\nuniform float k = 1.0;\\n\\nvoid fragment() {\\n\\tCOLOR = vec4(k);\\n}\\n"',
            "",
            '[node name="Quad" type="ColorRect"]',
            'material = SubResource("Shader_embedded1")',
            "",
        ]
    )


def plain_tscn(name, nodes=6):
    # A fixed uid: Python's str hash is randomized per process, and a fixture
    # that differs between runs makes eval results hard to compare.
    uid = "uid://bwindup%04d" % (abs(sum(ord(c) for c in name)) % 10000)
    out = ["[gd_scene format=3 uid=\"%s\"]" % uid, ""]
    for i in range(nodes):
        out.append('[node name="Node%d" type="Node2D" parent="."]' % i)
        out.append("position = Vector2(%d, %d)" % (i * 16, i * 8))
        out.append("")
    return "\n".join(out)


def build_windup(root, rng):
    p = os.path.join(root, "windup", "src")
    write(
        os.path.join(p, "project.godot"),
        "\n".join(
            [
                "; Engine configuration file.",
                "config_version=5",
                "",
                "[application]",
                "",
                'config/name="WindupWonderland"',
                'config/features=PackedStringArray("4.4", "Forward Plus")',
                "",
            ]
        ),
    )
    write(os.path.join(p, "main.gd"), gd_script(rng, "Main", "Node2D", funcs=4))
    write(os.path.join(p, "player", "puppet.gd"), gd_script(rng, "Puppet", "CharacterBody2D", funcs=12, messy=True))
    write(os.path.join(p, "player", "puppet_state.gd"), gd_script(rng, "PuppetState", "RefCounted", funcs=9))
    write(os.path.join(p, "ui", "hud.gd"), gd_script(rng, "Hud", "CanvasLayer", funcs=5))
    write(os.path.join(p, "ui", "menu.gd"), gd_script(rng, None, "Control", funcs=4, messy=True))
    write(os.path.join(p, "world", "level.gd"), gd_script(rng, "Level", "Node2D", funcs=14))
    write(os.path.join(p, "world", "spawner.gd"), gd_script(rng, None, "Node", funcs=6))
    write(os.path.join(p, "autoload", "game_state.gd"), gd_script(rng, "GameState", "Node", funcs=7, doc=False))
    write(os.path.join(p, "tools", "debug_overlay.gd"), gd_script(rng, None, "CanvasLayer", funcs=3, doc=False))

    for name in ("displace", "outline", "dissolve"):
        write(os.path.join(p, "shaders", "%s.gdshader" % name), shader(rng))
    write(
        os.path.join(p, "shaders", "common.gdshaderinc"),
        "\n".join(["// shared helpers", "float ease_out(float t) {", "\treturn 1.0 - pow(1.0 - t, 3.0);", "}", ""]),
    )

    write(os.path.join(p, "scenes", "main.tscn"), plain_tscn("main", 8))
    write(os.path.join(p, "scenes", "embedded_script.tscn"), tscn_with_embedded_script(rng))
    write(os.path.join(p, "scenes", "embedded_shader.tscn"), tscn_with_embedded_shader(rng))
    write(
        os.path.join(p, "resources", "theme.tres"),
        "\n".join(['[gd_resource type="Theme" format=3]', "", "[resource]", "default_font_size = 16", ""]),
    )
    write(
        os.path.join(p, "resources", "shared_shader.tres"),
        "\n".join(
            [
                '[gd_resource type="Shader" load_steps=2 format=3]',
                "",
                '[sub_resource type="Shader" id="Shader_main"]',
                'code = "shader_type canvas_item;\\n// main resource shader\\nuniform float a = 1.0;\\nvoid fragment() {\\n\\tCOLOR = vec4(a);\\n}\\n"',
                "",
                "[resource]",
                'shader = SubResource("Shader_main")',
                "",
            ]
        ),
    )
    # Not counted, but must show up as a note.
    write(os.path.join(p, "tools", "EditorPlugin.cs"), "// C# is not counted by gdloc\n")

    addons = os.path.join(p, "addons")
    write(
        os.path.join(addons, "ww_water", "plugin.cfg"),
        "\n".join(["[plugin]", 'name="WW Water"', 'description="Water tools"', 'version="0.1.0"', 'script="plugin.gd"', ""]),
    )
    write(os.path.join(addons, "ww_water", "plugin.gd"), gd_script(rng, None, "EditorPlugin", funcs=2, doc=False))
    for name, funcs in (("river.gd", 16), ("baker.gd", 11), ("flow_map.gd", 13)):
        write(os.path.join(addons, "ww_water", name), gd_script(rng, None, "Node", funcs=funcs, doc=False))
    write(os.path.join(addons, "ww_water", "water.gdshader"), shader(rng, "spatial"))

    for name, funcs in (("utils.gd", 8), ("curve.gd", 10), ("tween_ext.gd", 7)):
        write(os.path.join(addons, "gdquest_lib", name), gd_script(rng, None, "RefCounted", funcs=funcs, doc=False))

    write(os.path.join(addons, "broken_plugin", "plugin.cfg"), "name = no section here\n")
    write(os.path.join(addons, "broken_plugin", "run.gd"), gd_script(rng, None, "Node", funcs=2, doc=False))
    write(os.path.join(addons, "docs", "notes.md"), "# Not code\n")

    # A subdirectory that Godot itself would skip.
    write(os.path.join(p, "vendor", ".gdignore"), "")
    write(os.path.join(p, "vendor", "ignored.gd"), gd_script(rng, None, "Node", funcs=40, doc=False))
    write(os.path.join(p, "vendor", "kept.gd"), gd_script(rng, None, "Node", funcs=2, doc=False))

    # .gitignore lives at the scan root (src/) in this layout.
    write(os.path.join(p, ".gitignore"), "\n".join(["/tmp_build/", "*.generated.gd", ""]))
    write(os.path.join(p, "tmp_build", "junk.gd"), gd_script(rng, None, "Node", funcs=30, doc=False))
    write(os.path.join(p, "world", "level.generated.gd"), gd_script(rng, None, "Node", funcs=20, doc=False))
    write(os.path.join(p, ".godot", "cache.gd"), gd_script(rng, None, "Node", funcs=30, doc=False))


def build_astral(root, rng):
    p = os.path.join(root, "astral")
    write(
        os.path.join(p, "project.godot"),
        "\n".join(["[application]", "", 'config/name="astral-mason"', ""]),
    )
    write(os.path.join(p, "core", "grid.gd"), gd_script(rng, "Grid", "Node2D", funcs=9))
    write(os.path.join(p, "core", "cell.gd"), gd_script(rng, "Cell", "RefCounted", funcs=4))
    write(os.path.join(p, "game.gd"), gd_script(rng, "Game", "Node", funcs=5))
    write(os.path.join(p, "shaders", "starfield.gdshader"), shader(rng))
    write(os.path.join(p, "scenes", "game.tscn"), plain_tscn("astral", 4))
    # Deliberately no addons/ directory: a project without plugins.
    write(os.path.join(p, "README.md"), "# astral-mason\nA tiny puzzle toy.\n")


def main():
    if len(sys.argv) != 2:
        print(__doc__)
        return 1
    root = os.path.abspath(sys.argv[1])
    rng = random.Random(SEED)
    build_windup(root, rng)
    build_astral(root, rng)
    print("fixtures written to %s" % root)
    return 0


if __name__ == "__main__":
    sys.exit(main())
