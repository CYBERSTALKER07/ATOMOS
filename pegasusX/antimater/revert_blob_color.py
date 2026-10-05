with open('public/blob-anim.svg', 'r') as f:
    svg = f.read()

# I previously changed the blob from #f9f9f9 to #000000. Let's make it #ffffff.
svg = svg.replace('fill="#000000"', 'fill="#ffffff"')

with open('public/blob-anim.svg', 'w') as f:
    f.write(svg)
