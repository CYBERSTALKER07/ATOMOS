with open("the-lab-monorepo/apps/backend-go/bootstrap/new.go", "r") as f:
    text = f.read()

text = text.replace("Cache:        appCache,", "Cache:        c,")

with open("the-lab-monorepo/apps/backend-go/bootstrap/new.go", "w") as f:
    f.write(text)
