import os
import glob
import re

files_to_process = glob.glob('pegasus/apps/factory-portal/**/*.tsx', recursive=True)

for filepath in files_to_process:
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    # Strip shadow classes and hover-lift
    new_content = re.sub(r'\bshadow-(?:sm|md|lg|xl|2xl|inner|none|\[.*?\])\b', '', content)
    new_content = re.sub(r'\bmd-elevation-\d\b', '', new_content)
    new_content = re.sub(r'\bhover-lift\b', '', new_content)
    new_content = re.sub(r'\bglass-premium\b', 'md-card', new_content) # replace glass with generic card

    # Clean up double spaces in classNames
    new_content = re.sub(r'  +', ' ', new_content)
    new_content = new_content.replace('className=" "', 'className=""')

    # Fix HeroUI Button variant="flat" -> variant="primary" if it exists, or just let TS complain and fix it.
    # Actually wait flat is valid in some versions but earlier we changed flat to primary.
    # Let's string replace variant="flat" color="primary" to variant="primary" color="primary"
    
    if content \!= new_content:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(new_content)
        print(f"Patched {filepath}")

