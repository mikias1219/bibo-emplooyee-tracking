"""Regenerate packages/admin-ui/src/styles.css from the web admin theme.

Copies every rule from the web theme that targets a class used by the shared
admin screens, scoped under `.wi-admin` so it never restyles the host app.

usage (from repo root):
  python3 packages/admin-ui/scripts/extract-css.py apps/web-admin/src/shared/theme/theme.css \
      packages/admin-ui/src wi-admin > packages/admin-ui/src/styles.css
"""
import os, re, sys

theme, src_dir, scope = sys.argv[1], sys.argv[2], sys.argv[3]
css = re.sub(r"/\*.*?\*/", "", open(theme).read(), flags=re.S)

# classes used by the package source
used = set()
for root, _, files in os.walk(src_dir):
    for f in files:
        if f.endswith((".tsx", ".ts")):
            s = open(os.path.join(root, f)).read()
            for m in re.finditer(r'className=(?:"([^"]+)"|\{`([^`]+)`\}|\{[^}]*?"([^"]+)"[^}]*\})', s):
                txt = " ".join(g for g in m.groups() if g)
                for tok in re.findall(r"[a-zA-Z][\w-]*", re.sub(r"\$\{[^}]*\}", " ", txt)):
                    used.add(tok)
            # class names built in template strings like `notice-${kind}`, `bibo-tab${on}`
            for m in re.finditer(r"`([a-z][\w-]*?)-?\$\{", s):
                used.add(m.group(1))
used_re = re.compile(r"\.(" + "|".join(sorted(map(re.escape, used), key=len, reverse=True)) + r")(?![\w-])")
prefix_re = re.compile(r"\.(notice-|bibo-tab|bibo-seg|bibo-stat|bibo-card|admin-perf|set-)")


def parse(block):
    """Yield (prelude, body) for each top-level item."""
    i, n = 0, len(block)
    while i < n:
        j = block.find("{", i)
        if j < 0:
            return
        prelude = block[i:j].strip()
        depth, k = 1, j + 1
        while depth and k < n:
            if block[k] == "{":
                depth += 1
            elif block[k] == "}":
                depth -= 1
            k += 1
        yield prelude, block[j + 1 : k - 1]
        i = k


def scope_selector(sel):
    sel = sel.strip()
    m = re.match(r'^((?::root|html)(?:\[[^\]]*\])*|\[data-theme[^\]]*\])\s+(.*)$', sel)
    if m:
        return f"{m.group(1)} .{scope} {m.group(2)}"
    if sel in (":root", "html", "body") or sel.startswith(":root"):
        return None
    return f".{scope} {sel}"


def wanted(sel):
    return bool(used_re.search(sel) or prefix_re.search(sel))


anims = set()


def emit(block, indent=""):
    out = []
    for prelude, body in parse(block):
        if prelude.startswith("@media") or prelude.startswith("@supports"):
            inner = emit(body, indent + "  ")
            if inner.strip():
                out.append(f"{indent}{prelude} {{\n{inner}{indent}}}\n")
        elif prelude.startswith("@"):
            continue
        else:
            sels = [s for s in prelude.split(",") if wanted(s)]
            scoped = [x for x in (scope_selector(s) for s in sels) if x]
            if scoped:
                for a in re.findall(r"animation(?:-name)?\s*:\s*([\w-]+)", body):
                    anims.add(a)
                decls = "\n".join(indent + "  " + d.strip() + ";" for d in body.split(";") if d.strip())
                out.append(f"{indent}{(',\n' + indent).join(scoped)} {{\n{decls}\n{indent}}}\n")
    return "".join(out)


rules = emit(css)
frames = "".join(
    f"@keyframes {name} {{{body}}}\n"
    for prelude, body in parse(css)
    if prelude.startswith("@keyframes") and (name := prelude.split()[1]) in anims
)
print(f"/* Generated from apps/web-admin theme.css: rules for the shared admin screens,\n   scoped under .{scope} so they never leak into the host app. Tokens\n   (var(--…)) come from the host theme. */\n")
print(frames + rules, end="")
sys.stderr.write(f"classes used: {len(used)}; keyframes: {sorted(anims)}\n")
