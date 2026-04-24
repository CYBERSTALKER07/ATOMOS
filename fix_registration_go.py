with open("the-lab-monorepo/apps/backend-go/supplier/registration.go", "r") as f:
    text = f.read()

text = text.replace('validGateways := map[string]bool{"GLOBAL_PAY": true, "CASH": true, "CARD": true, "BANK": true, "GLOBAL_PAY": true, "CASH": true}', 'validGateways := map[string]bool{"GLOBAL_PAY": true, "CASH": true, "CARD": true, "BANK": true}')
text = text.replace('invalid payment_gateway — must be GLOBAL_PAY, CASH, CARD, BANK, GLOBAL_PAY, or CASH', 'invalid payment_gateway — must be GLOBAL_PAY, CASH, CARD, or BANK')

with open("the-lab-monorepo/apps/backend-go/supplier/registration.go", "w") as f:
    f.write(text)
