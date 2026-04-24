import re

with open("the-lab-monorepo/apps/backend-go/payment/webhooks.go", "r") as f:
    text = f.read()

# I will write a simple python script to insert outbox.EmitJSON into the transactions and remove the Kafka emit blocks.
