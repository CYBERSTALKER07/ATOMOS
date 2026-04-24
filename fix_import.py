import re
with open("the-lab-monorepo/apps/backend-go/payment/webhooks.go", "r") as f:
    t = f.read()

t = t.replace('"backend-go/outbox"\n\twsEvents "backend-go/ws"', '"backend-go/outbox"\n\t"backend-go/workers"\n\twsEvents "backend-go/ws"')

with open("the-lab-monorepo/apps/backend-go/payment/webhooks.go", "w") as f:
    f.write(t)
