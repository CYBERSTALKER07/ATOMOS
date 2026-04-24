import re

with open("the-lab-monorepo/apps/backend-go/order/service.go", "r") as f:
    text = f.read()

text = text.replace('case "CASH", "GLOBAL_PAY", "GLOBAL_PAY":', 'case "CASH", "GLOBAL_PAY":')

with open("the-lab-monorepo/apps/backend-go/order/service.go", "w") as f:
    f.write(text)
