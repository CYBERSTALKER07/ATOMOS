import re
with open('app/components/About.tsx', 'r') as f:
    content = f.read()

content = content.replace("url(/pegasus-logo-mask.svg)", "url(/productionlogo-mask.png)")

with open('app/components/About.tsx', 'w') as f:
    f.write(content)
