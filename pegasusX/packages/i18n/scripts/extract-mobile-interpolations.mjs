/**
 * Extract localizable interpolations from mobile apps.
 *
 * Best practices:
 * - Catalog templates use named {placeholders} (translationContract)
 * - Skip data-only strings (no letters outside interpolations)
 * - Skip complex args (ternaries with embedded copy, closures, nested quotes)
 * - Keep arg expressions at call sites; pass as format args
 *
 * Usage: node packages/i18n/scripts/extract-mobile-interpolations.mjs
 * Writes: packages/i18n/generated/mobile-interpolation-extract.json
 */
import fs from "node:fs";
import path from "node:path";

import {
  flattenCatalog,
  generatedDir,
  readCatalog,
  repoRoot,
  writeFile,
} from "./shared.mjs";

const APP_DOMAIN = {
  "driver-app-android": "mobile_driver",
  "driver-app-ios": "mobile_driver",
  "factory-app-android": "mobile_factory",
  "factory-app-ios": "mobile_factory",
  "payload-app-android": "mobile_payload",
  "payload-app-ios": "mobile_payload",
  "retailer-app-android": "mobile_retailer",
  "retailer-app-ios": "mobile_retailer",
  "supplier-app-android": "mobile_supplier",
  "supplier-app-ios": "mobile_supplier",
  "warehouse-app-android": "mobile_warehouse",
  "warehouse-app-ios": "mobile_warehouse",
};

const IOS_CALLEES = [
  "Text",
  "Button",
  "Label",
  "navigationTitle",
  "Section",
  "SecureField",
  "TextField",
];
const ANDROID_PROPS = [
  "text",
  "label",
  "title",
  "contentDescription",
  "placeholder",
];

/**
 * Read a Swift `"…"` literal starting at openQuote, allowing nested quotes
 * inside `\(...)` interpolations.
 */
function readSwiftStringLiteral(source, openQuote) {
  let i = openQuote + 1;
  let out = "";
  let parenDepth = 0;
  while (i < source.length) {
    const ch = source[i];
    if (parenDepth === 0) {
      if (ch === "\\") {
        if (source.startsWith("\\(", i)) {
          out += "\\(";
          i += 2;
          parenDepth = 1;
          continue;
        }
        out += ch + (source[i + 1] ?? "");
        i += 2;
        continue;
      }
      if (ch === '"') return { literal: out, end: i + 1 };
      out += ch;
      i += 1;
      continue;
    }
    // inside \( ... )
    if (ch === "\\") {
      if (source.startsWith("\\(", i)) {
        out += "\\(";
        i += 2;
        parenDepth += 1;
        continue;
      }
      out += ch + (source[i + 1] ?? "");
      i += 2;
      continue;
    }
    if (ch === "(") {
      parenDepth += 1;
      out += ch;
      i += 1;
      continue;
    }
    if (ch === ")") {
      parenDepth -= 1;
      out += ch;
      i += 1;
      continue;
    }
    if (ch === '"' || ch === "'") {
      const q = ch;
      out += q;
      i += 1;
      while (i < source.length) {
        if (source[i] === "\\") {
          out += source[i] + (source[i + 1] ?? "");
          i += 2;
          continue;
        }
        out += source[i];
        if (source[i] === q) {
          i += 1;
          break;
        }
        i += 1;
      }
      continue;
    }
    out += ch;
    i += 1;
  }
  return null;
}

/**
 * Read a Kotlin `"…"` literal starting at openQuote, allowing nested quotes
 * inside `${…}` interpolations.
 */
function readKotlinStringLiteral(source, openQuote) {
  let i = openQuote + 1;
  let out = "";
  let braceDepth = 0;
  while (i < source.length) {
    const ch = source[i];
    if (braceDepth === 0) {
      if (ch === "\\") {
        out += ch + (source[i + 1] ?? "");
        i += 2;
        continue;
      }
      if (ch === '"') return { literal: out, end: i + 1 };
      if (ch === "$" && source[i + 1] === "{") {
        out += "${";
        i += 2;
        braceDepth = 1;
        continue;
      }
      out += ch;
      i += 1;
      continue;
    }
    if (ch === "\\") {
      out += ch + (source[i + 1] ?? "");
      i += 2;
      continue;
    }
    if (ch === "{") {
      braceDepth += 1;
      out += ch;
      i += 1;
      continue;
    }
    if (ch === "}") {
      braceDepth -= 1;
      out += ch;
      i += 1;
      continue;
    }
    if (ch === '"' || ch === "'") {
      const q = ch;
      out += q;
      i += 1;
      while (i < source.length) {
        if (source[i] === "\\") {
          out += source[i] + (source[i + 1] ?? "");
          i += 2;
          continue;
        }
        out += source[i];
        if (source[i] === q) {
          i += 1;
          break;
        }
        i += 1;
      }
      continue;
    }
    out += ch;
    i += 1;
  }
  return null;
}

function findHits(source, platform) {
  const hits = [];
  if (platform === "ios") {
    for (const callee of IOS_CALLEES) {
      const re = new RegExp(`\\b${callee}\\s*\\(\\s*"`, "g");
      for (const m of source.matchAll(re)) {
        const open = m.index + m[0].length - 1;
        const read = readSwiftStringLiteral(source, open);
        if (!read) continue;
        if (!read.literal.includes("\\(")) continue;
        hits.push({
          callee,
          literal: read.literal,
          index: m.index,
          full: source.slice(m.index, read.end),
        });
      }
    }
    return hits;
  }

  for (const prop of ANDROID_PROPS) {
    const re = new RegExp(`\\b${prop}\\s*=\\s*"`, "g");
    for (const m of source.matchAll(re)) {
      const open = m.index + m[0].length - 1;
      const read = readKotlinStringLiteral(source, open);
      if (!read) continue;
      if (!read.literal.includes("$")) continue;
      hits.push({
        callee: prop,
        literal: read.literal,
        index: m.index,
        full: source.slice(m.index, read.end),
        kind: "assign",
      });
    }
  }
  const textRe = /\bText\s*\(\s*"/g;
  for (const m of source.matchAll(textRe)) {
    const open = m.index + m[0].length - 1;
    const read = readKotlinStringLiteral(source, open);
    if (!read) continue;
    if (!read.literal.includes("$")) continue;
    hits.push({
      callee: "Text",
      literal: read.literal,
      index: m.index,
      full: source.slice(m.index, read.end),
      kind: "text",
    });
  }
  return hits;
}

const skipDirectories = new Set([
  ".git",
  ".build",
  "build",
  "DerivedData",
  "Pods",
  "node_modules",
  "generated",
]);

function walk(dir, files = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      if (skipDirectories.has(entry.name)) continue;
      walk(path.join(dir, entry.name), files);
      continue;
    }
    if (entry.isFile()) files.push(path.join(dir, entry.name));
  }
  return files;
}

function slugify(text) {
  return (
    text
      .normalize("NFKD")
      .replace(/['']/g, "")
      .replace(/\{([a-zA-Z0-9_]+)\}/g, "$1")
      .replace(/[^a-zA-Z0-9]+/g, "_")
      .replace(/^_+|_+$/g, "")
      .replace(/_+/g, "_")
      .toLowerCase()
      .slice(0, 72) || "interp"
  );
}

function setNested(tree, dottedKey, value) {
  const parts = dottedKey.split(".");
  let cursor = tree;
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i];
    if (!cursor[part] || typeof cursor[part] === "string") cursor[part] = {};
    cursor = cursor[part];
  }
  if (typeof cursor[parts[parts.length - 1]] === "undefined") {
    cursor[parts[parts.length - 1]] = value;
  }
}

/** Parse Swift `"…\(expr)…"` into static parts + arg expressions. */
function parseSwiftInterpolation(literal) {
  const staticParts = [];
  const args = [];
  let i = 0;
  let buf = "";
  while (i < literal.length) {
    if (literal.startsWith("\\(", i)) {
      staticParts.push(buf);
      buf = "";
      let depth = 1;
      let j = i + 2;
      const start = j;
      while (j < literal.length && depth > 0) {
        if (literal.startsWith("\\(", j)) {
          depth += 1;
          j += 2;
          continue;
        }
        const ch = literal[j];
        if (ch === "(") depth += 1;
        else if (ch === ")") {
          depth -= 1;
          if (depth === 0) {
            args.push(literal.slice(start, j));
            j += 1;
            break;
          }
        }
        j += 1;
      }
      if (depth !== 0) return null;
      i = j;
      continue;
    }
    buf += literal[i];
    i += 1;
  }
  staticParts.push(buf);
  return { staticParts, args };
}

/** Parse Kotlin `"…$x…" / "…${expr}…"`. */
function parseKotlinInterpolation(literal) {
  const staticParts = [];
  const args = [];
  let i = 0;
  let buf = "";
  while (i < literal.length) {
    if (literal.startsWith("${", i)) {
      staticParts.push(buf);
      buf = "";
      let depth = 1;
      let j = i + 2;
      const start = j;
      while (j < literal.length && depth > 0) {
        const ch = literal[j];
        if (ch === "{") depth += 1;
        else if (ch === "}") {
          depth -= 1;
          if (depth === 0) {
            args.push(literal.slice(start, j));
            j += 1;
            break;
          }
        }
        j += 1;
      }
      if (depth !== 0) return null;
      i = j;
      continue;
    }
    if (
      literal[i] === "$" &&
      i + 1 < literal.length &&
      /[A-Za-z_]/.test(literal[i + 1])
    ) {
      staticParts.push(buf);
      buf = "";
      let j = i + 1;
      while (j < literal.length && /[A-Za-z0-9_]/.test(literal[j])) j += 1;
      args.push(literal.slice(i + 1, j));
      i = j;
      continue;
    }
    buf += literal[i];
    i += 1;
  }
  staticParts.push(buf);
  return { staticParts, args };
}

function stripSwiftFormatClause(expr) {
  return expr.replace(/,\s*format:\s*\.[A-Za-z0-9_]+/g, "").trim();
}

function isComplexArg(expr, platform) {
  const a = platform === "ios" ? stripSwiftFormatClause(expr) : expr.trim();
  if (/\n/.test(a)) return true;
  // Embedded quoted UI copy (letterful). Punctuation-only quotes ('_', '—') are OK.
  for (const m of a.matchAll(/'([^']*)'|"([^"]*)"/g)) {
    const s = m[1] ?? m[2] ?? "";
    if (/[A-Za-zА-Яа-я]{2,}/.test(s)) return true;
  }
  // Bare ifBlank { id } / ternaries without UI copy are fine as format args.
  return false;
}

function nameArg(expr, used, platform) {
  let e = platform === "ios" ? stripSwiftFormatClause(expr) : expr.trim();
  e = e.replace(/^(?:Int|Double|Float|String|CGFloat|NSNumber)\((.*)\)$/s, "$1");
  e = e.replace(/\([^)]*\)/g, "");
  const parts = e.split(".").map((p) => p.trim()).filter(Boolean);
  let base = parts[parts.length - 1] || "value";
  base = base
    .replace(/[^a-zA-Z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .toLowerCase();
  if (!base) base = "value";
  if (/^\d/.test(base)) base = `n_${base}`;
  let name = base;
  let n = 2;
  while (used.has(name)) {
    name = `${base}_${n}`;
    n += 1;
  }
  used.add(name);
  return name;
}

function classify(parsed, platform) {
  if (!parsed || parsed.args.length === 0) return { kind: "none" };
  if (parsed.args.length > 4) return { kind: "too_many_args" };
  if (parsed.args.some((a) => isComplexArg(a, platform))) {
    return { kind: "complex_arg" };
  }
  const joinedStatic = parsed.staticParts.join("");
  // Pure value display: single arg, no static skeleton → skip (IDs / formatters).
  // Multi-arg or joiner punctuation still gets a template so locales can reorder.
  if (parsed.args.length === 1 && !/\S/.test(joinedStatic)) {
    return { kind: "data_only" };
  }

  const used = new Set();
  const argNames = parsed.args.map((a) => nameArg(a, used, platform));
  let template = "";
  for (let i = 0; i < parsed.args.length; i++) {
    template += parsed.staticParts[i] + `{${argNames[i]}}`;
  }
  template += parsed.staticParts[parsed.args.length] || "";
  if (template.length > 180) return { kind: "too_long" };
  return { kind: "simple", template, argNames, args: parsed.args };
}

function lineNumber(source, index) {
  return source.slice(0, index).split("\n").length;
}

const enFlat = flattenCatalog(readCatalog("en"));
const usedKeys = new Set(Object.keys(enFlat));
const valueToKeys = new Map();
for (const [key, value] of Object.entries(enFlat)) {
  if (!valueToKeys.has(value)) valueToKeys.set(value, []);
  valueToKeys.get(value).push(key);
}

function pickExisting(template, domain) {
  const keys = valueToKeys.get(template);
  if (!keys?.length) return null;
  const scored = [...keys].sort((a, b) => {
    const score = (k) => {
      if (k.startsWith(`${domain}.`)) return 0;
      if (k.startsWith("common.")) return 1;
      if (k.startsWith("portal.")) return 2;
      return 4;
    };
    return score(a) - score(b) || a.length - b.length || a.localeCompare(b);
  });
  return scored[0];
}

const proposals = [];
const reuse = [];
const skipped = [];
const byDomain = {};
for (const d of Object.values(APP_DOMAIN)) {
  byDomain[d] = { simple: 0, reuse: 0, skipped: 0 };
}

const appsRoot = path.resolve(repoRoot, "apps");
const files = walk(appsRoot).filter((f) => {
  const rel = path.relative(repoRoot, f);
  return (
    /apps\/[a-z-]+-app-(ios|android)\//.test(rel) &&
    (f.endsWith(".swift") || f.endsWith(".kt"))
  );
});

for (const abs of files) {
  const rel = path.relative(repoRoot, abs).replaceAll("\\", "/");
  const app = rel.split("/")[1];
  const domain = APP_DOMAIN[app];
  if (!domain) continue;
  const platform = app.endsWith("-ios") ? "ios" : "android";
  const source = fs.readFileSync(abs, "utf8");

  const hits = findHits(source, platform);

  for (const hit of hits) {
    const parsed =
      platform === "ios"
        ? parseSwiftInterpolation(hit.literal)
        : parseKotlinInterpolation(hit.literal);
    if (!parsed) {
      byDomain[domain].skipped += 1;
      skipped.push({
        file: rel,
        line: lineNumber(source, hit.index),
        reason: "unparsed",
        text: hit.literal.slice(0, 160),
        platform,
      });
      continue;
    }
    const c = classify(parsed, platform);
    if (c.kind !== "simple") {
      byDomain[domain].skipped += 1;
      skipped.push({
        file: rel,
        line: lineNumber(source, hit.index),
        reason: c.kind,
        text: hit.literal.slice(0, 160),
        platform,
      });
      continue;
    }

    const existing = pickExisting(c.template, domain);
    const entry = {
      file: rel,
      line: lineNumber(source, hit.index),
      platform,
      domain,
      callee: hit.callee,
      kind: hit.kind || "call",
      original: hit.literal,
      template: c.template,
      args: c.args,
      argNames: c.argNames,
      match: hit.full,
    };

    if (existing) {
      byDomain[domain].reuse += 1;
      reuse.push({ ...entry, key: existing });
      continue;
    }

    byDomain[domain].simple += 1;
    proposals.push(entry);
  }
}

/** Assign keys once per domain+template */
const templateToKey = new Map();
const deduped = [];
for (const p of proposals) {
  const tk = `${p.domain}::${p.template}`;
  if (!templateToKey.has(tk)) {
    let key = `${p.domain}.ui.${slugify(p.template)}`;
    let n = 2;
    while (usedKeys.has(key)) {
      key = `${p.domain}.ui.${slugify(p.template)}_${n}`;
      n += 1;
    }
    usedKeys.add(key);
    templateToKey.set(tk, key);
    deduped.push({ ...p, key });
  }
  p.key = templateToKey.get(tk);
}

const enAdditions = {};
for (const p of deduped) {
  setNested(enAdditions, p.key, p.template);
}

const wireMap = new Map();
function addWire(hit) {
  if (!wireMap.has(hit.file)) wireMap.set(hit.file, []);
  wireMap.get(hit.file).push({
    key: hit.key,
    original: hit.original,
    template: hit.template,
    args: hit.args,
    argNames: hit.argNames,
    platform: hit.platform,
    callee: hit.callee,
    kind: hit.kind || "call",
    match: hit.match,
  });
}
for (const hit of reuse) addWire(hit);
for (const hit of proposals) addWire(hit);

const wireFiles = [...wireMap.entries()].map(([file, replacements]) => ({
  file,
  replacements,
}));

const output = {
  generated_at: new Date().toISOString(),
  summary: byDomain,
  new_key_count: deduped.length,
  reuse_count: reuse.length,
  skipped_count: skipped.length,
  proposals: deduped,
  reuse,
  skipped: skipped.slice(0, 500),
  en_additions: enAdditions,
  wire_files: wireFiles,
};

writeFile(
  path.resolve(generatedDir, "mobile-interpolation-extract.json"),
  `${JSON.stringify(output, null, 2)}\n`,
);

console.log("Mobile interpolation extract:");
for (const [domain, s] of Object.entries(byDomain)) {
  console.log(
    `  ${domain}: simple=${s.simple} reuse=${s.reuse} skipped=${s.skipped}`,
  );
}
console.log(`  unique new keys: ${deduped.length}`);
console.log(`  wire files: ${wireFiles.length}`);
console.log(`  skipped (sample capped): ${skipped.length}`);
