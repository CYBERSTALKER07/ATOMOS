import re

with open("the-lab-monorepo/apps/backend-go/payment/webhooks.go", "r") as f:
    text = f.read()

# Match standard Go top level declarations
decls_to_remove = [
    "clickWebhookRequest", "clickWebhookResponse", "HandleClickWebhook", "writeClickError",
    "paymeWebhookRequest", "paymeWebhookResponse", "paymeResult", "paymeError", 
    "paymeRPCRequest", "paymeRPCResponse", "HandlePaymeWebhook", "paymeCheckPerform",
    "paymeCreateTransaction", "paymePerformTransaction", "paymeCancelTransaction",
    "paymeCheckTransaction", "writePaymeResult", "writePaymeError", "notifyPaymeCancelled",
    "parseClickWebhookRequest", "parsePaymeWebhookRequest"
]

for decl in decls_to_remove:
    # Remove `type Name struct { ... }` or `func Name(...) { ... }` or `func (recv) Name(...) { ... }`
    # We'll use a very simple brace matching or regex since Go braces match nicely at top level.
    # Pattern: ^(func\s+(?:\(.*?\)\s+)?|type\s+)DECL_NAME[\s\S]*?\n\}
    pattern = r"^(?:func\s+(?:\([^\)]+\)\s+)?|type\s+)" + decl + r"[\s\S]*?\n\}"
    text = re.sub(pattern, "", text, flags=re.MULTILINE)

# also remove block headers
text = re.sub(r'// CLICK WEBHOOK\n// ═+\n', '', text)
text = re.sub(r'// PAYME WEBHOOK \(JSON-RPC\)\n// ═+\n', '', text)
text = re.sub(r'// ─── CLICK ─+\n', '', text)
text = re.sub(r'// ─── PAYME ─+\n', '', text)

with open("the-lab-monorepo/apps/backend-go/payment/webhooks.go", "w") as f:
    f.write(text)

