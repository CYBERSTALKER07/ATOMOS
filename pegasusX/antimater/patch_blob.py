import re

# 1. Modify the SVG
with open('public/blob-anim.svg', 'r') as f:
    svg = f.read()

# The blob itself is #f9f9f9. Let's change it to #000000
svg = svg.replace('fill="#f9f9f9"', 'fill="#000000"')

# The background rect is #0a0a0c. Let's make it transparent
svg = svg.replace('fill="#0a0a0c"', 'fill="transparent"')

with open('public/blob-anim.svg', 'w') as f:
    f.write(svg)

# 2. Modify OurApproach.tsx to change left background from bg-black to bg-white
with open('app/components/OurApproach.tsx', 'r') as f:
    approach = f.read()

approach = approach.replace('className="flex-1 relative bg-black', 'className="flex-1 relative bg-white')

with open('app/components/OurApproach.tsx', 'w') as f:
    f.write(approach)

