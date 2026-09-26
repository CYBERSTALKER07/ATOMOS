import sys
import re

file_path = "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/softwareengineercv-main/app/components/SiteAssistant.tsx"
with open(file_path, "r") as f:
    content = f.read()

# Add GooeyAgent import if it doesn't exist
if "import GooeyAgent" not in content:
    content = content.replace(
        "import React, {",
        "import GooeyAgent from '@/app/components/visuals/GooeyAgent';\nimport React, {"
    )

# Replace the img tag with GooeyAgent
img_pattern = r'<img\s+src="/pegasus\.jpg"[\s\S]*?/>'
replacement = '<GooeyAgent color="#ffffff" size={34} className="transition-transform duration-200 group-hover:scale-110" />'

content = re.sub(img_pattern, replacement, content)

with open(file_path, "w") as f:
    f.write(content)

print("Updated SiteAssistant.tsx with GooeyAgent")
