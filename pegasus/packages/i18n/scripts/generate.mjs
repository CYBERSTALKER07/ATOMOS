import path from "node:path";

import {
  escapeIos,
  escapeXml,
  generatedDir,
  readAllFlatCatalogs,
  supportedLocales,
  toAndroidName,
  validateCatalogs,
  writeFile,
} from "./shared.mjs";

function buildAndroidStrings(catalog) {
  const lines = Object.entries(catalog)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(
      ([key, value]) =>
        `    <string name="${toAndroidName(key)}">${escapeXml(value)}</string>`,
    );

  return `<resources>\n${lines.join("\n")}\n</resources>\n`;
}

function buildIosStrings(catalog) {
  return Object.entries(catalog)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, value]) => `"${key}" = "${escapeIos(value)}";`)
    .join("\n")
    .concat("\n");
}

const flatCatalogs = readAllFlatCatalogs();
validateCatalogs(flatCatalogs);

for (const locale of supportedLocales) {
  const catalog = flatCatalogs[locale];

  writeFile(
    path.resolve(generatedDir, "web", `${locale}.json`),
    `${JSON.stringify(catalog, null, 2)}\n`,
  );

  const androidFolder = locale === "en" ? "values" : `values-${locale}`;
  writeFile(
    path.resolve(generatedDir, "android", androidFolder, "strings.xml"),
    buildAndroidStrings(catalog),
  );

  writeFile(
    path.resolve(generatedDir, "ios", `${locale}.lproj`, "Localizable.strings"),
    buildIosStrings(catalog),
  );
}

console.log(
  `Generated localization assets for ${supportedLocales.length} locales in ${generatedDir}.`,
);
