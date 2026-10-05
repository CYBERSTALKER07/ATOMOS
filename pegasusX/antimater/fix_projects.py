import re

with open('app/components/Projects.tsx', 'r') as f:
    content = f.read()

# Update scroll container
content = content.replace(
    'className="flex flex-row overflow-x-auto scroll-smooth overscroll-x-contain snap-x snap-proximity gap-8 pl-6 pr-12 md:pl-[100px] md:pr-24 w-full h-[700px] lg:h-[800px]"',
    'className="flex flex-row overflow-x-auto scroll-smooth overscroll-x-contain snap-x snap-proximity gap-4 md:gap-6 pl-6 pr-12 md:pl-[100px] md:pr-24 w-full h-[650px] md:h-[750px]"'
)

# Update card container
content = content.replace(
    'className="relative rounded-3xl overflow-hidden flex-shrink-0 snap-center group/card bg-zinc-900 \n										w-[90vw] md:w-[70vw] xl:w-[1100px] xl:max-w-[1100px]"',
    'className="relative rounded-[32px] overflow-hidden flex-shrink-0 snap-center group/card bg-zinc-900 \n										w-[90vw] md:w-[80vw] lg:w-[960px]"'
)

content = content.replace(
    'className="absolute inset-0 p-8 md:p-12 flex flex-col justify-between pointer-events-none"',
    'className="absolute inset-0 p-8 md:p-16 lg:p-20 flex flex-col justify-between pointer-events-none"'
)

content = content.replace(
    'className="text-white text-4xl md:text-5xl lg:text-[54px] font-medium leading-[1.1] max-w-2xl mb-6 tracking-tight"',
    'className="text-white text-4xl md:text-5xl lg:text-[56px] font-medium leading-[1.1] max-w-2xl mb-8 tracking-tight"'
)

content = content.replace(
    'className="text-white/90 text-lg md:text-xl max-w-xl leading-relaxed"',
    'className="text-white/90 text-lg md:text-[22px] max-w-2xl leading-[1.4]"'
)

content = content.replace(
    'className="flex justify-start pointer-events-auto"',
    'className="flex justify-center pointer-events-auto pb-4 md:pb-8"'
)

content = content.replace(
    'className="bg-white text-black px-8 py-3 text-[10px] font-bold tracking-widest uppercase hover:bg-zinc-200 transition-colors rounded-sm"',
    'className="bg-[#f0f0f0] text-black px-12 py-4 text-xs font-bold tracking-[0.15em] uppercase hover:bg-white transition-colors rounded-md"'
)

with open('app/components/Projects.tsx', 'w') as f:
    f.write(content)
