#!/usr/bin/env node

import fs from "node:fs/promises";
import path from "node:path";
import process from "node:process";

const DEFAULT_INDEX_PATH = path.join(
  process.cwd(),
  ".agents",
  "extensions",
  "ast-engine",
  ".cache",
  "symbol-index.json",
);

const SOURCE_EXTENSIONS = new Set([".ts", ".tsx", ".js", ".jsx", ".go"]);
const SKIP_DIRS = new Set([
  ".git",
  ".next",
  ".turbo",
  ".cache",
  "node_modules",
  "dist",
  "build",
  "coverage",
  "vendor",
  "tmp",
  "test-results",
  "playwright-report",
]);

function usageAndExit(code = 0) {
  const message = [
    "Usage:",
    "  node engine.mjs index --root <dir> [--index <file>]",
    "  node engine.mjs definition --symbol <name> [--index <file>] [--root <dir>]",
    "  node engine.mjs usages --symbol <name> [--index <file>] [--root <dir>] [--limit <n>]",
    "  node engine.mjs graph --symbol <name> [--index <file>] [--root <dir>] [--limit <n>]",
    "",
    "Examples:",
    "  node engine.mjs index --root pegasus",
    "  node engine.mjs definition --symbol RegisterRoutes",
    "  node engine.mjs usages --symbol RegisterRoutes --limit 20",
  ].join("\n");
  process.stdout.write(`${message}\n`);
  process.exit(code);
}

function parseArgs(argv) {
  const positional = [];
  const named = {};

  for (let i = 0; i < argv.length; i += 1) {
    const token = argv[i];
    if (!token.startsWith("--")) {
      positional.push(token);
      continue;
    }

    const key = token.slice(2);
    const next = argv[i + 1];
    if (!next || next.startsWith("--")) {
      named[key] = true;
      continue;
    }

    named[key] = next;
    i += 1;
  }

  return { command: positional[0], named };
}

function toWorkspaceRelative(filePath, rootDir) {
  return path.relative(rootDir, filePath).split(path.sep).join("/");
}

function normalizeRoot(rootInput) {
  return path.resolve(process.cwd(), rootInput || ".");
}

async function pathExists(filePath) {
  try {
    await fs.access(filePath);
    return true;
  } catch {
    return false;
  }
}

async function collectSourceFiles(rootDir) {
  const out = [];

  async function walk(currentDir) {
    const entries = await fs.readdir(currentDir, { withFileTypes: true });

    for (const entry of entries) {
      const full = path.join(currentDir, entry.name);

      if (entry.isDirectory()) {
        if (SKIP_DIRS.has(entry.name)) {
          continue;
        }
        await walk(full);
        continue;
      }

      const ext = path.extname(entry.name);
      if (SOURCE_EXTENSIONS.has(ext)) {
        out.push(full);
      }
    }
  }

  await walk(rootDir);
  return out;
}

function languageFromExt(filePath) {
  const ext = path.extname(filePath);
  if (ext === ".go") {
    return "go";
  }
  return "ts";
}

function parseTSDefinitions(lines) {
  const defs = [];

  const patterns = [
    { kind: "class", regex: /^\s*export\s+default\s+class\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "class", regex: /^\s*export\s+class\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "class", regex: /^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "interface", regex: /^\s*export\s+interface\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "interface", regex: /^\s*interface\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "type", regex: /^\s*export\s+type\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "type", regex: /^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "function", regex: /^\s*export\s+async\s+function\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "function", regex: /^\s*export\s+function\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "function", regex: /^\s*async\s+function\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "function", regex: /^\s*function\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "variable", regex: /^\s*export\s+const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s*)?\([^)]*\)\s*=>/ },
    { kind: "variable", regex: /^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s*)?\([^)]*\)\s*=>/ },
  ];

  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i];
    for (const pattern of patterns) {
      const match = line.match(pattern.regex);
      if (!match) {
        continue;
      }

      defs.push({ name: match[1], kind: pattern.kind, line: i + 1 });
      break;
    }
  }

  return defs;
}

function parseGoDefinitions(lines) {
  const defs = [];

  const patterns = [
    { kind: "type", regex: /^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+(?:struct|interface|map|\[|chan|func|\*)/ },
    { kind: "function", regex: /^\s*func\s+\([^)]+\)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(/ },
    { kind: "function", regex: /^\s*func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(/ },
    { kind: "const", regex: /^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
    { kind: "var", regex: /^\s*var\s+([A-Za-z_][A-Za-z0-9_]*)\b/ },
  ];

  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i];

    for (const pattern of patterns) {
      const match = line.match(pattern.regex);
      if (!match) {
        continue;
      }

      defs.push({ name: match[1], kind: pattern.kind, line: i + 1 });
      break;
    }
  }

  return defs;
}

function parseImports(lines, language) {
  const imports = [];

  if (language === "go") {
    let inImportBlock = false;

    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed.startsWith("import (")) {
        inImportBlock = true;
        continue;
      }

      if (inImportBlock) {
        if (trimmed === ")") {
          inImportBlock = false;
          continue;
        }

        const blockMatch = trimmed.match(/^(?:[A-Za-z_][A-Za-z0-9_]*\s+)?"([^"]+)"/);
        if (blockMatch) {
          imports.push(blockMatch[1]);
        }
        continue;
      }

      const singleMatch = trimmed.match(/^import\s+(?:[A-Za-z_][A-Za-z0-9_]*\s+)?"([^"]+)"/);
      if (singleMatch) {
        imports.push(singleMatch[1]);
      }
    }

    return imports;
  }

  for (const line of lines) {
    const trimmed = line.trim();
    const importMatch = trimmed.match(/^import\s+.*?from\s+["']([^"']+)["']/);
    if (importMatch) {
      imports.push(importMatch[1]);
      continue;
    }

    const requireMatch = trimmed.match(/^const\s+.*?=\s*require\(["']([^"']+)["']\)/);
    if (requireMatch) {
      imports.push(requireMatch[1]);
    }
  }

  return imports;
}

async function buildIndex(rootDir) {
  const sourceFiles = await collectSourceFiles(rootDir);
  const files = [];
  const symbols = {};

  for (const filePath of sourceFiles) {
    const language = languageFromExt(filePath);
    const raw = await fs.readFile(filePath, "utf8");
    const lines = raw.split(/\r?\n/);

    const definitions =
      language === "go" ? parseGoDefinitions(lines) : parseTSDefinitions(lines);
    const imports = parseImports(lines, language);
    const workspaceRelativePath = toWorkspaceRelative(filePath, rootDir);

    files.push({
      path: workspaceRelativePath,
      language,
      lineCount: lines.length,
      imports,
      definitions,
    });

    for (const def of definitions) {
      if (!symbols[def.name]) {
        symbols[def.name] = [];
      }

      symbols[def.name].push({
        path: workspaceRelativePath,
        line: def.line,
        kind: def.kind,
        language,
      });
    }
  }

  return {
    generatedAt: new Date().toISOString(),
    rootDir,
    fileCount: files.length,
    files,
    symbols,
  };
}

async function writeIndex(indexPath, payload) {
  await fs.mkdir(path.dirname(indexPath), { recursive: true });
  await fs.writeFile(indexPath, `${JSON.stringify(payload, null, 2)}\n`, "utf8");
}

async function readIndex(indexPath) {
  const exists = await pathExists(indexPath);
  if (!exists) {
    return null;
  }

  const raw = await fs.readFile(indexPath, "utf8");
  return JSON.parse(raw);
}

function compileWordBoundaryRegex(symbol) {
  return new RegExp(`\\b${symbol.replace(/[.*+?^${}()|[\\]\\]/g, "\\$&")}\\b`);
}

async function findUsagesFromFiles(rootDir, symbol, limit) {
  const matcher = compileWordBoundaryRegex(symbol);
  const sourceFiles = await collectSourceFiles(rootDir);
  const out = [];

  for (const absolutePath of sourceFiles) {
    const rel = toWorkspaceRelative(absolutePath, rootDir);
    const raw = await fs.readFile(absolutePath, "utf8");
    const lines = raw.split(/\r?\n/);

    for (let i = 0; i < lines.length; i += 1) {
      const lineText = lines[i];
      if (!matcher.test(lineText)) {
        continue;
      }

      out.push({ path: rel, line: i + 1, text: lineText.trim() });
      if (out.length >= limit) {
        return out;
      }
    }
  }

  return out;
}

async function getOrBuildIndex(rootDir, indexPath) {
  const existing = await readIndex(indexPath);
  if (existing) {
    return existing;
  }

  const built = await buildIndex(rootDir);
  await writeIndex(indexPath, built);
  return built;
}

async function main() {
  const { command, named } = parseArgs(process.argv.slice(2));

  if (!command || command === "help" || command === "--help") {
    usageAndExit(0);
  }

  const rootDir = normalizeRoot(named.root);
  const indexPath = path.resolve(process.cwd(), named.index || DEFAULT_INDEX_PATH);
  const symbol = named.symbol;
  const limit = Number.parseInt(String(named.limit || "50"), 10);

  if (command === "index") {
    const payload = await buildIndex(rootDir);
    await writeIndex(indexPath, payload);

    process.stdout.write(
      `${JSON.stringify({ ok: true, indexPath, fileCount: payload.fileCount, generatedAt: payload.generatedAt }, null, 2)}\n`,
    );
    return;
  }

  if (!symbol || typeof symbol !== "string") {
    process.stderr.write("Missing required --symbol argument.\n");
    usageAndExit(1);
  }

  if (command === "definition") {
    const index = await getOrBuildIndex(rootDir, indexPath);
    const definitions = index.symbols[symbol] || [];

    process.stdout.write(
      `${JSON.stringify({ ok: true, command: "definition", symbol, count: definitions.length, definitions }, null, 2)}\n`,
    );
    return;
  }

  if (command === "usages") {
    const usages = await findUsagesFromFiles(rootDir, symbol, Number.isNaN(limit) ? 50 : limit);

    process.stdout.write(
      `${JSON.stringify({ ok: true, command: "usages", symbol, count: usages.length, usages }, null, 2)}\n`,
    );
    return;
  }

  if (command === "graph") {
    const index = await getOrBuildIndex(rootDir, indexPath);
    const definitions = index.symbols[symbol] || [];
    const usages = await findUsagesFromFiles(rootDir, symbol, Number.isNaN(limit) ? 50 : limit);

    const impactedFiles = Array.from(
      new Set([
        ...definitions.map((item) => item.path),
        ...usages.map((item) => item.path),
      ]),
    );

    const importNeighbors = [];
    for (const file of index.files) {
      if (!impactedFiles.includes(file.path)) {
        continue;
      }

      for (const source of file.imports) {
        importNeighbors.push({ from: file.path, import: source });
      }
    }

    process.stdout.write(
      `${JSON.stringify(
        {
          ok: true,
          command: "graph",
          symbol,
          definitions,
          usages,
          impactedFiles,
          importNeighbors,
        },
        null,
        2,
      )}\n`,
    );
    return;
  }

  process.stderr.write(`Unknown command: ${command}\n`);
  usageAndExit(1);
}

main().catch((error) => {
  process.stderr.write(
    `${JSON.stringify(
      {
        ok: false,
        error: error?.message || String(error),
      },
      null,
      2,
    )}\n`,
  );
  process.exit(1);
});
