/**
 * Wire Android Compose/XML hardcoded strings to stringResource(R.string.*).
 * Uses mobile-extract.json; Android resource names from dotted keys via toAndroidName.
 *
 * Usage: node packages/i18n/scripts/wire-mobile-android.mjs
 */
import fs from "node:fs";
import path from "node:path";

import { generatedDir, repoRoot, toAndroidName } from "./shared.mjs";

const extract = JSON.parse(
  fs.readFileSync(path.resolve(generatedDir, "mobile-extract.json"), "utf8"),
);

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function ensureStringResourceImport(source) {
  if (/stringResource\s*\(/.test(source) && /androidx\.compose\.ui\.res\.stringResource/.test(source)) {
    return source;
  }
  if (!/stringResource\s*\(/.test(source) && !/R\.string\./.test(source)) {
    // will add when we replace
  }
  if (/import\s+androidx\.compose\.ui\.res\.stringResource/.test(source)) {
    return source;
  }
  // Insert after package / other imports
  const pkg = source.match(/^package\s+[^\n]+\n/);
  if (!pkg) return source;
  return (
    source.slice(0, pkg[0].length) +
    "\nimport androidx.compose.ui.res.stringResource\n" +
    source.slice(pkg[0].length)
  );
}

function ensureRImport(source, filePath) {
  if (/\bimport\s+[\w.]+.R\b/.test(source) || /\bcom\.pegasusx\.\w+\.R\b/.test(source)) {
    // may already use R without import in same package
  }
  // Derive package R from path namespace heuristically — apps use R from app package automatically
  return source;
}

let filesTouched = 0;
let replacements = 0;

for (const wireFile of extract.wire_files) {
  if (!wireFile.file.includes("-android/")) continue;
  if (!/\.(kt|xml)$/.test(wireFile.file)) continue;

  const abs = path.resolve(repoRoot, wireFile.file);
  if (!fs.existsSync(abs)) continue;

  let source = fs.readFileSync(abs, "utf8");
  const orig = source;
  const isXml = abs.endsWith(".xml");
  const isKt = abs.endsWith(".kt");

  const sorted = [...wireFile.replacements].sort(
    (a, b) => b.text.length - a.text.length,
  );

  for (const { text, key } of sorted) {
    const androidName = toAndroidName(key);
    const esc = escapeRegExp(text);

    if (isKt) {
      // text = "Foo"  or  label = "Foo" etc. in compose
      const patterns = [
        // text = "Literal"
        new RegExp(`(\\btext\\s*=\\s*)"${esc}"`, "g"),
        new RegExp(`(\\blabel\\s*=\\s*)"${esc}"`, "g"),
        new RegExp(`(\\btitle\\s*=\\s*)"${esc}"`, "g"),
        new RegExp(`(\\bcontentDescription\\s*=\\s*)"${esc}"`, "g"),
        new RegExp(`(\\bplaceholder\\s*=\\s*)"${esc}"`, "g"),
        // Text("Literal") / Button("Literal") style if any
        new RegExp(`(\\bText\\s*\\(\\s*)"${esc}"`, "g"),
        // string literals passed to snackbar etc: common pattern "Foo" as sole arg in some calls — skip aggressive
      ];

      for (const re of patterns) {
        const before = source;
        source = source.replace(
          re,
          `$1stringResource(R.string.${androidName})`,
        );
        if (source !== before) replacements += 1;
      }
    }

    if (isXml) {
      const re = new RegExp(
        `(android:(?:text|contentDescription|hint|title)\\s*=\\s*)"${esc}"`,
        "g",
      );
      const before = source;
      source = source.replace(re, `$1"@string/${androidName}"`);
      if (source !== before) replacements += 1;
    }
  }

  if (source !== orig && isKt) {
    source = ensureStringResourceImport(source);
    source = ensureRImport(source, abs);
  }

  if (source !== orig) {
    fs.writeFileSync(abs, source);
    filesTouched += 1;
  }
}

console.log(
  `Android wire: ${filesTouched} files, ~${replacements} replacement groups.`,
);
