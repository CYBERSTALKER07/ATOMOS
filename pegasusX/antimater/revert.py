import re
with open('app/components/About.tsx', 'r') as f:
    content = f.read()

content = content.replace("url(/productionlogo-mask.png)", "url(/pegasus-logo-mask.svg)")

with open('app/components/About.tsx', 'w') as f:
    f.write(content)
