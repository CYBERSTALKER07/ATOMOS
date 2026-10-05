import re

with open('app/layout.tsx', 'r') as f:
    content = f.read()

# Add Mr Dafoe and Mrs Saint Delafield
content = content.replace(
    "import { Libre_Baskerville, Libre_Franklin, Caveat, Great_Vibes } from 'next/font/google';",
    "import { Libre_Baskerville, Libre_Franklin, Caveat, Great_Vibes, Herr_Von_Muellerhoff, Mrs_Saint_Delafield } from 'next/font/google';"
)

font_defs = """
const herrVon = Herr_Von_Muellerhoff({
  subsets: ['latin'],
  weight: ['400'],
  variable: '--font-herr-von',
});

const mrsSaint = Mrs_Saint_Delafield({
  subsets: ['latin'],
  weight: ['400'],
  variable: '--font-mrs-saint',
});
"""

content = content.replace(
    "const libreBaskerville =",
    font_defs + "\nconst libreBaskerville ="
)

# Add to body className
content = re.sub(
    r'\$\{greatVibes.variable\}',
    r'${greatVibes.variable} ${herrVon.variable} ${mrsSaint.variable}',
    content
)

with open('app/layout.tsx', 'w') as f:
    f.write(content)
