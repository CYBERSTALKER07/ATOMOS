import os
import re
import sys
import json
from pathlib import Path

FILES = [
    # Pegasus
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus/agents.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/ARCHITECTURE.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/BACKEND_SERVICES.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/FEATURES_AND_PORTALS.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/INFRASTRUCTURE.md",

    # PegasusX
    "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/agents.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/ARCHITECTURE.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/BACKEND_SERVICES.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/FEATURES_AND_ROLE_ROWS.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/INFRASTRUCTURE.md",

    # Pegasus.x
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/agents.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/ARCHITECTURE.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/BACKEND_AND_PLANNING.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/FEATURES_AND_APPS.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/INFRASTRUCTURE.md",
]

def check_file_line(path, line_str):
    try:
        with open(path, 'r', encoding='utf-8', errors='ignore') as f:
            total_lines = sum(1 for _ in f)
        
        lines = re.findall(r'\d+', line_str)
        if not lines:
            return True, total_lines, f"Valid file ({total_lines} lines)"
        
        for l in lines:
            val = int(l)
            if val > total_lines:
                return False, total_lines, f"Line {val} exceeds file length {total_lines}"
            if val <= 0:
                return False, total_lines, f"Line {val} is invalid <= 0"
        return True, total_lines, f"Valid line anchor (file has {total_lines} lines)"
    except Exception as e:
        return False, 0, str(e)

def parse_link(raw_match):
    # raw_match is e.g. "file:///Users/shakhzod/...#L12-L30" or "file:///Users/.../main.go:123"
    # First, strip trailing markdown / sentence punctuation: `, ), ], >, `, ., ;, "
    cleaned = raw_match.rstrip('`)]>"\',.;:')

    # Now separate line spec
    # Two common patterns:
    # 1. file:///path#L10-L20 or file:///path#10
    # 2. file:///path:10 or file:///path:10-20
    # Note: don't confuse drive letters (though we are on mac: /Users/...)
    anchor = ""
    target_path = cleaned

    if '#' in cleaned:
        target_path, anchor = cleaned.split('#', 1)
    elif re.search(r':\d+(?:-\d+)?$', cleaned):
        m = re.search(r'^(.*):(\d+(?:-\d+)?)$', cleaned)
        if m:
            target_path = m.group(1)
            anchor = m.group(2)

    # strip file:// or file:///
    if target_path.startswith("file://"):
        target_path = target_path[7:]
    
    return cleaned, target_path, anchor

def audit():
    results = {
        "files": {},
        "summary": {
            "total_files": len(FILES),
            "files_found": 0,
            "total_links": 0,
            "resolved_links": 0,
            "broken_links": 0,
            "line_errors": 0,
            "unique_resolved_files": set(),
            "unique_missing_files": set()
        }
    }

    # Regex to find any occurrence of file:///...
    # Up to delimiter: whitespace, newline, backtick, quotes, angle brackets, parentheses, square brackets
    link_pattern = re.compile(r'file:///[^\s\"\'\`\<\>\)\]]+')

    for file_path in FILES:
        file_info = {
            "path": file_path,
            "exists": os.path.exists(file_path),
            "size": 0,
            "lines": 0,
            "links_count": 0,
            "resolved_links_count": 0,
            "broken_links_count": 0,
            "line_error_count": 0,
            "links": [],
            "keywords": {}
        }

        if not file_info["exists"]:
            results["files"][file_path] = file_info
            continue

        results["summary"]["files_found"] += 1
        file_info["size"] = os.path.getsize(file_path)
        with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
            content = f.read()

        lines = content.splitlines()
        file_info["lines"] = len(lines)

        # Check keywords for R1 & R2
        file_info["keywords"]["zero_theatre"] = bool(re.search(r'zero[\s\-_]?theatre', content, re.I))
        file_info["keywords"]["honesty"] = bool(re.search(r'honesty', content, re.I))
        file_info["keywords"]["what_it_is"] = bool(re.search(r'what it is', content, re.I))
        file_info["keywords"]["how_it_works"] = bool(re.search(r'how it works', content, re.I))
        file_info["keywords"]["why_it_is_there"] = bool(re.search(r'why it is there', content, re.I))
        file_info["keywords"]["todos"] = len(re.findall(r'\bTODO\b|\bTBD\b|\bFIXME\b', content))

        raw_matches = link_pattern.findall(content)
        for raw in raw_matches:
            results["summary"]["total_links"] += 1
            file_info["links_count"] += 1

            cleaned, target_path, anchor = parse_link(raw)

            target_exists = os.path.exists(target_path)
            link_record = {
                "raw": raw,
                "cleaned": cleaned,
                "target_path": target_path,
                "anchor": anchor,
                "exists": target_exists,
                "status": "PASS",
                "details": ""
            }

            if not target_exists:
                link_record["status"] = "BROKEN_FILE"
                link_record["details"] = f"Path '{target_path}' does not exist on disk"
                file_info["broken_links_count"] += 1
                results["summary"]["broken_links"] += 1
                results["summary"]["unique_missing_files"].add(target_path)
            else:
                results["summary"]["unique_resolved_files"].add(target_path)
                if anchor and os.path.isfile(target_path):
                    valid_line, tot_l, det = check_file_line(target_path, anchor)
                    if not valid_line:
                        link_record["status"] = "LINE_OUT_OF_BOUNDS"
                        link_record["details"] = det
                        file_info["line_error_count"] += 1
                        results["summary"]["line_errors"] += 1
                    else:
                        file_info["resolved_links_count"] += 1
                        results["summary"]["resolved_links"] += 1
                else:
                    file_info["resolved_links_count"] += 1
                    results["summary"]["resolved_links"] += 1

            file_info["links"].append(link_record)

        results["files"][file_path] = file_info

    results["summary"]["unique_resolved_files"] = sorted(list(results["summary"]["unique_resolved_files"]))
    results["summary"]["unique_missing_files"] = sorted(list(results["summary"]["unique_missing_files"]))
    return results

if __name__ == "__main__":
    res = audit()
    out_path = "/Users/shakhzod/Desktop/V.O.I.D/.agents/reviewer_2/audit_data.json"
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump(res, f, indent=2)

    print("=== PROGRAMMATIC AUDIT RESULTS ===")
    print(f"Total files audited: {res['summary']['total_files']}")
    print(f"Files found: {res['summary']['files_found']}")
    print(f"Total file:/// links: {res['summary']['total_links']}")
    print(f"Resolved links (Valid path & line): {res['summary']['resolved_links']}")
    print(f"Broken links (File not found): {res['summary']['broken_links']}")
    print(f"Line errors (Line out of bounds): {res['summary']['line_errors']}")
    print(f"Unique resolved targets on disk: {len(res['summary']['unique_resolved_files'])}")
    print(f"Unique missing targets on disk: {len(res['summary']['unique_missing_files'])}")
    
    print("\n--- PER-FILE BREAKDOWN ---")
    for fpath, finfo in res["files"].items():
        rel = fpath.replace("/Users/shakhzod/Desktop/V.O.I.D/", "")
        print(f"[{rel}] Lines: {finfo['lines']}, Size: {finfo['size']}B, Links: {finfo['links_count']} (PASS: {finfo['resolved_links_count']}, BROKEN: {finfo['broken_links_count']}, LINE_ERR: {finfo['line_error_count']})")

    if res['summary']['broken_links'] > 0 or res['summary']['line_errors'] > 0:
        print("\n--- ISSUES DETECTED ---")
        for fpath, finfo in res["files"].items():
            if finfo["broken_links_count"] > 0 or finfo["line_error_count"] > 0:
                rel = fpath.replace("/Users/shakhzod/Desktop/V.O.I.D/", "")
                print(f"\nIn {rel}:")
                for l in finfo["links"]:
                    if l["status"] != "PASS":
                        print(f"  [{l['status']}] {l['raw']} (parsed: {l['target_path']}) -> {l['details']}")
