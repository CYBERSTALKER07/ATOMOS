/**
 * Wire packages/i18n/generated/{android,ios} into all mobile apps.
 *
 * Android: add sourceSets.main.res.srcDir pointing at generated/android
 * XcodeGen iOS: add en/ru/uz.lproj resource paths
 * Xcodeproj iOS (retailer/driver): sync-copy lproj into app Resources and
 *   ensure folders exist (pbxproj membership via folder reference sync file)
 *
 * Usage: node packages/i18n/scripts/wire-mobile-resources.mjs
 */
import fs from "node:fs";
import path from "node:path";

import { generatedDir, repoRoot, writeFile } from "./shared.mjs";

const ANDROID_APPS = [
  "driver-app-android",
  "factory-app-android",
  "payload-app-android",
  "retailer-app-android",
  "supplier-app-android",
  "warehouse-app-android",
];

const XCODEGEN_IOS = [
  "factory-app-ios",
  "payload-app-ios",
  "supplier-app-ios",
  "warehouse-app-ios",
];

/** Raw .xcodeproj apps: sync lproj into these relative destinations. */
const XCODEPROJ_IOS = [
  {
    app: "retailer-app-ios",
    dest: "retailerapp/reatilerapp",
  },
  {
    app: "driver-app-ios",
    dest: "driverappios/driverappios",
  },
];

const MARKER = "pegasusx-i18n-generated";

const androidResSnippet = `
    // ${MARKER}: shared en/ru/uz string catalogs
    sourceSets {
        getByName("main") {
            res.srcDir(rootProject.file("../../packages/i18n/generated/android"))
        }
    }
`;

function patchAndroidGradle(app) {
  const filePath = path.resolve(repoRoot, "apps", app, "app", "build.gradle.kts");
  if (!fs.existsSync(filePath)) {
    console.warn(`skip missing ${filePath}`);
    return false;
  }
  let source = fs.readFileSync(filePath, "utf8");
  if (source.includes(MARKER)) {
    console.log(`android already wired: ${app}`);
    return false;
  }

  // Insert after `android {` block opening — after namespace / compileSdk area,
  // prefer right after `android {` line.
  if (!/android\s*\{/.test(source)) {
    console.warn(`no android {} in ${app}`);
    return false;
  }

  source = source.replace(
    /android\s*\{/,
    (match) => `${match}\n${androidResSnippet}`,
  );
  fs.writeFileSync(filePath, source);
  console.log(`android wired: ${app}`);
  return true;
}

const iosResourceLines = [
  `      # ${MARKER}`,
  `      - path: ../../packages/i18n/generated/ios/en.lproj`,
  `      - path: ../../packages/i18n/generated/ios/ru.lproj`,
  `      - path: ../../packages/i18n/generated/ios/uz.lproj`,
];

function patchXcodegenProject(app) {
  const filePath = path.resolve(repoRoot, "apps", app, "project.yml");
  if (!fs.existsSync(filePath)) {
    console.warn(`skip missing ${filePath}`);
    return false;
  }
  let source = fs.readFileSync(filePath, "utf8");
  if (source.includes(MARKER)) {
    console.log(`ios xcodegen already wired: ${app}`);
    return false;
  }

  if (!/resources:\s*\n/.test(source)) {
    console.warn(`no resources: in ${app}/project.yml`);
    return false;
  }

  source = source.replace(
    /resources:\s*\n/,
    (match) => `${match}${iosResourceLines.join("\n")}\n`,
  );
  fs.writeFileSync(filePath, source);
  console.log(`ios xcodegen wired: ${app}`);
  return true;
}

function syncLprojCopy(app, destRel) {
  const destRoot = path.resolve(repoRoot, "apps", app, destRel);
  if (!fs.existsSync(destRoot)) {
    console.warn(`skip missing dest ${destRoot}`);
    return false;
  }

  for (const locale of ["en", "ru", "uz"]) {
    const src = path.resolve(generatedDir, "ios", `${locale}.lproj`);
    const dest = path.resolve(destRoot, `${locale}.lproj`);
    fs.mkdirSync(dest, { recursive: true });
    const srcFile = path.join(src, "Localizable.strings");
    const destFile = path.join(dest, "Localizable.strings");
    fs.copyFileSync(srcFile, destFile);
  }

  // Marker readme so regenerate knows to re-sync
  writeFile(
    path.join(destRoot, "I18N_SYNC.md"),
    `# i18n sync\n\nCopied from packages/i18n/generated/ios via wire-mobile-resources.mjs / sync-mobile-i18n.mjs.\nDo not edit Localizable.strings here by hand — edit catalogs and regenerate.\n`,
  );
  console.log(`ios lproj synced: ${app} -> ${destRel}`);
  return true;
}

let n = 0;
for (const app of ANDROID_APPS) {
  if (patchAndroidGradle(app)) n += 1;
}
for (const app of XCODEGEN_IOS) {
  if (patchXcodegenProject(app)) n += 1;
}
for (const { app, dest } of XCODEPROJ_IOS) {
  if (syncLprojCopy(app, dest)) n += 1;
}

console.log(`Mobile resource wiring touched ${n} targets.`);
