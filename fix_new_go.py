with open("the-lab-monorepo/apps/backend-go/bootstrap/new.go", "r") as f:
    text = f.read()

text = text.replace("Cache:        redisCache,", "Cache:        appCache,")

with open("the-lab-monorepo/apps/backend-go/bootstrap/new.go", "w") as f:
    f.write(text)
