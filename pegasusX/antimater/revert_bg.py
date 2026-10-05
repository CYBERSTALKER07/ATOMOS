import re

with open('app/components/OurApproach.tsx', 'r') as f:
    approach = f.read()

approach = approach.replace('className="flex-1 relative bg-white min-h-[280px]', 'className="flex-1 relative bg-black min-h-[280px]')

with open('app/components/OurApproach.tsx', 'w') as f:
    f.write(approach)

