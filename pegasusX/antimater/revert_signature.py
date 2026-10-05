import re

with open('app/components/PegasusTestimonialsSection.tsx', 'r') as f:
    content = f.read()

# Extract from "{/* Row 3" to "</div>" at the end of the col-span-2 block
pattern = re.compile(r'\{\/\* Row 3: Read Case Study Link & Signature \*\/.*?<\/div>\n\t\t\t\t\t<\/div>', re.DOTALL)

original_row_3 = """{/* Row 3: Read Case Study Link */}
					<div className="col-span-2 mt-[-16px]">
						<Link href="#" className="text-zinc-400 hover:text-white transition-colors text-base md:text-lg flex items-center gap-2">
							{language === 'ru' ? 'Читать кейс' : 'Read case study'} <span aria-hidden="true" className="font-light">&rarr;</span>
						</Link>
					</div>"""

content = re.sub(pattern, original_row_3, content)

with open('app/components/PegasusTestimonialsSection.tsx', 'w') as f:
    f.write(content)
