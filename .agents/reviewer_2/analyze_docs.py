import os
import re

DOC_FILES = [
    # Pegasus
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/ARCHITECTURE.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/BACKEND_SERVICES.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/FEATURES_AND_PORTALS.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/INFRASTRUCTURE.md",

    # PegasusX
    "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/ARCHITECTURE.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/BACKEND_SERVICES.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/FEATURES_AND_ROLE_ROWS.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/INFRASTRUCTURE.md",

    # Pegasus.x
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/ARCHITECTURE.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/BACKEND_AND_PLANNING.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/FEATURES_AND_APPS.md",
    "/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/INFRASTRUCTURE.md",
]

def analyze_docs():
    for df in DOC_FILES:
        rel = df.replace("/Users/shakhzod/Desktop/V.O.I.D/", "")
        with open(df, 'r', encoding='utf-8') as f:
            content = f.read()

        words = len(content.split())
        h1 = re.findall(r'^#\s+(.+)$', content, re.M)
        h2 = re.findall(r'^##\s+(.+)$', content, re.M)
        h3 = re.findall(r'^###\s+(.+)$', content, re.M)

        # Check for What it is / How it works / Why it is there
        has_what = len(re.findall(r'what\s+it\s+is', content, re.I))
        has_how = len(re.findall(r'how\s+it\s+works', content, re.I))
        has_why = len(re.findall(r'why\s+it\s+is\s+there', content, re.I))

        print(f"\n==========================================")
        print(f"File: {rel}")
        print(f"Title: {h1[0] if h1 else 'None'}")
        print(f"Stats: {len(content.splitlines())} lines, {words} words, {len(content)} bytes")
        print(f"Triad matches: What it is ({has_what}), How it works ({has_how}), Why it is there ({has_why})")
        print(f"Sections (H2 count: {len(h2)}, H3 count: {len(h3)}):")
        for s in h2:
            print(f"  - {s}")

if __name__ == "__main__":
    analyze_docs()
