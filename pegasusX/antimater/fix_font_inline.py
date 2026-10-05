import re

with open('app/components/PegasusTestimonialsSection.tsx', 'r') as f:
    content = f.read()

# Replace font-herr-von with inline style
content = content.replace(
    'className="relative z-10 font-herr-von text-[4rem] md:text-[5rem]',
    'className="relative z-10 text-[4rem] md:text-[5rem]" style={{ fontFamily: "var(--font-herr-von), cursive" }}'
)

# And if I had any font-signature left, replace it too (though it shouldn't be there)
content = content.replace(
    'font-signature',
    ''
)

with open('app/components/PegasusTestimonialsSection.tsx', 'w') as f:
    f.write(content)
