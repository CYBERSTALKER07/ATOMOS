import re

with open('app/components/About.tsx', 'r') as f:
    content = f.read()

target = "relative h-[240px] sm:h-[320px] md:h-[400px] lg:h-[500px]"
replacement = "relative h-[120px] sm:h-[160px] md:h-[200px] lg:h-[240px]"

content = content.replace(target, replacement)

with open('app/components/About.tsx', 'w') as f:
    f.write(content)
