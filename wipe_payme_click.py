import sys
import re

def process_file(path, regexes):
    try:
        with open(path, "r") as f:
            text = f.read()
        for pattern in regexes:
            text = re.sub(pattern, "", text, flags=re.DOTALL | re.MULTILINE)
        with open(path, "w") as f:
            f.write(text)
    except Exception as e:
        print(f"Skipped {path}: {e}")

# In webhooks.go, we just rip out HandleClickWebhook, HandlePaymeWebhook, and their structs
# They start from HandleClickWebhook to HandleGlobalPayWebhook
process_file(
    "the-lab-monorepo/apps/backend-go/payment/webhooks.go",
    [
        re.compile(r'// \-\-\- CLICK .*?(?=\n// \-\-\- PAYME )'),
        re.compile(r'// \-\-\- PAYME .*?(?=\n// \-\-\- GLOBAL PAY )')
    ]
)

process_file(
    "the-lab-monorepo/apps/backend-go/payment/webhook_contract_test.go",
    [
        re.compile(r'// Click Webhook Contract Tests.*?(?=\n// Payme Webhook Contract Tests)'),
        re.compile(r'// Payme Webhook Contract Tests.*?(?=\n// Global Pay Webhook Contract Tests)')
    ]
)

