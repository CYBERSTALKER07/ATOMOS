/**
 * Wire mobile interpolations to localized format calls.
 *
 * iOS:  Text("Hi \(name)") → Text(L10n.format("key", "\(name)"))
 * Android: text = "Hi $name" → text = stringResource(R.string.key, name)
 *
 * Also installs packages/i18n/scripts/templates/L10n.swift into each iOS app.
 *
 * Usage: node packages/i18n/scripts/wire-mobile-interpolations.mjs
 */
import fs from "node:fs";
import path from "node:path";

import {
  generatedDir,
  repoRoot,
  toAndroidName,
  writeFile,
} from "./shared.mjs";

const extract = JSON.parse(
  fs.readFileSync(
    path.resolve(generatedDir, "mobile-interpolation-extract.json"),
    "utf8",
  ),
);

const L10N_TEMPLATE = fs.readFileSync(
  path.resolve(repoRoot, "packages/i18n/scripts/templates/L10n.swift"),
  "utf8",
);

const IOS_L10N_DEST = [
  "apps/supplier-app-ios/SupplierApp/L10n.swift",
  "apps/factory-app-ios/FactoryApp/L10n.swift",
  "apps/warehouse-app-ios/WarehouseApp/L10n.swift",
  "apps/payload-app-ios/payload-app-ios/L10n.swift",
  "apps/retailer-app-ios/retailerapp/reatilerapp/L10n.swift",
  "apps/driver-app-ios/driverappios/driverappios/L10n.swift",
];

function ensureStringResourceImport(source) {
  if (/import\s+androidx\.compose\.ui\.res\.stringResource/.test(source)) {
    return source;
  }
  const pkg = source.match(/^package\s+[^\n]+\n/);
  if (!pkg) return source;
  return (
    source.slice(0, pkg[0].length) +
    "\nimport androidx.compose.ui.res.stringResource\n" +
    source.slice(pkg[0].length)
  );
}

function iosCall(rep) {
  const args = rep.args.map((a) => `"\\(${a})"`).join(", ");
  return args
    ? `L10n.format("${rep.key}", ${args})`
    : `L10n.format("${rep.key}")`;
}

function androidCall(rep) {
  const name = toAndroidName(rep.key);
  const args = rep.args.join(", ");
  return args
    ? `stringResource(R.string.${name}, ${args})`
    : `stringResource(R.string.${name})`;
}

/** Replace first occurrence of needle with value. */
function replaceFirst(source, needle, value) {
  const idx = source.indexOf(needle);
  if (idx < 0) return { source, hit: false };
  return {
    source: source.slice(0, idx) + value + source.slice(idx + needle.length),
    hit: true,
  };
}

let filesTouched = 0;
let replacements = 0;
let failed = 0;

for (const dest of IOS_L10N_DEST) {
  writeFile(path.resolve(repoRoot, dest), L10N_TEMPLATE);
}

for (const wireFile of extract.wire_files) {
  const abs = path.resolve(repoRoot, wireFile.file);
  if (!fs.existsSync(abs)) continue;

  let source = fs.readFileSync(abs, "utf8");
  const orig = source;
  const isIos = wireFile.file.includes("-ios/");
  const isAndroid = wireFile.file.includes("-android/");

  const sorted = [...wireFile.replacements].sort(
    (a, b) => (b.match?.length || b.original.length) - (a.match?.length || a.original.length),
  );

  for (const rep of sorted) {
    if (!rep.match) {
      failed += 1;
      continue;
    }

    if (isIos) {
      const next = `${rep.callee}(${iosCall(rep)}`;
      const result = replaceFirst(source, rep.match, next);
      source = result.source;
      if (result.hit) replacements += 1;
      else failed += 1;
    }

    if (isAndroid) {
      const call = androidCall(rep);
      let next;
      if (rep.kind === "text" || rep.callee === "Text") {
        next = `Text(${call}`;
      } else {
        next = `${rep.callee} = ${call}`;
      }
      // match begins with `Text("` or `text = "`
      const result = replaceFirst(source, rep.match, next);
      source = result.source;
      if (result.hit) replacements += 1;
      else failed += 1;
    }
  }

  if (source !== orig) {
    if (isAndroid) source = ensureStringResourceImport(source);
    fs.writeFileSync(abs, source);
    filesTouched += 1;
  }
}

console.log(
  `Interpolation wire: ${filesTouched} files, ~${replacements} replacements, ${failed} unmatched, L10n.swift → ${IOS_L10N_DEST.length} apps.`,
);
