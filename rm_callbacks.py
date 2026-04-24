import re

with open("the-lab-monorepo/apps/backend-go/payment/webhooks.go", "r") as f:
    text = f.read()

# Instead of blindly removing from // CLICK WEBHOOK to // PAYME WEBHOOK,
# let's just use Python parser to locate the structs and funcs.

# Strip type clickWebhookRequest
text = re.sub(r'type clickWebhookRequest struct \{.*?\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'type clickWebhookResponse struct \{.*?\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func parseClickWebhookRequest.*?\{.*?\n\}\n', '', text, flags=re.DOTALL)
text = re.sub(r'func \(c \*clickWebhookRequest\) Validate.*?\{.*?\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func \(ws \*WebhookService\) HandleClickWebhook.*?\{.*?// \-\-\- PAYME', '// --- PAYME', text, flags=re.DOTALL)
text = re.sub(r'// ─── CLICK ────────────────────────────────────────────────────────────────────\n', '', text)
text = re.sub(r'// CLICK WEBHOOK\n', '', text)

# Payme
text = re.sub(r'type paymeWebhookRequest struct \{.*?\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'type paymeWebhookResponse struct \{.*?\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'type paymeResult struct \{.*?\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'type paymeError struct \{.*?\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func parsePaymeWebhookRequest.*?\{.*?\n\}\n', '', text, flags=re.DOTALL)
text = re.sub(r'func writePaymeResponse.*?\{.*?\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func writePaymeError.*?\{.*?\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func \(ws \*WebhookService\) HandlePaymeWebhook.*?\{.*?// \-\-\- GLOBAL PAY', '// --- GLOBAL PAY', text, flags=re.DOTALL)
text = re.sub(r'// ─── PAYME ────────────────────────────────────────────────────────────────────\n', '', text)
text = re.sub(r'// PAYME WEBHOOK\n', '', text)


with open("the-lab-monorepo/apps/backend-go/payment/webhooks.go", "w") as f:
    f.write(text)

