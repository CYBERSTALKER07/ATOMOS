/**
 * Repair wire-desktop-portals fallout:
 * - default params `foo = {t("key")}` → keep EN default, resolve via t in body where needed
 * - insert missing `const t = usePortalT()` for files that call t() without the hook
 * - revert test files that incorrectly received t()
 */
import fs from "node:fs";
import path from "node:path";
import { flattenCatalog, readCatalog, repoRoot } from "./shared.mjs";

const enFlat = flattenCatalog(readCatalog("en"));

const apps = [
  "supplier-portal",
  "warehouse-portal",
  "factory-portal",
  "retailer-app-desktop",
];

function walkTsx(app) {
  const roots = [
    path.resolve(repoRoot, "apps", app, "app"),
    path.resolve(repoRoot, "apps", app, "components"),
  ];
  const files = [];
  for (const root of roots) {
    if (!fs.existsSync(root)) continue;
    const stack = [root];
    while (stack.length) {
      const dir = stack.pop();
      for (const ent of fs.readdirSync(dir, { withFileTypes: true })) {
        if (ent.name === "node_modules" || ent.name === ".next") continue;
        const full = path.join(dir, ent.name);
        if (ent.isDirectory()) stack.push(full);
        else if (ent.name.endsWith(".tsx")) files.push(full);
      }
    }
  }
  return files;
}

function enForKey(key) {
  return enFlat[key] ?? key;
}

function fixBrokenDefaults(source) {
  // name = {t("key")}  → name = "EN"
  return source.replace(
    /(\b[A-Za-z_][A-Za-z0-9_]*)\s+=\s+\{t\("([^"]+)"\)\}/g,
    (match, name, key) => {
      const en = enForKey(key).replace(/\\/g, "\\\\").replace(/"/g, '\\"');
      return `${name} = "${en}"`;
    },
  );
}

function ensureHook(source) {
  if (/\bconst\s+t\s*=\s*usePortalT\s*\(/.test(source)) return source;
  if (!/\bt\(["']/.test(source)) return source;

  // Match function signatures with nested parens limited; allow multiline params
  const patterns = [
    /export\s+default\s+function\s+[A-Za-z0-9_]+\s*\([\s\S]*?\)\s*(?::\s*[^{]+)?\{/,
    /export\s+function\s+[A-Za-z0-9_]+\s*\([\s\S]*?\)\s*(?::\s*[^{]+)?\{/,
    /function\s+[A-Za-z0-9_]+\s*\([\s\S]*?\)\s*(?::\s*[^{]+)?\{/,
  ];

  for (const pattern of patterns) {
    const match = pattern.exec(source);
    if (!match) continue;
    // Avoid matching too greedily across multiple functions — cap param length
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

function ensureImport(source) {
  if (/usePortalT/.test(source) && /from\s+["']@\/lib\/i18n["']/.test(source)) {
    return source;
  }
  if (!/\bt\(["']/.test(source) && !/usePortalT/.test(source)) return source;
  const useClient = source.match(/^["']use client["'];\s*/m);
  const insertAt = useClient ? useClient[0].length : 0;
  return (
    source.slice(0, insertAt) +
    `import { usePortalT } from "@/lib/i18n";\n` +
    source.slice(insertAt)
  );
}

function ensureUseClient(source) {
  if (/^["']use client["']/m.test(source)) return source;
  if (/^["']use server["']/m.test(source.trimStart())) return source;
  return `"use client";\n\n${source}`;
}

function revertTests(source) {
  // Replace t("key") with EN string literal in test files
  return source.replace(/\bt\("([^"]+)"\)/g, (m, key) => {
    const en = enForKey(key).replace(/\\/g, "\\\\").replace(/"/g, '\\"');
    return `"${en}"`;
  });
}

let fixedDefaults = 0;
let fixedHooks = 0;
let revertedTests = 0;

for (const app of apps) {
  for (const file of walkTsx(app)) {
    let source = fs.readFileSync(file, "utf8");
    const original = source;
    const isTest = file.includes(`${path.sep}__tests__${path.sep}`) || file.endsWith(".test.tsx");

    if (isTest && /\bt\(["']/.test(source)) {
      source = revertTests(source);
      // remove unused import/hook if present
      source = source.replace(
        /import\s*\{\s*usePortalT\s*\}\s*from\s*["']@\/lib\/i18n["'];\n?/g,
        "",
      );
      source = source.replace(/\n\s*const t = usePortalT\(\);\n?/g, "\n");
      if (source !== original) {
        revertedTests += 1;
        fs.writeFileSync(file, source);
      }
      continue;
    }

    if (/[A-Za-z_][A-Za-z0-9_]*\s+=\s+\{t\("/.test(source)) {
      source = fixBrokenDefaults(source);
      fixedDefaults += 1;
    }

    const needsHook =
      /\bt\(["']/.test(source) && !/\bconst\s+t\s*=\s*usePortalT\s*\(/.test(source);

    if (needsHook) {
      source = ensureUseClient(source);
      source = ensureImport(source);
      const before = source;
      source = ensureHook(source);
      if (source !== before && /\bconst\s+t\s*=\s*usePortalT\s*\(/.test(source)) {
        fixedHooks += 1;
      }
    }

    // PageChrome: after fixing default, ensure hook exists for renderError t()
    if (file.endsWith(`${path.sep}PageChrome.tsx`) && /\bt\(["']/.test(source)) {
      source = ensureUseClient(source);
      source = ensureImport(source);
      if (!/\bconst\s+t\s*=\s*usePortalT\s*\(/.test(source)) {
        source = ensureHook(source);
        fixedHooks += 1;
      }
    }

    if (source !== original) {
      fs.writeFileSync(file, source);
    }
  }
}

console.log(
  `Repaired defaults in ${fixedDefaults} files, inserted hooks in ${fixedHooks}, reverted ${revertedTests} tests.`,
);
