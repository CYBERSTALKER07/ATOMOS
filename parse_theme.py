import json

zed_theme = {
          "attribute": {
            "color": "#ff5050",
            "font_style": "italic",
            "font_weight": None
          },
          "editor.foreground": {
            "color": "#e0d8ee"
          },
          "boolean": {
            "color": "#ff8c00"
          },
          "comment": {
            "color": "#58b361",
            "font_style": "italic"
          },
          "comment.doc": {
            "color": "#dfe269",
            "font_style": "italic"
          },
          "constant": {
            "color": "#ff8c00"
          },
          "constructor": {
            "color": "#00ffff"
          },
          "embedded": {
            "color": "#e0d8ee"
          },
          "emphasis": {
            "color": "#cccc66"
          },
          "emphasis.strong": {
            "color": "#ff8c00"
          },
          "enum": {
            "color": "#d557d5"
          },
          "function": {
            "color": "#00ff7f",
            "font_weight": 600
          },
          "function.method": {
            "color": "#73e5ff"
          },
          "hint": {
            "color": "#00ffff"
          },
          "keyword": {
            "color": "#ff4da2"
          },
          "label": {
            "color": "#f0e8ff"
          },
          "link_text": {
            "color": "#ffff00"
          },
          "link_uri": {
            "color": "#00ffff"
          },
          "number": {
            "color": "#ff8c00"
          },
          "operator": {
            "color": "#ff00ff"
          },
          "predictive": {
            "color": "#9080bb"
          },
          "preproc": {
            "color": "#ff8c00"
          },
          "primary": {
            "color": "#e0d8ee00"
          },
          "punctuation": {
            "color": "#f0e8ff"
          },
          "property": {
            "color": "#00ffff"
          },
          "punctuation.bracket": {
            "color": "#e0d8eed4"
          },
          "punctuation.delimiter": {
            "color": "#ff00ff"
          },
          "punctuation.list_marker": {
            "color": "#00ffff"
          },
          "punctuation.special": {
            "color": "#ff00ff"
          },
          "string": {
            "color": "#ffff00"
          },
          "string.escape": {
            "color": "#ff00ff"
          },
          "string.regex": {
            "color": "#cc6699"
          },
          "string.special": {
            "color": "#ff00ff"
          },
          "string.special.symbol": {
            "color": "#00ffff"
          },
          "tag": {
            "color": "#ff00ff"
          },
          "text.literal": {
            "color": "#ffff00"
          },
          "title": {
            "color": "#00ffff"
          },
          "type": {
            "color": "#54ff98"
          },
          "type.builtin": {
            "color": "#54ff98"
          },
          "variable": {
            "color": "#f8deff"
          },
          "variable.special": {
            "color": "#00ffff"
          }
}

with open('void-theme/themes/theme.json', 'r') as f:
    vs_theme = json.load(f)

# Update Token Colors
tokens = vs_theme.get("tokenColors", [])

def update_or_add(scope, fg, style=None):
    for t in tokens:
        if t.get("scope") == scope or (isinstance(t.get("scope"), list) and scope in t.get("scope")):
            t["settings"]["foreground"] = fg
            if style:
                t["settings"]["fontStyle"] = style
            return
    new_t = {"name": scope, "scope": scope, "settings": {"foreground": fg}}
    if style:
        new_t["settings"]["fontStyle"] = style
    tokens.append(new_t)


update_or_add(["comment", "punctuation.definition.comment"], zed_theme['comment']['color'], "italic")
update_or_add("string", zed_theme['string']['color'])
update_or_add("constant.numeric", zed_theme['number']['color'])
update_or_add("constant.language", zed_theme['boolean']['color'], "italic")
update_or_add(["constant.character", "constant.other"], zed_theme['constant']['color'])
update_or_add("variable", zed_theme['variable']['color'])
update_or_add(["keyword", "storage.modifier", "storage.type"], zed_theme['keyword']['color'], "italic")
update_or_add(["entity.name.function", "support.function", "meta.function-call"], zed_theme['function']['color'], "bold")
update_or_add(["entity.name.type", "entity.name.class", "support.type"], zed_theme['type']['color'])
update_or_add(["keyword.operator"], zed_theme['operator']['color'])
update_or_add(["meta.object-literal.key", "support.type.property-name"], zed_theme['property']['color'])
update_or_add(["entity.other.attribute-name"], zed_theme['attribute']['color'], "italic")

vs_theme["colors"].update({
    "editor.background": "#250045",
    "editor.foreground": "#e0d8ee",
    "editorLineNumber.foreground": "#9080bb80",
    "editorLineNumber.activeForeground": "#66cccc",
    "editor.lineHighlightBackground": "#35005f44",
    "editor.wordHighlightBackground": "#cccc6642",
    "editor.wordHighlightStrongBackground": "#66cccc55",
    "statusBar.background": "#1a0030",
    "tab.activeBackground": "#35005f",
    "tab.inactiveBackground": "#2a004f00",
    "terminal.background": "#1a0030",
    "terminal.ansiBlack": "#1a0030",
    "terminal.ansiBlue": "#5599cc",
    "terminal.ansiBrightBlack": "#45006f",
    "terminal.ansiBrightBlue": "#66cccc",
    "terminal.ansiBrightCyan": "#66cccc",
    "terminal.ansiBrightGreen": "#66cc99",
    "terminal.ansiBrightMagenta": "#cc66cc",
    "terminal.ansiBrightRed": "#cc66cc",
    "terminal.ansiBrightWhite": "#e0d8ee",
    "terminal.ansiBrightYellow": "#cccc66",
    "terminal.ansiCyan": "#55aaaa",
    "terminal.ansiGreen": "#55aa80",
    "terminal.ansiMagenta": "#aa55aa",
    "terminal.ansiRed": "#cc6699",
    "terminal.ansiWhite": "#c0b8d0",
    "terminal.ansiYellow": "#ccaa55"
})

with open('void-theme/themes/theme.json', 'w') as f:
    json.dump(vs_theme, f, indent=2)

