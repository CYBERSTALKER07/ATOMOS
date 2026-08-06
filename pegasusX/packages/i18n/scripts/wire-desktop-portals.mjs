/**
 * Mechanically wire desktop portal TSX to usePortalT using desktop-extract.json.
 *
 * For each file with replacements:
 *  - ensure "use client" if missing (hook requirement)
 *  - ensure `import { usePortalT } from "@/lib/i18n"`
 *  - ensure `const t = usePortalT()` inside the primary exported function component
 *  - replace JSX text nodes and selected string props with t("key")
 *
 * Usage: node packages/i18n/scripts/wire-desktop-portals.mjs [--app supplier-portal]
 */
import fs from "node:fs";
import path from "node:path";

import { generatedDir, repoRoot } from "./shared.mjs";

const args = process.argv.slice(2);
const appFilterIdx = args.indexOf("--app");
const appFilter = appFilterIdx >= 0 ? args[appFilterIdx + 1] : null;
const dryRun = args.includes("--dry-run");

const extract = JSON.parse(
  fs.readFileSync(path.resolve(generatedDir, "desktop-extract.json"), "utf8"),
);

const IMPORT_RE =
  /import\s*\{\s*([^}]*\busePortalT\b[^}]*)\}\s*from\s*["']@\/lib\/i18n["']/;
const USE_CLIENT_RE = /^["']use client["'];\s*/m;

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function ensureUseClient(source) {
  if (USE_CLIENT_RE.test(source)) return source;
  if (/^["']use server["']/.test(source.trimStart())) return source;
  return `"use client";\n\n${source}`;
}

function ensureImport(source) {
  if (IMPORT_RE.test(source)) return source;

  const useClientMatch = source.match(USE_CLIENT_RE);
  const insertAt = useClientMatch ? useClientMatch[0].length : 0;
  const importLine = `import { usePortalT } from "@/lib/i18n";\n`;
  return source.slice(0, insertAt) + importLine + source.slice(insertAt);
}

/**
 * Insert `const t = usePortalT();` after the opening of the first
 * exported function component body.
 */
function ensureHook(source) {
  if (/\bconst\s+t\s*=\s*usePortalT\s*\(/.test(source)) {
    return source;
  }

  // Allow multiline params; also React.FC / arrow components
  const patterns = [
    /export\s+default\s+function\s+[A-Za-z0-9_]+\s*\([\s\S]*?\)\s*(?::\s*[^{]+)?\{/,
    /export\s+default\s+function\s*\([\s\S]*?\)\s*(?::\s*[^{]+)?\{/,
    /export\s+function\s+[A-Za-z0-9_]+\s*\([\s\S]*?\)\s*(?::\s*[^{]+)?\{/,
    /function\s+[A-Za-z0-9_]+\s*\([\s\S]*?\)\s*(?::\s*[^{]+)?\{/,
    /export\s+const\s+[A-Za-z0-9_]+\s*(?::\s*[^=]+)?=\s*\([\s\S]*?\)\s*=>\s*\{/,
    /const\s+[A-Za-z0-9_]+\s*(?::\s*[^=]+)?=\s*\([\s\S]*?\)\s*=>\s*\{/,
  ];

  for (const pattern of patterns) {
    const match = pattern.exec(source);
    if (!match) continue;
    if (match[0].length > 2500) continue;
    const insertPos = match.index + match[0].length;
    return (
      source.slice(0, insertPos) +
      `\n  const t = usePortalT();` +
      source.slice(insertPos)
    );
  }

  return source;
}

function applyReplacements(source, replacements) {
  // Longer strings first to avoid partial overlaps
  const sorted = [...replacements].sort(
    (a, b) => b.text.length - a.text.length,
  );

  let next = source;
  let count = 0;

  for (const { text, key } of sorted) {
    const esc = escapeRegExp(text);

    // JSX text node: >Text<  (no nested tags)
    const jsxRe = new RegExp(`>(\\s*)${esc}(\\s*)<`, "g");
    const beforeJsx = next;
    next = next.replace(jsxRe, `>$1{t(${JSON.stringify(key)})}$2<`);
    if (next !== beforeJsx) count += 1;

    // Common string props
    const attrs = [
      "placeholder",
      "aria-label",
      "title",
      "subtitle",
      "label",
      "description",
      "headline",
      "body",
      "emptyMessage",
      "emptyTitle",
      "emptyDescription",
      "registerPrompt",
      "registerLabel",
      "alt",
      "helperText",
      "confirmLabel",
      "cancelLabel",
      "actionLabel",
      "primaryLabel",
      "secondaryLabel",
    ];
    for (const attr of attrs) {
      const attrRe = new RegExp(
        `(${attr}\\s*=\\s*)"${esc}"`,
        "g",
      );
      const before = next;
      next = next.replace(attrRe, `$1{t(${JSON.stringify(key)})}`);
      if (next !== before) count += 1;
    }

    // setError("...") / toast-like string literals when exact match
    const errRe = new RegExp(
      `(\\b(?:setError|setMessage|toast(?:\\.(?:success|error|info|warning))?|showToast)\\(\\s*)"${esc}"`,
      "g",
    );
    const beforeErr = next;
    next = next.replace(errRe, `$1t(${JSON.stringify(key)})`);
    if (next !== beforeErr) count += 1;

    // UI label fields in local arrays/objects (label/headline/title/description only)
    const labelRe = new RegExp(
      `(\\b(?:label|headline|title|description|emptyTitle|emptyDescription|message|helperText)\\s*:\\s*)"${esc}"`,
      "g",
    );
    const beforeLabel = next;
    next = next.replace(labelRe, `$1t(${JSON.stringify(key)})`);
    if (next !== beforeLabel) count += 1;
  }

  return { source: next, count };
}

let filesTouched = 0;
let replacementsApplied = 0;
let skipped = 0;

for (const wireFile of extract.wire_files) {
  if (appFilter && !wireFile.file.includes(`apps/${appFilter}/`)) {
    continue;
  }

  const abs = path.resolve(repoRoot, wireFile.file);
  if (!fs.existsSync(abs)) {
    skipped += 1;
    continue;
  }
  if (!/\.(tsx|jsx)$/.test(abs)) {
    skipped += 1;
    continue;
  }

  let source = fs.readFileSync(abs, "utf8");
  const { source: replaced, count } = applyReplacements(
    source,
    wireFile.replacements,
  );

  if (count === 0) {
    skipped += 1;
    continue;
  }

  let next = replaced;
  next = ensureUseClient(next);
  next = ensureImport(next);
  next = ensureHook(next);

  if (next !== source) {
    filesTouched += 1;
    replacementsApplied += count;
    if (!dryRun) {
      fs.writeFileSync(abs, next);
    }
  }
}

console.log(
  `${dryRun ? "[dry-run] " : ""}Wired ${filesTouched} files, ~${replacementsApplied} replacement groups, skipped ${skipped}.`,
);
