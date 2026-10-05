import re

with open('app/components/PegasusTestimonialsSection.tsx', 'r') as f:
    content = f.read()

# I want to change the signature from "Shakhzod Soliyev" in Great Vibes to a bespoke Oprah-style Pegasus signature
old_sig = """						<div 
							className="text-white/40 font-signature text-[2.5rem] md:text-[4rem] transform -rotate-[6deg] select-none translate-y-4 md:translate-y-10 pr-4 md:pr-12 pointer-events-none"
							style={{ textShadow: '0 4px 24px rgba(255,255,255,0.1)' }}
						>
							Shakhzod Soliyev
						</div>"""

new_sig = """						<div 
							className="relative text-white/50 transform -rotate-[4deg] select-none translate-y-4 md:translate-y-6 pr-8 md:pr-16 pointer-events-none"
							style={{ textShadow: '0 4px 24px rgba(255,255,255,0.1)' }}
						>
							{/* Oprah-style swooping loop behind/around the word */}
							<svg className="absolute top-1/2 left-0 w-[140%] h-[180%] -translate-y-[45%] -translate-x-[15%] text-white/50 overflow-visible" viewBox="0 0 200 100" fill="none" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round">
								<path d="M 50,85 C 10,80 -10,30 30,15 C 80,-5 150,10 160,50 C 170,90 90,110 50,90 C 20,75 25,45 45,40" />
								{/* Tail of the 's' swooping down and back up */}
								<path d="M 165,65 C 170,110 180,120 195,100 C 210,80 200,45 190,50" />
							</svg>

							{/* The name itself using Herr Von Muellerhoff for extreme cursive style */}
							<span className="relative z-10 font-herr-von text-[4rem] md:text-[5rem] tracking-tighter flex items-end leading-none">
								<span className="text-[5rem] md:text-[6.5rem] -mr-3 md:-mr-4 -mb-2 md:-mb-3">P</span>
								<span className="pb-2">egasus</span>
							</span>
						</div>"""

content = content.replace(old_sig, new_sig)

with open('app/components/PegasusTestimonialsSection.tsx', 'w') as f:
    f.write(content)
