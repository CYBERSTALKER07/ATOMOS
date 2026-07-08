import re

with open('app/(dashboard)/settings/page.tsx', 'r') as f:
    content = f.read()

# The patch script added:
#   useRetailerSessionReconcile(() => {
#     void mutateAutoOrder();
#   });

pattern = re.compile(r'\s*useRetailerSessionReconcile\(\(\) => \{\n\s*void mutateAutoOrder\(\);\n\s*\}\);\n', re.MULTILINE)

# We want to remove all matches EXCEPT the one inside SettingsPage which is just before return (
# Let's count them
matches = list(re.finditer(pattern, content))
print(f"Found {len(matches)} matches")

# We want to keep the one that appears before:
#   return (
#     <div
#       className="min-h-full p-6 md:p-8"

# Instead of regex, let's just do text replacement
text_to_remove = """
  useRetailerSessionReconcile(() => {
    void mutateAutoOrder();
  });
"""

# Let's read the original from a clean checkout
import subprocess
subprocess.run(['git', 'show', 'HEAD:apps/retailer-app-desktop/app/(dashboard)/settings/page.tsx'], stdout=open('clean.tsx', 'w'))
