import re
with open('the-lab-monorepo/apps/backend-go/vault/capabilities.go', 'r') as f:
    text = f.read()

# Replace that orphaned chunk manually
broken_chunk = """
                ManualHint: "Enter your GlobalPay merchant credentials from the GlobalPay Business cabinet.",
        },"""

text = text.replace(broken_chunk, '')

with open('the-lab-monorepo/apps/backend-go/vault/capabilities.go', 'w') as f:
    f.write(text)

