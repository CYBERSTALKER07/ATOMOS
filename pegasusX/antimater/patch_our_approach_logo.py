import re

with open('app/components/OurApproach.tsx', 'r') as f:
    content = f.read()

# Replace Digit369 import and usage
content = content.replace("import Digit369 from './Digit369';\n", "")

old_code = """<div className="absolute inset-0">
 <Digit369 color="#e8e4e3" backgroundColor="#000000" />
 </div>"""

new_code = """<div className="absolute inset-0 flex items-center justify-center p-8 lg:p-12">
 <img src="/blob-anim.svg" alt="Animated Logo" className="w-full h-full object-contain" />
 </div>"""

content = content.replace(old_code, new_code)

with open('app/components/OurApproach.tsx', 'w') as f:
    f.write(content)
