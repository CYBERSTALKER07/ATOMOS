import { readAllFlatCatalogs, validateCatalogs } from "./shared.mjs";

const flatCatalogs = readAllFlatCatalogs();
const result = validateCatalogs(flatCatalogs);

console.log(
  `Localization catalogs are valid for ${Object.keys(flatCatalogs).length} locales and ${result.keyCount} keys.`,
);
