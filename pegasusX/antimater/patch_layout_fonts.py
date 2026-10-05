import re

with open('app/layout.tsx', 'r') as f:
    content = f.read()

# Add imports
content = content.replace(
    "import { Libre_Baskerville, Libre_Franklin } from 'next/font/google';",
    "import { Libre_Baskerville, Libre_Franklin, Caveat, Great_Vibes } from 'next/font/google';"
)

# Add font definitions
font_defs = """
const caveat = Caveat({
  subsets: ['latin'],
  weight: ['400', '700'],
  variable: '--font-caveat',
});

const greatVibes = Great_Vibes({
  subsets: ['latin'],
  weight: ['400'],
  variable: '--font-great-vibes',
});
"""

content = content.replace(
    "const libreBaskerville =",
    font_defs + "\nconst libreBaskerville ="
)

# Add to body className
content = re.sub(
    r'\$\{libreFranklin.variable\}',
    r'${libreFranklin.variable} ${caveat.variable} ${greatVibes.variable}',
    content
)

with open('app/layout.tsx', 'w') as f:
    f.write(content)
