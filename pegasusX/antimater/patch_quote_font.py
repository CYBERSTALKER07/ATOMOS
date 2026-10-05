import re

with open('app/components/PegasusTestimonialsSection.tsx', 'r') as f:
    content = f.read()

# Replace font-light with font-libre
content = content.replace('font-light text-white leading-[1.3] tracking-tight', 'font-libre text-white leading-[1.3] tracking-tight')

with open('app/components/PegasusTestimonialsSection.tsx', 'w') as f:
    f.write(content)
