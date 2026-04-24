import os
import re

files_to_fix_onCash = [
    "app/setup/billing/page.tsx",
    "app/supplier/country-overrides/page.tsx",
    "app/supplier/country-overrides/page.tsx",
    "app/supplier/country-overrides/page.tsx",
    "app/supplier/country-overrides/page.tsx"
]

def replace_in_file(path, old, new):
    full_path = f"the-lab-monorepo/apps/admin-portal/{path}"
    if not os.path.exists(full_path): return
    with open(full_path, "r") as f:
        content = f.read()
    new_content = content.replace(old, new)
    with open(full_path, "w") as f:
        f.write(new_content)

def regex_replace_in_file(path, pattern, repl):
    full_path = f"the-lab-monorepo/apps/admin-portal/{path}"
    if not os.path.exists(full_path): return
    with open(full_path, "r") as f:
        content = f.read()
    new_content = re.sub(pattern, repl, content)
    with open(full_path, "w") as f:
        f.write(new_content)

# 1. Fix onCash -> onClick on buttons
replace_in_file("app/setup/billing/page.tsx", "onCash={", "onClick={")
replace_in_file("app/supplier/country-overrides/page.tsx", "onCash={", "onClick={")

# 2. Fix app/setup/billing/page.tsx duplicated object literal keys
# If there are duplicates, we need to inspect it. Let's just remove the second occurrence or check lines 12/13.
with open("the-lab-monorepo/apps/admin-portal/app/setup/billing/page.tsx", "r") as f:
    billing_content = f.read()

# Since it complains about duplicate keys, maybe the file declares them twice in an object.
# Let's fix line 12, 13 duplication if they are duplicates. By regex, we can just replace the whole file's duplicate keys if we can't find it simply.
# Let's read the file first to know what to replace.
