import re

with open('app/components/PegasusTestimonialsSection.tsx', 'r') as f:
    content = f.read()

# Fix the broken line
old_broken = 'className="relative z-10 text-[4rem] md:text-[5rem]" style={{ fontFamily: "var(--font-herr-von), cursive" }} tracking-tighter flex items-end leading-none"'
new_fixed = 'className="relative z-10 text-[4rem] md:text-[5rem] tracking-tighter flex items-end leading-none" style={{ fontFamily: "var(--font-herr-von), cursive" }}'

content = content.replace(old_broken, new_fixed)

with open('app/components/PegasusTestimonialsSection.tsx', 'w') as f:
    f.write(content)
