import re
with open('the-lab-monorepo/apps/backend-go/vault/capabilities.go', 'r') as f:
    text = f.read()

# Replace that orphaned chunk
text = re.sub(r'                ManualHint: "Enter your GlobalPay merchant credentials from the GlobalPay Business cabinet.",\n        },\n', '', text)

with open('the-lab-monorepo/apps/backend-go/vault/capabilities.go', 'w') as f:
    f.write(text)

with open('the-lab-monorepo/apps/backend-go/analytics/revenue.go', 'r') as f:
    text2 = f.read()

# Fix the escaped bang in go
text2 = text2.replace('err \\\!= nil', 'err \!= nil')
with open('the-lab-monorepo/apps/backend-go/analytics/revenue.go', 'w') as f:
    f.write(text2)
