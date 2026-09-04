/**
 * Wire iOS SwiftUI inline strings to localization keys.
 * Text("English") -> Text("dotted.key") where Localizable.strings has "dotted.key".
 *
 * Usage: node packages/i18n/scripts/wire-mobile-ios.mjs
 */
import fs from "node:fs";
import path from "node:path";

import { generatedDir, repoRoot } from "./shared.mjs";

const extract = JSON.parse(
  fs.readFileSync(path.resolve(generatedDir, "mobile-extract.json"), "utf8"),
);

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

let filesTouched = 0;
let replacements = 0;

for (const wireFile of extract.wire_files) {
  if (!wireFile.file.includes("-ios/")) continue;
  if (!wireFile.file.endsWith(".swift")) continue;

  const abs = path.resolve(repoRoot, wireFile.file);
  if (!fs.existsSync(abs)) continue;

  let source = fs.readFileSync(abs, "utf8");
  const orig = source;

  const sorted = [...wireFile.replacements].sort(
    (a, b) => b.text.length - a.text.length,
  );

  for (const { text, key } of sorted) {
    if (text.includes("\\(")) continue; // keep interpolations
    const esc = escapeRegExp(text);
    // Text("..."), Button("..."), Label("...", ...), navigationTitle("..."),
    // TextField("...", ...), Section("...")
    const patterns = [
      new RegExp(`(\\bText\\s*\\(\\s*)"${esc}"`, "g"),
      new RegExp(`(\\bButton\\s*\\(\\s*)"${esc}"`, "g"),
      new RegExp(`(\\bnavigationTitle\\s*\\(\\s*)"${esc}"`, "g"),
      new RegExp(`(\\bTextField\\s*\\(\\s*)"${esc}"`, "g"),
      new RegExp(`(\\bSecureField\\s*\\(\\s*)"${esc}"`, "g"),
      new RegExp(`(\\bLabel\\s*\\(\\s*)"${esc}"`, "g"),
      new RegExp(`(\\bSection\\s*\\(\\s*(?:header:\\s*)?)"${esc}"`, "g"),
      new RegExp(`(\\btoggleStyle[^\\n]*label:\\s*Text\\s*\\(\\s*)"${esc}"`, "g"),
      // String(localized: "English") already key-like — also plain assigned titles
      new RegExp(`(\\bString\\s*\\(\\s*localized:\\s*)"${esc}"`, "g"),
    ];

    for (const re of patterns) {
      const before = source;
      source = source.replace(re, `$1"${key}"`);
      if (source !== before) replacements += 1;
    }
  }

  if (source !== orig) {
    fs.writeFileSync(abs, source);
    filesTouched += 1;
  }
}

console.log(
  `iOS wire: ${filesTouched} files, ~${replacements} replacement groups.`,
);
