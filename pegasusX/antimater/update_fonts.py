import re

with open('app/layout.tsx', 'r') as f:
    content = f.read()

font_imports = """
import { Libre_Baskerville, Libre_Franklin } from 'next/font/google';

const libreBaskerville = Libre_Baskerville({
  subsets: ['latin'],
  weight: ['400', '700'],
  style: ['normal', 'italic'],
  variable: '--font-libre-baskerville',
});

const libreFranklin = Libre_Franklin({
  subsets: ['latin'],
  weight: ['300', '400', '500', '600', '700'],
  variable: '--font-libre-franklin',
});
"""

# add imports after the globals.css import
content = content.replace('import "./globals.css";', 'import "./globals.css";\n' + font_imports)

# add variables to body className
content = content.replace(
    'className="font-sans antialiased relative bg-[#F8FAFC] text-zinc-900 dark:bg-black dark:text-white transition-colors duration-200"',
    'className={`font-sans antialiased relative bg-[#F8FAFC] text-zinc-900 dark:bg-black dark:text-white transition-colors duration-200 ${libreBaskerville.variable} ${libreFranklin.variable}`}'
)

with open('app/layout.tsx', 'w') as f:
    f.write(content)
