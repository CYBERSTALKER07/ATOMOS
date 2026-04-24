with open("the-lab-monorepo/apps/backend-go/vault/capabilities.go", "r") as f:
    text = f.read()

text = text.replace("if \!ok", "if !ok")
text = text.replace("\\!", "!")
with open("the-lab-monorepo/apps/backend-go/vault/capabilities.go", "w") as f:
    f.write(text)
