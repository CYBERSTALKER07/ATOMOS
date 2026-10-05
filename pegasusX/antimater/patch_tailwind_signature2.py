import re

with open('tailwind.config.ts', 'r') as f:
    content = f.read()

content = content.replace(
    "signature: ['var(--font-great-vibes)', 'cursive'],",
    "signature: ['var(--font-great-vibes)', 'cursive'],\n        'herr-von': ['var(--font-herr-von)', 'cursive'],\n        'mrs-saint': ['var(--font-mrs-saint)', 'cursive'],"
)

with open('tailwind.config.ts', 'w') as f:
    f.write(content)
