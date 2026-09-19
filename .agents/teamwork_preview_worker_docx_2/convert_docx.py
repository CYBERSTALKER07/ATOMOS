import os
import sys
import zipfile
import re
import xml.etree.ElementTree as ET

NS = {
    'w': 'http://schemas.openxmlformats.org/wordprocessingml/2006/main',
    'r': 'http://schemas.openxmlformats.org/officeDocument/2006/relationships',
    'a': 'http://schemas.openxmlformats.org/drawingml/2006/main',
    'rel': 'http://schemas.openxmlformats.org/package/2006/relationships'
}

def get_relationships(zf):
    rels = {}
    if 'word/_rels/document.xml.rels' in zf.namelist():
        xml_content = zf.read('word/_rels/document.xml.rels')
        root = ET.fromstring(xml_content)
        for rel in root.findall('{http://schemas.openxmlformats.org/package/2006/relationships}Relationship'):
            r_id = rel.get('Id')
            target = rel.get('Target')
            rels[r_id] = target
    return rels

def parse_run(r_elem, rels):
    text_pieces = []
    rPr = r_elem.find('w:rPr', NS)
    is_bold = False
    is_italic = False
    is_code = False
    is_strike = False

    if rPr is not None:
        b_elem = rPr.find('w:b', NS)
        if b_elem is not None and b_elem.get(f"{{{NS['w']}}}val") != "0":
            is_bold = True
        bcs_elem = rPr.find('w:bCs', NS)
        if bcs_elem is not None and bcs_elem.get(f"{{{NS['w']}}}val") != "0":
            is_bold = True

        i_elem = rPr.find('w:i', NS)
        if i_elem is not None and i_elem.get(f"{{{NS['w']}}}val") != "0":
            is_italic = True
        ics_elem = rPr.find('w:iCs', NS)
        if ics_elem is not None and ics_elem.get(f"{{{NS['w']}}}val") != "0":
            is_italic = True

        if rPr.find('w:strike', NS) is not None:
            is_strike = True

        rStyle = rPr.find('w:rStyle', NS)
        if rStyle is not None:
            s_val = rStyle.get(f"{{{NS['w']}}}val", "").lower()
            if s_val in ['code', 'verbatimchar', 'macro', 'sourcecode']:
                is_code = True

        rFonts = rPr.find('w:rFonts', NS)
        if rFonts is not None:
            ascii_font = (rFonts.get(f"{{{NS['w']}}}ascii") or "").lower()
            hAnsi_font = (rFonts.get(f"{{{NS['w']}}}hAnsi") or "").lower()
            if any(k in ascii_font or k in hAnsi_font for k in ['consolas', 'courier', 'monospace', 'menlo', 'monaco']):
                is_code = True

    for child in r_elem:
        tag = child.tag.split('}')[-1]
        if tag == 't':
            text_pieces.append(child.text or '')
        elif tag == 'tab':
            text_pieces.append('    ')
        elif tag == 'br':
            text_pieces.append('\n')

    raw_text = "".join(text_pieces)
    if not raw_text:
        return ""

    formatted = raw_text
    # Avoid applying formatting markers if text is purely whitespace
    if formatted.strip() == "":
        return formatted

    # Handle leading/trailing spaces when formatting to avoid ` **text** ` vs `** text **`
    leading_ws = len(formatted) - len(formatted.lstrip())
    trailing_ws = len(formatted) - len(formatted.rstrip())
    core_text = formatted.strip()

    prefix = formatted[:leading_ws] if leading_ws > 0 else ""
    suffix = formatted[len(formatted)-trailing_ws:] if trailing_ws > 0 else ""

    if is_code:
        core_text = f"`{core_text}`"
    if is_bold and is_italic:
        core_text = f"***{core_text}***"
    elif is_bold:
        core_text = f"**{core_text}**"
    elif is_italic:
        core_text = f"*{core_text}*"
    if is_strike:
        core_text = f"~~{core_text}~~"

    return prefix + core_text + suffix

def parse_paragraph(p_elem, rels, in_table=False):
    pPr = p_elem.find('w:pPr', NS)
    style_val = ""
    is_list = False
    is_numbered = False
    list_lvl = 0

    if pPr is not None:
        pStyle = pPr.find('w:pStyle', NS)
        if pStyle is not None:
            style_val = pStyle.get(f"{{{NS['w']}}}val", "")
        numPr = pPr.find('w:numPr', NS)
        if numPr is not None:
            is_list = True
            ilvl_elem = numPr.find('w:ilvl', NS)
            if ilvl_elem is not None:
                try:
                    list_lvl = int(ilvl_elem.get(f"{{{NS['w']}}}val", "0"))
                except ValueError:
                    list_lvl = 0
            numId_elem = numPr.find('w:numId', NS)
            if numId_elem is not None:
                num_id = numId_elem.get(f"{{{NS['w']}}}val", "")
                if num_id and num_id != "0":
                    # Mark as list
                    is_list = True

    if "bullet" in style_val.lower():
        is_list = True
    elif "number" in style_val.lower():
        is_list = True
        is_numbered = True

    runs_text = []
    for child in p_elem:
        tag = child.tag.split('}')[-1]
        if tag == 'r':
            runs_text.append(parse_run(child, rels))
        elif tag == 'hyperlink':
            r_id = child.get(f"{{{NS['r']}}}id")
            target = rels.get(r_id, "")
            link_runs = [parse_run(r, rels) for r in child.findall('w:r', NS)]
            link_text = "".join(link_runs).strip()
            if target and link_text:
                runs_text.append(f"[{link_text}]({target})")
            elif link_text:
                runs_text.append(link_text)

    p_text = "".join(runs_text).strip()
    return p_text, style_val, is_list, is_numbered, list_lvl

def parse_table(tbl_elem, rels):
    rows = []
    for tr in tbl_elem.findall('w:tr', NS):
        row_cells = []
        for tc in tr.findall('w:tc', NS):
            # Check gridSpan
            span = 1
            tcPr = tc.find('w:tcPr', NS)
            if tcPr is not None:
                gridSpan = tcPr.find('w:gridSpan', NS)
                if gridSpan is not None:
                    try:
                        span = int(gridSpan.get(f"{{{NS['w']}}}val", "1"))
                    except ValueError:
                        span = 1

            cell_paras = []
            for p in tc.findall('w:p', NS):
                p_text, _, is_list, _, _ = parse_paragraph(p, rels, in_table=True)
                if p_text:
                    if is_list and not p_text.startswith('-'):
                        p_text = f"• {p_text}"
                    cell_paras.append(p_text)
            cell_content = "<br>".join(cell_paras) if cell_paras else ""
            cell_content = cell_content.replace("|", "\\|").replace("\n", "<br>")
            row_cells.append(cell_content)
            for _ in range(span - 1):
                row_cells.append("")
        if row_cells and any(c.strip() for c in row_cells):
            rows.append(row_cells)

    if not rows:
        return ""

    max_cols = max(len(r) for r in rows)
    normalized_rows = []
    for r in rows:
        normalized_rows.append(r + [""] * (max_cols - len(r)))

    md_lines = []
    header = normalized_rows[0]
    md_lines.append("| " + " | ".join(header) + " |")
    md_lines.append("| " + " | ".join(["---"] * max_cols) + " |")
    for row in normalized_rows[1:]:
        md_lines.append("| " + " | ".join(row) + " |")

    return "\n".join(md_lines)

def convert_docx_file_to_md(docx_path):
    with zipfile.ZipFile(docx_path, 'r') as zf:
        rels = get_relationships(zf)
        doc_xml = zf.read('word/document.xml')
        root = ET.fromstring(doc_xml)
        body = root.find('w:body', NS)
        if body is None:
            return ""

        output_blocks = []
        list_counters = {}

        for child in body:
            tag = child.tag.split('}')[-1]
            if tag == 'p':
                p_text, style_val, is_list, is_numbered, list_lvl = parse_paragraph(child, rels)
                if not p_text:
                    continue

                style_lower = style_val.lower()
                indent = "  " * list_lvl

                # Check if it starts with markdown heading already or should be formatted as heading
                if "title" in style_lower:
                    if not p_text.startswith('#'):
                        p_text = f"# {p_text}"
                    output_blocks.append(p_text)
                    list_counters.clear()
                elif "subtitle" in style_lower:
                    if not p_text.startswith('#'):
                        p_text = f"## {p_text}"
                    output_blocks.append(p_text)
                    list_counters.clear()
                elif "heading1" in style_lower or "heading 1" in style_lower:
                    # Strip existing # if any to avoid ## # Heading
                    clean_text = p_text.lstrip('#').strip()
                    output_blocks.append(f"# {clean_text}")
                    list_counters.clear()
                elif "heading2" in style_lower or "heading 2" in style_lower:
                    clean_text = p_text.lstrip('#').strip()
                    output_blocks.append(f"## {clean_text}")
                    list_counters.clear()
                elif "heading3" in style_lower or "heading 3" in style_lower:
                    clean_text = p_text.lstrip('#').strip()
                    output_blocks.append(f"### {clean_text}")
                    list_counters.clear()
                elif "heading4" in style_lower or "heading 4" in style_lower:
                    clean_text = p_text.lstrip('#').strip()
                    output_blocks.append(f"#### {clean_text}")
                    list_counters.clear()
                elif "heading5" in style_lower or "heading 5" in style_lower:
                    clean_text = p_text.lstrip('#').strip()
                    output_blocks.append(f"##### {clean_text}")
                    list_counters.clear()
                elif "heading6" in style_lower or "heading 6" in style_lower:
                    clean_text = p_text.lstrip('#').strip()
                    output_blocks.append(f"###### {clean_text}")
                    list_counters.clear()
                elif is_list:
                    # Clean up if already starts with bullet or number
                    if p_text.startswith('- ') or p_text.startswith('* ') or re.match(r'^\d+\.\s', p_text):
                        output_blocks.append(f"{indent}{p_text}")
                    elif is_numbered:
                        ctr = list_counters.get(list_lvl, 1)
                        output_blocks.append(f"{indent}{ctr}. {p_text}")
                        list_counters[list_lvl] = ctr + 1
                    else:
                        output_blocks.append(f"{indent}- {p_text}")
                else:
                    # Reset list counters on regular paragraph
                    list_counters.clear()
                    output_blocks.append(p_text)

            elif tag == 'tbl':
                list_counters.clear()
                tbl_md = parse_table(child, rels)
                if tbl_md:
                    output_blocks.append(tbl_md)

        # Post-processing: clean excessive blank lines and formatting artifacts
        result = "\n\n".join(output_blocks)
        result = re.sub(r'\n{3,}', '\n\n', result)
        return result.strip() + "\n"

if __name__ == "__main__":
    for f in sys.argv[1:]:
        print(f"Converting {f}...")
        md_content = convert_docx_file_to_md(f)
        out_name = os.path.splitext(f)[0] + ".md"
        print(f"-> {out_name} ({len(md_content)} chars)")
