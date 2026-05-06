import re

with open('pegasus/apps/admin-portal/app/globals.css', 'r') as f:
    content = f.read()

# Replace the HeroUI monochrome block with the ElevenLabs theme
new_heroui = """
:root {
  /* ElevenLabs Theme Background & Surface */
  --background: #f5f5f5;
  --foreground: #0c0a09;
  --surface: #ffffff;
  --surface-foreground: #4e4e4e;
  --overlay: #fafafa;
  --overlay-foreground: #0c0a09;

  /* Accent — ink primary action */
  --accent: #292524;
  --accent-foreground: #ffffff;
  --accent-soft: #f0efed;
  --accent-soft-foreground: #0c0a09;

  /* Default */
  --default: #e7e5e4;
  --default-foreground: #4e4e4e;

  /* Muted, Border, Focus */
  --muted: #777169;
  --border: #e7e5e4;
  --focus: #292524;
  --link: #292524;
  --scrollbar: #d6d3d1;

  /* Form Fields */
  --field-background: #ffffff;
  --field-foreground: #0c0a09;
  --field-placeholder: #777169;
  --field-border: #d6d3d1;

  /* Semantic */
  --success: #16a34a;
  --success-foreground: #ffffff;
  --danger: #dc2626;
  --danger-foreground: #ffffff;
}

:root.dark {
  /* Dark hero / Canvas deep */
  --background: #0c0a09;
  --foreground: #ffffff;
  --surface: #1c1917;
  --surface-foreground: #a8a29e;
  --overlay: #1c1917;
  --overlay-foreground: #ffffff;

  --accent: #ffffff;
  --accent-foreground: #0c0a09;
  --accent-soft: #292524;
  --accent-soft-foreground: #ffffff;

  --default: #292524;
  --default-foreground: #a8a29e;

  --muted: #a8a29e;
  --border: #292524;
  --focus: #ffffff;
  --field-background: #0c0a09;
  --field-foreground: #ffffff;
  --field-border: #292524;
}
"""

content = re.sub(r':root \{[^\}]*--background: oklch[^\}]*\}', new_heroui.split(':root.dark')[0].strip() + '\n', content, count=1, flags=re.DOTALL)
content = re.sub(r':root\.dark \{[^\}]*--background: oklch[^\}]*\}', new_heroui.split(':root.dark')[1].strip() + '\n', content, count=1, flags=re.DOTALL)

with open('pegasus/apps/admin-portal/app/globals.css', 'w') as f:
    f.write(content)
