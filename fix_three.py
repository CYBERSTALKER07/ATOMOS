with open("the-lab-monorepo/apps/backend-go/analytics/revenue.go", "r") as f:
    text = f.read()

text = text.replace("err \!= nil", "err != nil")

with open("the-lab-monorepo/apps/backend-go/analytics/revenue.go", "w") as f:
    f.write(text)
