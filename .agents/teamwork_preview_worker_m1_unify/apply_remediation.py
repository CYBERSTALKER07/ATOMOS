import os
import re
import json

base = '/Users/shakhzod/Desktop/V.O.I.D'
apps_to_audit = [
    ('pegasus', 'admin-portal'),
    ('pegasus', 'factory-portal'),
    ('pegasus', 'payload-terminal'),
    ('pegasus', 'retailer-app-desktop'),
    ('pegasus', 'warehouse-portal'),
    ('pegasus.x', 'payloader-tablet'),
    ('pegasus.x', 'retailer-desktop'),
    ('pegasus.x', 'supplier-desktop'),
    ('pegasus.x', 'telegram-miniapp'),
    ('pegasus.x', 'warehouse-desktop'),
    ('pegasusX', 'admin-portal'),
    ('pegasusX', 'factory-portal'),
    ('pegasusX', 'payload-terminal'),
    ('pegasusX', 'retailer-app-desktop'),
    ('pegasusX', 'supplier-portal'),
    ('pegasusX', 'warehouse-portal'),
]

def find_tag_end(content, start):
    pos = start + len('<input')
    in_str = None
    brace_depth = 0
    escape = False
    for i in range(pos, len(content)):
        ch = content[i]
        if escape:
            escape = False
            continue
        if ch == '\\' and in_str:
            escape = True
            continue
        if in_str:
            if ch == in_str:
                in_str = None
            continue
        if ch in ('"', "'", '`'):
            in_str = ch
            continue
        if ch == '{':
            brace_depth += 1
        elif ch == '}':
            brace_depth -= 1
        elif brace_depth == 0:
            if ch == '/' and i + 1 < len(content) and content[i+1] == '>':
                return i + 2
            elif ch == '>':
                return i + 1
    return -1

def clean_slug(s):
    s = re.sub(r'[^a-zA-Z0-9]+', '-', s.strip()).strip('-').lower()
    return s or 'input'

def humanize(s):
    s = re.sub(r'([a-z])([A-Z])', r'\1 \2', s)
    s = re.sub(r'[^a-zA-Z0-9]+', ' ', s).strip()
    return s.capitalize() or 'Input'

def extract_label_from_preceding(text):
    m = re.findall(r'<label[^>]*>(.*?)(?:</label>|$)', text, re.DOTALL)
    if m:
        last = m[-1]
        cleaned = re.sub(r'<[^>]+>', ' ', last).strip()
        cleaned = re.sub(r'[\r\n\t]+', ' ', cleaned)
        cleaned = re.sub(r'\s+', ' ', cleaned).strip()
        t_match = re.search(r'[\"\']([^\"\']{2,40})[\"\']', cleaned)
        if '{' in cleaned and t_match:
            cand = t_match.group(1).split('.')[-1]
            return humanize(cand)
        if cleaned and len(cleaned) < 60 and not cleaned.startswith('{'):
            return cleaned
    return None

scanner_pattern = re.compile(r'<input(?![^>]*(aria-label|id=|aria-labelledby))[^>]*>')

total_files_modified = 0
total_inputs_modified = 0
modified_files = []

for tree, app in apps_to_audit:
    p = os.path.join(base, tree, 'apps', app)
    if not os.path.exists(p):
        continue

    for root, dirs, files in os.walk(p):
        if any(skip in root for skip in ['node_modules', '.next', 'dist', 'build', '.git', 'src-tauri/target']):
            continue
        for f in files:
            if not f.endswith(('.tsx', '.jsx', '.html')):
                continue
            filepath = os.path.join(root, f)
            with open(filepath, 'r', encoding='utf-8', errors='ignore') as code_file:
                content = code_file.read()

            if not scanner_pattern.search(content):
                continue

            file_slug = clean_slug(os.path.splitext(f)[0])
            counter = 0
            new_content = content

            matches = list(scanner_pattern.finditer(content))
            matches.sort(key=lambda m: m.start(), reverse=True)

            for m in matches:
                start = m.start()
                end = find_tag_end(content, start)
                if end == -1:
                    continue

                counter += 1
                tag_str = content[start:end]

                id_match = re.search(r'\bid=(?:\"([^\"]*)\"|\'([^\']*)\'|\{([^}]*)\})', tag_str)
                aria_match = re.search(r'\baria-label=(?:\"([^\"]*)\"|\'([^\']*)\'|\{([^}]*)\})', tag_str)

                existing_id = None
                existing_aria = None

                if id_match:
                    existing_id = id_match.group(1) or id_match.group(2) or id_match.group(3)
                    tag_str = tag_str[:id_match.start()] + tag_str[id_match.end():]

                if aria_match:
                    existing_aria = aria_match.group(1) or aria_match.group(2) or (f"{{{aria_match.group(3)}}}" if aria_match.group(3) else None)
                    re_aria = re.search(r'\baria-label=(?:\"([^\"]*)\"|\'([^\']*)\'|\{([^}]*)\})', tag_str)
                    if re_aria:
                        tag_str = tag_str[:re_aria.start()] + tag_str[re_aria.end():]

                # Preceding context
                prec = content[max(0, start - 300):start]
                lbl = extract_label_from_preceding(prec)
                name_m = re.search(r'\bname=[\"\']([^\"\']+)[\"\']', tag_str)
                ph_m = re.search(r'\bplaceholder=[\"\']([^\"\']+)[\"\']', tag_str)
                val_m = re.search(r'\bvalue=\{([a-zA-Z0-9_\.]+)\}', tag_str)
                type_m = re.search(r'\btype=[\"\']([^\"\']+)[\"\']', tag_str)

                if not existing_id:
                    if lbl:
                        existing_id = f"{clean_slug(lbl)[:25]}-input-{counter}"
                    elif name_m:
                        existing_id = f"{clean_slug(name_m.group(1))}-input-{counter}"
                    elif val_m:
                        var_name = val_m.group(1).split('.')[-1]
                        existing_id = f"{clean_slug(var_name)}-input-{counter}"
                    elif ph_m:
                        existing_id = f"{clean_slug(ph_m.group(1))[:25]}-input-{counter}"
                    elif type_m:
                        existing_id = f"{file_slug}-{clean_slug(type_m.group(1))}-{counter}"
                    else:
                        existing_id = f"{file_slug}-input-{counter}"

                if not existing_aria:
                    if lbl:
                        existing_aria = lbl.replace('"', '&quot;')
                    elif ph_m:
                        existing_aria = ph_m.group(1).replace('"', '&quot;')
                    elif name_m:
                        existing_aria = humanize(name_m.group(1))
                    elif val_m:
                        var_name = val_m.group(1).split('.')[-1]
                        existing_aria = humanize(var_name)
                    elif type_m and type_m.group(1) == 'checkbox':
                        existing_aria = f"Select {humanize(file_slug)} option"
                    elif type_m and type_m.group(1) == 'search':
                        existing_aria = "Search"
                    elif type_m and type_m.group(1) == 'date':
                        existing_aria = "Select date"
                    elif type_m and type_m.group(1) == 'file':
                        existing_aria = "Upload file"
                    else:
                        existing_aria = f"{humanize(file_slug)} input field"

                existing_id = clean_slug(str(existing_id))
                if existing_aria.startswith('{') and existing_aria.endswith('}'):
                    aria_attr = f"aria-label={existing_aria}"
                else:
                    clean_lbl = existing_aria.replace('"', '')
                    aria_attr = f'aria-label="{clean_lbl}"'

                id_attr = f'id="{existing_id}"'

                m_lead = re.match(r'<input(\s+)', tag_str)
                if not m_lead:
                    continue

                ws = m_lead.group(1)
                rest = tag_str[m_lead.end():].lstrip()

                if '\n' in ws:
                    indent = ws.split('\n')[-1]
                    new_tag = f"<input\n{indent}{id_attr}\n{indent}{aria_attr}\n{indent}{rest}"
                else:
                    new_tag = f"<input {id_attr} {aria_attr} {rest}"

                new_content = new_content[:start] + new_tag + new_content[end:]
                total_inputs_modified += 1

            # Write file back
            with open(filepath, 'w', encoding='utf-8') as out_f:
                out_f.write(new_content)

            total_files_modified += 1
            modified_files.append(os.path.relpath(filepath, base))

print(f"Applied input label pairing remediation.")
print(f"Total files modified: {total_files_modified}")
print(f"Total inputs modified: {total_inputs_modified}")
