import re

with open('app/components/OurApproach.tsx', 'r') as f:
    content = f.read()

# Change right container bg and text
content = content.replace('className="flex-1 flex flex-col bg-[#e6e6e6] text-black relative"', 'className="flex-1 flex flex-col bg-black text-white relative border-l border-white/10"')

# Change grid
content = content.replace('bg-[linear-gradient(to_right,#0000000a_1px,transparent_1px),linear-gradient(to_bottom,#0000000a_1px,transparent_1px)]', 'bg-[linear-gradient(to_right,#ffffff0a_1px,transparent_1px),linear-gradient(to_bottom,#ffffff0a_1px,transparent_1px)]')

# Change borders
content = content.replace('border-black/10', 'border-white/10')

# Change SVG arrow
content = content.replace('fill-black', 'fill-white')

# Change text colors
content = content.replace('text-black/60', 'text-white/60')
content = content.replace('text-black/70', 'text-white/70')
content = content.replace('group-hover:text-black/85', 'group-hover:text-white/85')

# Change hover backgrounds
content = content.replace('hover:bg-black/[0.04]', 'hover:bg-white/[0.04]')

# Change lucide icons text color
content = content.replace('className="text-black transition-all', 'className="text-white transition-all')

with open('app/components/OurApproach.tsx', 'w') as f:
    f.write(content)
