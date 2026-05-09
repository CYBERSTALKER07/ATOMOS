with open('pegasus/apps/backend-go/kafka/billing_tier_worker.go', 'r') as f:
    content = f.read()

content = content.replace("\\\"", "\"")

with open('pegasus/apps/backend-go/kafka/billing_tier_worker.go', 'w') as f:
    f.write(content)
