import fs from "node:fs";
import path from "node:path";

import { repoRoot, writeFile } from "./shared.mjs";

const inventoryRoots = [
  path.resolve(repoRoot, "apps"),
  path.resolve(repoRoot, "packages"),
];

const skipDirectories = new Set([
  ".git",
  ".next",
  ".turbo",
  "build",
  "dist",
  "dist-build-check",
  "generated",
  "node_modules",
  "Pods",
]);

const analyzers = [
  {
    kind: "web-jsx-text",
    matches(relativePath) {
      return /\.(ts|tsx|js|jsx)$/.test(relativePath);
    },
    patterns: [
      />([A-Za-z][^<{]{2,})</g,
      /\bplaceholder\s*=\s*"([^"]+)"/g,
      /\baria-label\s*=\s*"([^"]+)"/g,
      /\btitle\s*=\s*"([^"]+)"/g,
    ],
  },
  {
    kind: "swiftui-inline-text",
    matches(relativePath) {
      return relativePath.endsWith(".swift");
    },
    patterns: [
      /\bText\("([^"]+)"/g,
      /\bButton\("([^"]+)"/g,
      /\bnavigationTitle\("([^"]+)"/g,
      /\bTextField\("([^"]+)"/g,
      /\bLabel\("([^"]+)"/g,
    ],
  },
  {
    kind: "compose-inline-text",
    matches(relativePath) {
      return relativePath.endsWith(".kt");
    },
    patterns: [
      /\btext\s*=\s*"([^"]+)"/g,
      /\blabel\s*=\s*"([^"]+)"/g,
      /\bcontentDescription\s*=\s*"([^"]+)"/g,
      /\btitle\s*=\s*"([^"]+)"/g,
    ],
  },
  {
    kind: "xml-inline-text",
    matches(relativePath) {
      return relativePath.endsWith(".xml");
    },
    patterns: [
      /\bandroid:text\s*=\s*"([^"]+)"/g,
      /\bandroid:contentDescription\s*=\s*"([^"]+)"/g,
      />\s*([A-Za-z][^<]{2,})\s*</g,
    ],
  },
  {
    kind: "backend-inline-problem-text",
    matches(relativePath) {
      return relativePath.endsWith(".go");
    },
    patterns: [
      /WriteProblem\([^)]*"([^"]+)"[^)]*"([^"]+)"/g,
      /Title:\s*"([^"]+)"/g,
      /Detail:\s*"([^"]+)"/g,
    ],
  },
];

function walkDirectory(directoryPath, files = []) {
  for (const entry of fs.readdirSync(directoryPath, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      if (skipDirectories.has(entry.name)) {
        continue;
      }

      walkDirectory(path.join(directoryPath, entry.name), files);
      continue;
    }

    files.push(path.join(directoryPath, entry.name));
  }

  return files;
}

function findLineNumber(content, index) {
  return content.slice(0, index).split("\n").length;
}

function collectEntries(filePath) {
  const relativePath = path.relative(repoRoot, filePath);
  const content = fs.readFileSync(filePath, "utf8");
  const entries = [];

  for (const analyzer of analyzers) {
    if (!analyzer.matches(relativePath)) {
      continue;
    }

    for (const pattern of analyzer.patterns) {
      for (const match of content.matchAll(pattern)) {
        const capture = match
          .slice(1)
          .filter(Boolean)
          .join(" | ")
          .trim();

        if (!capture) {
          continue;
        }

        entries.push({
          kind: analyzer.kind,
          file: relativePath,
          line: findLineNumber(content, match.index ?? 0),
          text: capture,
        });
      }
    }
  }

  return entries;
}

const files = inventoryRoots.flatMap((rootPath) => walkDirectory(rootPath));
const entries = files.flatMap((filePath) => collectEntries(filePath));
const summary = entries.reduce((accumulator, entry) => {
  accumulator[entry.kind] = (accumulator[entry.kind] ?? 0) + 1;
  return accumulator;
}, {});

const output = {
  generated_at: new Date().toISOString(),
  total_files_scanned: files.length,
  total_matches: entries.length,
  summary,
  entries,
};

const outputPath = path.resolve(repoRoot, "packages/i18n/generated/inventory.json");
writeFile(outputPath, `${JSON.stringify(output, null, 2)}\n`);

console.log(
  `Localization inventory captured ${entries.length} potential hardcoded strings across ${files.length} files.`,
);
console.log(`Inventory written to ${outputPath}.`);
