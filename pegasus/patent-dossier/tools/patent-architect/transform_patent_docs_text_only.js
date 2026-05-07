const fs = require('fs');
const path = require('path');

const dossierRoot = path.resolve(__dirname, '../..');
const outputRoot = path.join(dossierRoot, 'text-architecture');
const includeI18n = !process.argv.includes('--exclude-i18n');

function toPosix(p) {
  return p.split(path.sep).join('/');
}

function collectMarkdownFiles(dir, acc = []) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    const rel = toPosix(path.relative(dossierRoot, fullPath));

    if (rel.startsWith('text-architecture/')) {
      continue;
    }

    if (!includeI18n && rel.startsWith('i18n/')) {
      continue;
    }

    if (entry.isDirectory()) {
      collectMarkdownFiles(fullPath, acc);
      continue;
    }

    if (entry.isFile() && entry.name.endsWith('.md')) {
      acc.push(fullPath);
    }
  }
  return acc;
}

function uniq(items) {
  return Array.from(new Set(items.map((x) => x.trim()).filter(Boolean)));
}

function parseSections(content) {
  const lines = content.split(/\r?\n/);
  const sections = [];
  let current = { title: 'Preamble', level: 0, lines: [] };

  for (const line of lines) {
    const m = line.match(/^(#{1,6})\s+(.+)$/);
    if (m) {
      sections.push(current);
      current = { title: m[2].trim(), level: m[1].length, lines: [] };
      continue;
    }
    current.lines.push(line);
  }

  sections.push(current);

  return sections.filter((s) => {
    const hasBody = s.lines.some((line) => line.trim().length > 0);
    return hasBody || s.title !== 'Preamble';
  });
}

function parseMetadata(content) {
  const metadata = {};
  const lines = content.split(/\r?\n/);

  for (const line of lines) {
    let m = line.match(/^\*\*([^*]+)\*\*:\s*(.+)$/);
    if (m) {
      metadata[m[1].trim()] = m[2].trim();
      continue;
    }

    m = line.match(/^([A-Za-z][A-Za-z0-9 _\-/]+):\s*(.+)$/);
    if (m && !line.startsWith('http')) {
      metadata[m[1].trim()] = m[2].trim();
    }
  }

  return metadata;
}

function extractCodeBlocks(content) {
  const blocks = [];
  const re = /```([a-zA-Z0-9_-]*)\n([\s\S]*?)```/g;
  let m = re.exec(content);
  while (m) {
    blocks.push({ lang: (m[1] || '').trim().toLowerCase(), body: m[2] });
    m = re.exec(content);
  }
  return blocks;
}

function cleanNode(raw) {
  const s = raw.trim();
  const square = s.match(/\[([^\]]+)\]/);
  if (square) return square[1].trim();
  const curly = s.match(/\{([^}]+)\}/);
  if (curly) return curly[1].trim();
  const round = s.match(/\(([^)]+)\)/);
  if (round) return round[1].trim();
  return s.replace(/^[A-Za-z0-9_]+\s*/, '').trim() || s;
}

function extractMermaidSteps(content) {
  const blocks = extractCodeBlocks(content).filter((b) => b.lang === 'mermaid');
  const steps = [];

  for (const block of blocks) {
    const lines = block.body.split(/\r?\n/);
    for (const line of lines) {
      if (!line.includes('-->')) continue;

      const parts = line.split('-->');
      if (parts.length < 2) continue;

      const left = cleanNode(parts[0]);
      const rightRaw = parts.slice(1).join('-->');
      const condMatch = rightRaw.match(/\|([^|]+)\|/);
      const condition = condMatch ? condMatch[1].trim() : '';
      const right = cleanNode(rightRaw.replace(/\|[^|]+\|/g, ''));

      if (!left || !right) continue;

      if (condition) {
        steps.push(`${left} -> (${condition}) -> ${right}`);
      } else {
        steps.push(`${left} -> ${right}`);
      }
    }
  }

  return uniq(steps);
}

function extractTableFeatures(content) {
  const lines = content.split(/\r?\n/).map((line) => line.trim());
  const rows = [];
  let headers = null;

  for (const line of lines) {
    if (!line.startsWith('|')) {
      headers = null;
      continue;
    }

    if (line.includes('---')) {
      continue;
    }

    const cells = line
      .split('|')
      .map((c) => c.trim())
      .filter(Boolean);

    if (cells.length === 0) continue;

    if (!headers) {
      headers = cells;
      continue;
    }

    const row = {};
    for (let i = 0; i < headers.length; i += 1) {
      row[headers[i]] = cells[i] || '';
    }
    rows.push(row);
  }

  const featureKeys = [
    'Feature Name',
    'Feature Family',
    'Feature',
    'Capability',
    'Feature Expansion Area',
    'Title'
  ];

  const features = [];
  for (const row of rows) {
    for (const key of featureKeys) {
      if (row[key]) {
        features.push(row[key]);
        break;
      }
    }
  }

  return uniq(features);
}

function extractEndpoints(content) {
  const endpoints = [];
  const re = /(\/v1\/[A-Za-z0-9_{}\-/]+|\/ws\/[A-Za-z0-9_{}\-/]+)/g;
  let m = re.exec(content);
  while (m) {
    endpoints.push(m[1]);
    m = re.exec(content);
  }
  return uniq(endpoints).sort();
}

function extractPaths(content) {
  const paths = [];
  const re = /(apps\/[A-Za-z0-9_.\-/]+\.(?:ts|tsx|js|jsx|go|kt|swift|md))/g;
  let m = re.exec(content);
  while (m) {
    paths.push(m[1]);
    m = re.exec(content);
  }
  return uniq(paths);
}

function extractFormulas(content) {
  const formulaLines = [];
  const lines = content.split(/\r?\n/);
  let inCode = false;

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (line.startsWith('```')) {
      inCode = !inCode;
      continue;
    }
    if (inCode) continue;
    if (line.length < 6) continue;
    if (line.startsWith('#')) continue;
    if (line.startsWith('-')) continue;
    if (line.startsWith('|')) continue;
    if (/^\*\*generated/i.test(line)) continue;
    if (/^generated\s*:/i.test(line)) continue;

    const isFormulaLike =
      line.includes('=') ||
      /\b(arg|max|min|var|sum|sigma|delta|covariance|likelihood|TOA|TOT|Tprop|f\(|EKF|Kalman)\b/i.test(line) ||
      /[xX]\s*,\s*[yY]\s*,\s*[zZ]/.test(line) ||
      /\b\d+\s+[+\-*/]\s+\d+\b/.test(line);

    if (isFormulaLike && line.length <= 220) {
      formulaLines.push(line);
    }
  }

  return uniq(formulaLines).slice(0, 20);
}

function extractAbstract(content, metadata) {
  const hints = [];

  const preferredKeys = [
    'Purpose',
    'Invention Summary',
    'Summary',
    'Abstract',
    'Title',
    'Invention Name'
  ];

  for (const key of preferredKeys) {
    if (metadata[key]) {
      hints.push(metadata[key]);
    }
  }

  const text = content
    .replace(/```[\s\S]*?```/g, ' ')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('#') && !line.startsWith('|') && !line.startsWith('**'));

  for (const line of text) {
    if (line.length > 60) {
      hints.push(line);
    }
    if (hints.length >= 3) break;
  }

  return uniq(hints).slice(0, 3);
}

function extractSectionLines(sections, patterns, limit = 24) {
  const result = [];
  const regexes = patterns.map((p) => new RegExp(p, 'i'));

  for (const section of sections) {
    if (!regexes.some((re) => re.test(section.title))) continue;

    let inCode = false;

    for (const line of section.lines) {
      const trimmed = line.trim();
      if (trimmed.startsWith('```')) {
        inCode = !inCode;
        continue;
      }
      if (inCode) continue;
      if (!trimmed) continue;
      if (trimmed.startsWith('#')) continue;
      if (trimmed.startsWith('|')) continue;
      result.push(trimmed.replace(/^[-*]\s+/, ''));
      if (result.length >= limit) return uniq(result);
    }
  }

  return uniq(result);
}

function buildClaimsElements({ features, algorithms, endpoints, formulas, constraints }) {
  const items = [];

  if (features.length > 0) {
    items.push(`Feature family coverage includes ${features.slice(0, 6).join('; ')}.`);
  }

  if (algorithms.length > 0) {
    items.push(`Algorithmic sequence includes ${algorithms.slice(0, 3).join(' | ')}.`);
  }

  if (endpoints.length > 0) {
    items.push(`Contract surface is exposed through ${endpoints.slice(0, 6).join(', ')}.`);
  }

  if (formulas.length > 0) {
    items.push('Mathematical or scoring expressions are explicitly used for optimization or estimation.');
  }

  if (constraints.length > 0) {
    items.push(`Integrity constraints include ${constraints.slice(0, 4).join('; ')}.`);
  }

  if (items.length === 0) {
    items.push('The document defines a deterministic input-processing-output method with explicit state controls and bounded side effects.');
  }

  return items;
}

function normalizeDocument(sourcePath) {
  const rel = toPosix(path.relative(dossierRoot, sourcePath));
  const content = fs.readFileSync(sourcePath, 'utf8');

  const sections = parseSections(content);
  const metadata = parseMetadata(content);
  const featureRows = extractTableFeatures(content);
  const endpoints = extractEndpoints(content);
  const implementationAnchors = extractPaths(content);
  const mermaidSteps = extractMermaidSteps(content);
  const formulas = extractFormulas(content);
  const abstractLines = extractAbstract(content, metadata);

  const logicLines = extractSectionLines(sections, [
    'interactiveflows?',
    'algorithm',
    'workflow',
    'state',
    'protocol',
    'sequence',
    'logic'
  ]);

  const architectureLines = extractSectionLines(sections, [
    'layoutzones?',
    'system',
    'components?',
    'sourcefiles?',
    'shell',
    'platform',
    'role'
  ]);

  const dataContractLines = extractSectionLines(sections, [
    'datadependencies',
    'readendpoints?',
    'writeendpoints?',
    'contracts?',
    'api',
    'event'
  ]);

  const constraintLines = uniq([
    ...extractSectionLines(sections, ['statevariants?', 'edge', 'constraint', 'visibilityrule', 'guard', 'lock', 'idempotency']),
    ...logicLines.filter((line) => /lock|guard|idempotency|conflict|reject|retry|ttl|scope|authorization/i.test(line))
  ]).slice(0, 24);

  const algorithms = uniq([...mermaidSteps, ...logicLines]).slice(0, 28);
  const features = uniq([
    ...featureRows,
    ...sections
      .filter((s) => s.level >= 2)
      .map((s) => s.title)
      .filter((title) => !/generatedat|pageid|route|platform|role|status|purpose|layoutzones|buttonplacements|iconplacements|figureblueprints/i.test(title))
  ]).slice(0, 40);

  const claimElements = buildClaimsElements({
    features,
    algorithms,
    endpoints,
    formulas,
    constraints: constraintLines
  });

  const metadataTitle =
    metadata.Pageid ||
    metadata.PageId ||
    metadata.pageid ||
    metadata['Invention Name'] ||
    metadata.Title ||
    metadata.title;

  let title = metadataTitle || sections.find((s) => s.level === 1)?.title || path.basename(sourcePath, '.md');
  if (/^sourcefiles$/i.test(title)) {
    title = path.basename(sourcePath, '.md');
  }

  const generatedAt = new Date().toISOString();

  const out = [];
  out.push(`# Technical Patent Architecture: ${title}`);
  out.push('');
  out.push(`Source Document: ${rel}`);
  out.push(`Generated At: ${generatedAt}`);
  out.push('Mode: Text-only architecture extraction (no visual blueprint blocks)');
  out.push('');

  out.push('## Technical Abstract');
  if (abstractLines.length === 0) {
    out.push('- No explicit abstract paragraph detected; refer to system and algorithm sections below.');
  } else {
    for (const line of abstractLines) {
      out.push(`- ${line}`);
    }
  }
  out.push('');

  out.push('## System Architecture');
  const architectureItems = uniq([
    ...Object.entries(metadata)
      .filter(([k]) => /platform|role|route|navroute|sourcefile|sourcefiles|shell|status|purpose|pageid/i.test(k))
      .map(([k, v]) => `${k}: ${v}`),
    ...implementationAnchors.map((p) => `Implementation Anchor: ${p}`),
    ...architectureLines
  ]);

  if (architectureItems.length === 0) {
    out.push('- Architecture signals were not explicitly tagged in metadata.');
  } else {
    for (const item of architectureItems.slice(0, 30)) {
      out.push(`- ${item}`);
    }
  }
  out.push('');

  out.push('## Feature Set');
  if (features.length === 0) {
    out.push('- No explicit feature table detected.');
  } else {
    features.slice(0, 40).forEach((item, idx) => {
      out.push(`${idx + 1}. ${item}`);
    });
  }
  out.push('');

  out.push('## Algorithmic and Logical Flow');
  if (algorithms.length === 0) {
    out.push('- No algorithm or workflow section detected.');
  } else {
    algorithms.slice(0, 28).forEach((step, idx) => {
      out.push(`${idx + 1}. ${step}`);
    });
  }
  out.push('');

  out.push('## Mathematical Formulations');
  if (formulas.length === 0) {
    out.push('- No explicit closed-form equations were detected in this source file.');
  } else {
    formulas.forEach((f) => out.push(`- ${f}`));
  }
  out.push('');

  out.push('## Interfaces and Data Contracts');
  const contracts = uniq([
    ...endpoints.map((ep) => `Endpoint: ${ep}`),
    ...dataContractLines
  ]);
  if (contracts.length === 0) {
    out.push('- No explicit API contract lines detected.');
  } else {
    contracts.slice(0, 36).forEach((line) => out.push(`- ${line}`));
  }
  out.push('');

  out.push('## Operational Constraints and State Rules');
  if (constraintLines.length === 0) {
    out.push('- No explicit constraint block detected.');
  } else {
    constraintLines.forEach((line) => out.push(`- ${line}`));
  }
  out.push('');

  out.push('## Claims-Oriented Technical Elements');
  claimElements.forEach((line, idx) => {
    out.push(`${idx + 1}. ${line}`);
  });
  out.push('');

  return out.join('\n');
}

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function writeIndex(records) {
  const lines = [];
  lines.push('# Text-Only Patent Architecture Corpus');
  lines.push('');
  lines.push('Generated by tools/patent-architect/transform_patent_docs_text_only.js');
  lines.push('');
  lines.push(`Total documents transformed: ${records.length}`);
  lines.push('');
  lines.push('| Source | Output |');
  lines.push('|---|---|');

  for (const rec of records) {
    lines.push(`| ${rec.source} | ${rec.output} |`);
  }

  lines.push('');
  lines.push('## Notes');
  lines.push('');
  lines.push('- Original files are preserved.');
  lines.push('- Output focuses on architecture, system, features, algorithms, logic, formulas, contracts, and constraints.');
  lines.push('- Visual blueprint sections are transformed into text workflow and component statements.');
  lines.push('');

  fs.writeFileSync(path.join(outputRoot, 'INDEX.md'), lines.join('\n'));
}

function main() {
  ensureDir(outputRoot);
  const files = collectMarkdownFiles(dossierRoot);
  const records = [];

  for (const filePath of files) {
    const rel = toPosix(path.relative(dossierRoot, filePath));
    const outPath = path.join(outputRoot, rel);
    ensureDir(path.dirname(outPath));

    const normalized = normalizeDocument(filePath);
    fs.writeFileSync(outPath, normalized);

    records.push({
      source: rel,
      output: toPosix(path.relative(dossierRoot, outPath))
    });
  }

  writeIndex(records);

  console.log(`Transformed ${records.length} markdown documents.`);
  console.log(`Output root: ${toPosix(path.relative(process.cwd(), outputRoot))}`);
  if (!includeI18n) {
    console.log('i18n documents were excluded by flag.');
  }
}

main();
