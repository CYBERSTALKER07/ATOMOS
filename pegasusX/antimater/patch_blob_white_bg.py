# 1. Modify the SVG
with open('public/blob-anim.svg', 'r') as f:
    svg = f.read()

# Change the blob from #ffffff to #000000
svg = svg.replace('fill="#ffffff"', 'fill="#000000"')

with open('public/blob-anim.svg', 'w') as f:
    f.write(svg)

# 2. Modify OurApproach.tsx to change left background from bg-black to bg-white
with open('app/components/OurApproach.tsx', 'r') as f:
    approach = f.read()

approach = approach.replace('className="flex-1 relative bg-black', 'className="flex-1 relative bg-white')

with open('app/components/OurApproach.tsx', 'w') as f:
    f.write(approach)
