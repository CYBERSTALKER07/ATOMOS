import re

with open('tailwind.config.ts', 'r') as f:
    content = f.read()

content = content.replace(
    "'libre-sans': ['var(--font-libre-franklin)', 'sans-serif'],",
    "'libre-sans': ['var(--font-libre-franklin)', 'sans-serif'],\n        caveat: ['var(--font-caveat)', 'cursive'],\n        signature: ['var(--font-great-vibes)', 'cursive'],"
)

with open('tailwind.config.ts', 'w') as f:
    f.write(content)
