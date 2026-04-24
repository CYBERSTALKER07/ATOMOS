import sys
import re

with open("the-lab-monorepo/apps/backend-go/payment/webhooks.go", "r") as f:
    text = f.read()

# Rip out CLICK WEBHOOK
text = re.sub(r'// CLICK WEBHOOK[.\s\S]*?(?=\n// PAYME WEBHOOK)', '', text, flags=re.DOTALL)
# Rip out PAYME WEBHOOK
text = re.sub(r'// PAYME WEBHOOK[.\s\S]*?(?=\n// GLOBAL PAY WEBHOOK)', '', text, flags=re.DOTALL)

with open("the-lab-monorepo/apps/backend-go/payment/webhooks.go", "w") as f:
    f.write(text)

with open("the-lab-monorepo/apps/backend-go/payment/webhook_contract_test.go", "r") as f:
    text2 = f.read()

text2 = re.sub(r'// Click Webhook Contract Tests[.\s\S]*?(?=\n// Payme Webhook Contract Tests)', '', text2, flags=re.DOTALL)
text2 = re.sub(r'// Payme Webhook Contract Tests[.\s\S]*?(?=\n// Global Pay Webhook Contract Tests)', '', text2, flags=re.DOTALL)

with open("the-lab-monorepo/apps/backend-go/payment/webhook_contract_test.go", "w") as f:
    f.write(text2)

