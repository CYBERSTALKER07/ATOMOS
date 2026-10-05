import re

with open('app/components/PegasusTestimonialsSection.tsx', 'r') as f:
    content = f.read()

old_row3 = """					{/* Row 3: Read Case Study Link */}
					<div className="col-span-2 mt-[-16px]">
						<Link href="#" className="text-zinc-400 hover:text-white transition-colors text-base md:text-lg flex items-center gap-2">
							{language === 'ru' ? 'Читать кейс' : 'Read case study'} <span aria-hidden="true" className="font-light">&rarr;</span>
						</Link>
					</div>"""

new_row3 = """					{/* Row 3: Read Case Study Link & Signature */}
					<div className="col-span-2 mt-[-16px] flex justify-between items-end relative">
						<Link href="#" className="text-zinc-400 hover:text-white transition-colors text-base md:text-lg flex items-center gap-2 relative z-10">
							{language === 'ru' ? 'Читать кейс' : 'Read case study'} <span aria-hidden="true" className="font-light">&rarr;</span>
						</Link>
						<div 
							className="text-white/40 font-signature text-[2.5rem] md:text-[4rem] transform -rotate-[6deg] select-none translate-y-4 md:translate-y-10 pr-4 md:pr-12 pointer-events-none"
							style={{ textShadow: '0 4px 24px rgba(255,255,255,0.1)' }}
						>
							Shakhzod Soliyev
						</div>
					</div>"""

content = content.replace(old_row3, new_row3)

with open('app/components/PegasusTestimonialsSection.tsx', 'w') as f:
    f.write(content)
