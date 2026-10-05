import re

with open('app/components/About.tsx', 'r') as f:
    content = f.read()

# Change height back to large sizes
target_height = "relative h-[120px] sm:h-[160px] md:h-[200px] lg:h-[240px]"
replacement_height = "relative h-[240px] sm:h-[320px] md:h-[400px] lg:h-[500px]"
content = content.replace(target_height, replacement_height)

# Change mask URL
content = content.replace("url(/pegasus-logo-mask.svg)", "url(/productionlogo-mask.png)")

with open('app/components/About.tsx', 'w') as f:
    f.write(content)
