with open("the-lab-monorepo/apps/backend-go/vault/capabilities.go", "r") as f:
    text = f.read()

text = text.replace("func labelForField(gateway, field string)", "func labelForField(cap ProviderCapability, field string)")
text = text.replace("cap, ok := providerRegistry[gateway]\n        if !ok {\n                return field\n        }", "")
with open("the-lab-monorepo/apps/backend-go/vault/capabilities.go", "w") as f:
    f.write(text)
