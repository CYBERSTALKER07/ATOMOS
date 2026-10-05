import re

with open('app/components/About.tsx', 'r') as f:
    content = f.read()

# Add CSS mask to the container holding the Dither
# The container is currently: <div className="relative h-[240px] sm:h-[320px] md:h-[400px] lg:h-[500px] overflow-hidden bg-black rounded-lg" style={{ width: '100%' }}>

target = """<div className="relative h-[240px] sm:h-[320px] md:h-[400px] lg:h-[500px] overflow-hidden bg-black rounded-lg" style={{ width: '100%' }}>"""
replacement = """<div className="relative h-[240px] sm:h-[320px] md:h-[400px] lg:h-[500px] overflow-hidden bg-black rounded-lg" style={{ 
   width: '100%',
   WebkitMaskImage: 'url(/pegasus-badge-mask.svg)',
   WebkitMaskSize: 'contain',
   WebkitMaskRepeat: 'no-repeat',
   WebkitMaskPosition: 'center',
   maskImage: 'url(/pegasus-badge-mask.svg)',
   maskSize: 'contain',
   maskRepeat: 'no-repeat',
   maskPosition: 'center'
}}>"""

content = content.replace(target, replacement)

with open('app/components/About.tsx', 'w') as f:
    f.write(content)
