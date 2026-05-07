const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

const defaultInput = path.resolve(
  __dirname,
  '../../text-architecture/pegasus-protected-technical-brief.md'
);
const defaultOutput = path.resolve(
  __dirname,
  '../../text-architecture-pdf/pegasus-protected-technical-brief.pdf'
);

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
  s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  s = s.replace(/`([^`]+)`/g, '<code>$1</code>');
  return s;
}

function markdownToHtml(markdown) {
  const lines = markdown.split(/\r?\n/);
  const out = [];
  let inCode = false;
  let inMath = false;
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
    if (line.trim() === '$$') {
      closeParagraph();
      closeLists();
      if (!inMath) {
        out.push('<div class="math-block">');
        inMath = true;
      } else {
        out.push('</div>');
        inMath = false;
      }
      continue;
    }

    if (line.startsWith('```')) {
      closeParagraph();
      closeLists();
      if (!inCode) {
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

    if (inMath) {
      out.push(`<div>${escapeHtml(line)}</div>`);
      continue;
    }

    const trimmed = line.trim();

    if (!trimmed) {
      closeParagraph();
      closeLists();
      continue;
    }

    const h = trimmed.match(/^(#{1,6})\s+(.+)$/);
    if (h) {
      closeParagraph();
      closeLists();
      const level = h[1].length;
      out.push(`<h${level}>${renderInline(h[2])}</h${level}>`);
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

    if (!inParagraph) {
      out.push('<p>');
      inParagraph = true;
      out.push(renderInline(trimmed));
    } else {
      out.push(`<br>${renderInline(trimmed)}`);
    }
  }

  if (inParagraph) out.push('</p>');
  if (inUl) out.push('</ul>');
  if (inOl) out.push('</ol>');
  if (inCode) out.push('</code></pre>');
  if (inMath) out.push('</div>');

  return out.join('\n');
}

function buildHtml(title, bodyHtml) {
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>${escapeHtml(title)}</title>
  <style>
    @page {
      size: A4;
      margin: 16mm 14mm 16mm 14mm;
    }
    html, body {
      margin: 0;
      padding: 0;
      color: #111;
      background: #fff;
      font-family: "Charter", "Palatino", "Times New Roman", serif;
      font-size: 11.5px;
      line-height: 1.48;
    }
    body::before {
      content: "CONTROLLED DISCLOSURE";
      position: fixed;
      top: 45%;
      left: 20%;
      transform: rotate(-28deg);
      font-size: 42px;
      color: rgba(0,0,0,0.045);
      letter-spacing: 2px;
      z-index: 0;
      pointer-events: none;
    }
    .page {
      position: relative;
      z-index: 1;
    }
    .cover {
      border: 1px solid #222;
      padding: 22px;
      margin-bottom: 16px;
      page-break-inside: avoid;
    }
    .cover .label {
      font-size: 10px;
      letter-spacing: 1.2px;
      text-transform: uppercase;
      color: #333;
      margin-bottom: 8px;
    }
    .cover h1 {
      margin: 0;
      font-size: 24px;
      line-height: 1.2;
    }
    .cover .sub {
      margin-top: 12px;
      font-size: 12px;
      color: #333;
    }
    h1, h2, h3, h4 {
      margin: 12px 0 6px;
      line-height: 1.25;
      page-break-after: avoid;
    }
    h1 { font-size: 22px; }
    h2 {
      font-size: 15px;
      text-transform: uppercase;
      letter-spacing: 0.4px;
      border-bottom: 1px solid #ddd;
      padding-bottom: 4px;
    }
    h3 { font-size: 13px; }
    h4 { font-size: 12px; }
    p {
      margin: 7px 0;
      orphans: 3;
      widows: 3;
    }
    ul, ol {
      margin: 4px 0 8px 20px;
      padding: 0;
    }
    li { margin: 3px 0; }
    .math-block {
      margin: 8px 0;
      padding: 8px 10px;
      border-left: 3px solid #222;
      background: #fafafa;
      font-family: "Menlo", "Consolas", monospace;
      font-size: 10.5px;
      page-break-inside: avoid;
      white-space: pre-wrap;
      word-break: break-word;
    }
    pre {
      margin: 8px 0;
      padding: 8px 10px;
      border: 1px solid #ddd;
      background: #f7f7f7;
      font-family: "Menlo", "Consolas", monospace;
      font-size: 10.2px;
      white-space: pre-wrap;
      word-break: break-word;
      page-break-inside: avoid;
    }
    code {
      font-family: "Menlo", "Consolas", monospace;
      font-size: 0.95em;
    }
    .footer-note {
      margin-top: 18px;
      padding-top: 8px;
      border-top: 1px solid #ddd;
      font-size: 10px;
      color: #444;
    }
  </style>
</head>
<body>
  <div class="page">
    <section class="cover">
      <div class="label">Patent-Oriented Technical Dossier</div>
      <h1>${escapeHtml(title)}</h1>
      <div class="sub">Professional summary aligned for architecture, logic, formula landscapes, and controlled disclosure boundaries.</div>
    </section>
    ${bodyHtml}
    <div class="footer-note">Generated for controlled technical review. This document intentionally omits implementation-critical secrets.</div>
  </div>
</body>
</html>`;
}

async function main() {
  const inputPath = process.argv[2] ? path.resolve(process.argv[2]) : defaultInput;
  const outputPath = process.argv[3] ? path.resolve(process.argv[3]) : defaultOutput;

  if (!fs.existsSync(inputPath)) {
    throw new Error(`Input markdown not found: ${inputPath}`);
  }

  fs.mkdirSync(path.dirname(outputPath), { recursive: true });

  const markdown = fs.readFileSync(inputPath, 'utf8');
  const titleMatch = markdown.match(/^#\s+(.+)$/m);
  const title = titleMatch ? titleMatch[1].trim() : path.basename(inputPath, '.md');

  const html = buildHtml(title, markdownToHtml(markdown));

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  try {
    await page.setContent(html, { waitUntil: 'load' });
    await page.emulateMedia({ media: 'print' });
    await page.pdf({
      path: outputPath,
      format: 'A4',
      printBackground: true,
      margin: {
        top: '14mm',
        right: '12mm',
        bottom: '14mm',
        left: '12mm'
      }
    });
  } finally {
    await page.close();
    await browser.close();
  }

  console.log(`Input: ${inputPath}`);
  console.log(`Output: ${outputPath}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
