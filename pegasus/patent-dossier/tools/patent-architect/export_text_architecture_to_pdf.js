const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

const dossierRoot = path.resolve(__dirname, '../..');
const sourceRoot = path.join(dossierRoot, 'text-architecture');
const outputRoot = path.join(dossierRoot, 'text-architecture-pdf');

function toPosix(p) {
  return p.split(path.sep).join('/');
}

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function collectMarkdownFiles(dir, acc = []) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
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

function escapeHtml(text) {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function renderInline(raw) {
  let s = escapeHtml(raw);

  s = s.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_m, text, href) => {
    const safeHref = escapeHtml(href);
    return `<a href="${safeHref}">${text}</a>`;
  });

  s = s.replace(/`([^`]+)`/g, '<code>$1</code>');
  s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');

  return s;
}

function markdownToHtml(markdown) {
  const lines = markdown.split(/\r?\n/);
  const out = [];
  let inCode = false;
  let inUl = false;
  let inOl = false;
  let inParagraph = false;

  const closeParagraph = () => {
    if (inParagraph) {
      out.push('</p>');
      inParagraph = false;
    }
  };

  const closeLists = () => {
    if (inUl) {
      out.push('</ul>');
      inUl = false;
    }
    if (inOl) {
      out.push('</ol>');
      inOl = false;
    }
  };

  for (const line of lines) {
    if (line.startsWith('```')) {
      if (!inCode) {
        closeParagraph();
        closeLists();
        out.push('<pre><code>');
        inCode = true;
      } else {
        out.push('</code></pre>');
        inCode = false;
      }
      continue;
    }

    if (inCode) {
      out.push(`${escapeHtml(line)}\n`);
      continue;
    }

    const trimmed = line.trim();

    if (trimmed.length === 0) {
      closeParagraph();
      closeLists();
      continue;
    }

    const heading = trimmed.match(/^(#{1,6})\s+(.+)$/);
    if (heading) {
      closeParagraph();
      closeLists();
      const level = heading[1].length;
      out.push(`<h${level}>${renderInline(heading[2])}</h${level}>`);
      continue;
    }

    const ul = trimmed.match(/^[-*+]\s+(.+)$/);
    if (ul) {
      closeParagraph();
      if (inOl) {
        out.push('</ol>');
        inOl = false;
      }
      if (!inUl) {
        out.push('<ul>');
        inUl = true;
      }
      out.push(`<li>${renderInline(ul[1])}</li>`);
      continue;
    }

    const ol = trimmed.match(/^\d+\.\s+(.+)$/);
    if (ol) {
      closeParagraph();
      if (inUl) {
        out.push('</ul>');
        inUl = false;
      }
      if (!inOl) {
        out.push('<ol>');
        inOl = true;
      }
      out.push(`<li>${renderInline(ol[1])}</li>`);
      continue;
    }

    if (trimmed.startsWith('|')) {
      closeParagraph();
      closeLists();
      out.push(`<p><code>${escapeHtml(trimmed)}</code></p>`);
      continue;
    }

    if (!inParagraph) {
      closeLists();
      out.push('<p>');
      inParagraph = true;
      out.push(renderInline(trimmed));
    } else {
      out.push(`<br>${renderInline(trimmed)}`);
    }
  }

  closeParagraph();
  closeLists();

  return out.join('\n');
}

function buildHtmlDocument(title, relPath, markdown) {
  const body = markdownToHtml(markdown);
  const safeTitle = escapeHtml(title);
  const safeRelPath = escapeHtml(relPath);

  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>${safeTitle}</title>
  <style>
    @page {
      size: A4;
      margin: 14mm 12mm 14mm 12mm;
    }
    html, body {
      padding: 0;
      margin: 0;
      color: #111;
      background: #fff;
      font-family: -apple-system, BlinkMacSystemFont, Segoe UI, Helvetica, Arial, sans-serif;
      font-size: 11.5px;
      line-height: 1.45;
    }
    main {
      padding: 0;
      margin: 0;
    }
    header.meta {
      border-bottom: 1px solid #ddd;
      margin-bottom: 10px;
      padding-bottom: 6px;
    }
    header.meta h1 {
      font-size: 16px;
      margin: 0 0 4px 0;
      line-height: 1.25;
    }
    header.meta p {
      margin: 0;
      font-size: 10px;
      color: #444;
      word-break: break-all;
    }
    h1, h2, h3, h4, h5, h6 {
      line-height: 1.25;
      margin: 12px 0 6px;
      page-break-after: avoid;
    }
    h1 { font-size: 18px; }
    h2 { font-size: 15px; }
    h3 { font-size: 13px; }
    p {
      margin: 6px 0;
      orphans: 3;
      widows: 3;
    }
    ul, ol {
      margin: 4px 0 8px 20px;
      padding: 0;
    }
    li { margin: 2px 0; }
    pre {
      white-space: pre-wrap;
      word-break: break-word;
      background: #f6f6f6;
      border: 1px solid #e3e3e3;
      border-radius: 4px;
      padding: 8px;
      margin: 8px 0;
      font-size: 10.5px;
      line-height: 1.35;
      page-break-inside: avoid;
    }
    code {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, monospace;
      font-size: 0.95em;
    }
    a {
      color: #0b57d0;
      text-decoration: none;
    }
  </style>
</head>
<body>
  <main>
    <header class="meta">
      <h1>${safeTitle}</h1>
      <p>${safeRelPath}</p>
    </header>
    ${body}
  </main>
</body>
</html>`;
}

function writeIndex(records) {
  const lines = [];
  lines.push('# Text Architecture PDF Export');
  lines.push('');
  lines.push(`Generated: ${new Date().toISOString()}`);
  lines.push(`Total PDFs: ${records.length}`);
  lines.push('');
  lines.push('| Markdown Source | PDF Output |');
  lines.push('|---|---|');

  for (const rec of records) {
    lines.push(`| ${rec.source} | ${rec.output} |`);
  }

  lines.push('');
  fs.writeFileSync(path.join(outputRoot, 'INDEX.md'), lines.join('\n'));
}

async function main() {
  if (!fs.existsSync(sourceRoot)) {
    throw new Error(`Source folder not found: ${sourceRoot}`);
  }

  ensureDir(outputRoot);

  const files = collectMarkdownFiles(sourceRoot);
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  const records = [];

  try {
    for (const filePath of files) {
      const relFromSource = toPosix(path.relative(sourceRoot, filePath));
      const relPdf = relFromSource.replace(/\.md$/i, '.pdf');
      const outPath = path.join(outputRoot, relPdf);
      ensureDir(path.dirname(outPath));

      const markdown = fs.readFileSync(filePath, 'utf8');
      const title = path.basename(filePath, '.md');
      const html = buildHtmlDocument(title, relFromSource, markdown);

      await page.setContent(html, { waitUntil: 'load' });
      await page.emulateMedia({ media: 'print' });
      await page.pdf({
        path: outPath,
        format: 'A4',
        printBackground: true,
        margin: {
          top: '12mm',
          right: '10mm',
          bottom: '12mm',
          left: '10mm'
        }
      });

      records.push({
        source: toPosix(path.relative(dossierRoot, filePath)),
        output: toPosix(path.relative(dossierRoot, outPath))
      });
    }
  } finally {
    await page.close();
    await browser.close();
  }

  writeIndex(records);

  console.log(`Exported ${records.length} PDFs.`);
  console.log(`Output root: ${toPosix(path.relative(process.cwd(), outputRoot))}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
