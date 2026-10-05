import re

with open('app/components/CloudEcosystemBento.tsx', 'r') as f:
    content = f.read()

old_span = 'className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-none border border-white/10 bg-white/[0.04] text-[1.55rem] text-white transition-transform duration-300 group-hover:scale-110"'
new_span = 'className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-none border border-white/10 bg-white/[0.04] text-[1.55rem] text-white transition-transform duration-300 group-hover:animate-float-icon group-hover:bg-white/[0.1] group-hover:border-white/20 group-hover:text-emerald-300"'

content = content.replace(old_span, new_span)

with open('app/components/CloudEcosystemBento.tsx', 'w') as f:
    f.write(content)
