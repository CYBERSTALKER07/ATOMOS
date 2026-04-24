import re

with open("the-lab-monorepo/apps/backend-go/payment/gateway_client.go", "r") as f:
    text = f.read()

# Remove struct PaymeClient and ClickClient
text = re.sub(r'(?m)^type (Payme|Click)Client struct {[\s\S]*?\n\}', '', text)

# Remove all their methods
text = re.sub(r'(?m)^func \(\w+ \*(Payme|Click)Client\) [\s\S]*?\n\}', '', text)

# Remove specific funcs
funcs_to_delete = [
    "NewPaymeClient", "paymeCheckoutURL", "paymeCheckoutURLWithCreds",
    "NewClickClient", "clickCheckoutURL", "clickCheckoutURLWithCreds"
]
for fn in funcs_to_delete:
    text = re.sub(r'(?m)^func ' + fn + r'\([\s\S]*?\n\}', '', text)

# Remove case "PAYME": ... case "CLICK": ... blocks inside switch statements
# A bit tricky, but we can match case "PAYME":\n\s+return.*\n
text = re.sub(r'(?m)^\s+case "PAYME":\n(?:.*?)\n', '', text)
text = re.sub(r'(?m)^\s+case "CLICK":\n(?:.*?)\n', '', text)

# also remove block headers
text = re.sub(r'// ─── CLICK ─+\n', '', text)
text = re.sub(r'// ─── PAYME ─+\n', '', text)

with open("the-lab-monorepo/apps/backend-go/payment/gateway_client.go", "w") as f:
    f.write(text)

