import re

with open('app/data/megaNavigation.ts', 'r') as f:
    content = f.read()

# Replace export const MEGA_NAV_FOOTER_LINKS = [ ... ]
# with export const getMegaNavFooterLinks = (t: any) => [ ... ]
content = re.sub(r'export const MEGA_NAV_FOOTER_LINKS = \[', r'export const getMegaNavFooterLinks = (t: any) => [', content)
content = re.sub(r'\] as const;', r'];', content)

# Replace export const MEGA_NAV_CATEGORIES: MegaNavCategory[] = [ ... ]
# with export const getMegaNavCategories = (t: any): MegaNavCategory[] => [ ... ]
content = re.sub(r'export const MEGA_NAV_CATEGORIES: MegaNavCategory\[\] = \[', r'export const getMegaNavCategories = (t: any): MegaNavCategory[] => [', content)

with open('app/data/megaNavigation.ts', 'w') as f:
    f.write(content)
