"""Render an SGR-coloured terminal capture (tmux capture-pane -e) to SVG.

Every character is placed on its own cell so fallback fonts (braille, box
drawing) can't drift the columns.
"""
import html, re, sys

PALETTE = {  # dark theme, ANSI 0-15
    0: "#1e2030", 1: "#ed8796", 2: "#a6da95", 3: "#eed49f", 4: "#8aadf4",
    5: "#c6a0f6", 6: "#8bd5ca", 7: "#cad3f5", 8: "#6e738d", 9: "#ed8796",
    10: "#a6da95", 11: "#eed49f", 12: "#8aadf4", 13: "#c6a0f6", 14: "#8bd5ca",
    15: "#f4f6ff",
}
BG, FG = "#24273a", "#cad3f5"
CW, RH, FS, PAD = 8.4, 17, 14, 20
SGR = re.compile(r"\x1b\[([0-9;]*)m")


def runs(line):
    fg, bold, col, pos = None, False, 0, 0
    for m in list(SGR.finditer(line)) + [None]:
        text = line[pos:m.start() if m else len(line)]
        if text:
            yield col, text, fg, bold
            col += len(text)
        if not m:
            break
        pos = m.end()
        for code in [int(c) for c in (m.group(1) or "0").split(";") if c]:
            if code == 0:
                fg, bold = None, False
            elif code == 1:
                bold = True
            elif code == 22:
                bold = False
            elif code == 39:
                fg = None
            elif 30 <= code <= 37:
                fg = code - 30
            elif 90 <= code <= 97:
                fg = code - 90 + 8


def main(src, dst):
    lines = open(src, encoding="utf-8").read().rstrip("\n").split("\n")
    cols = max(len(SGR.sub("", l)) for l in lines)
    w, h = cols * CW + 2 * PAD, len(lines) * RH + 2 * PAD
    out = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{w:.0f}" height="{h:.0f}" viewBox="0 0 {w:.0f} {h:.0f}">',
        f'<rect width="100%" height="100%" rx="8" fill="{BG}"/>',
        f'<g font-family="ui-monospace, \'DejaVu Sans Mono\', Menlo, Consolas, monospace" font-size="{FS}" xml:space="preserve">',
    ]
    for row, line in enumerate(lines):
        y = PAD + row * RH + RH * 0.78
        for col, text, fg, bold in runs(line):
            if not text.strip():
                continue
            xs = " ".join(f"{PAD + (col + i) * CW:.1f}" for i in range(len(text)))
            attrs = f'fill="{PALETTE[fg] if fg is not None else FG}"'
            if bold:
                attrs += ' font-weight="bold"'
            out.append(f'<text x="{xs}" y="{y:.1f}" {attrs}>{html.escape(text)}</text>')
    out.append("</g></svg>")
    open(dst, "w", encoding="utf-8").write("\n".join(out) + "\n")


if __name__ == "__main__":
    main(*sys.argv[1:3])
