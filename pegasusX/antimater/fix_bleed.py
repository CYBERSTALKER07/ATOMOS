import re

with open('app/components/Projects.tsx', 'r') as f:
    content = f.read()

content = content.replace(
    '<PageSection id="projects" className="py-24 bg-black">',
    '<PageSection id="projects" className="py-24 bg-black" bleed={true}>'
)

with open('app/components/Projects.tsx', 'w') as f:
    f.write(content)
