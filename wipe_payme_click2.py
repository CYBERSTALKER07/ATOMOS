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

process_file(
    "the-lab-monorepo/apps/backend-go/payment/webhooks.go",
    [
        r'// \-\-\- CLICK [.\s\S]*?(?=\n// \-\-\- PAYME )',
        r'// \-\-\- PAYME [.\s\S]*?(?=\n// \-\-\- GLOBAL PAY )'
    ]
)

process_file(
    "the-lab-monorepo/apps/backend-go/payment/webhook_contract_test.go",
    [
        r'// Click Webhook Contract Tests[.\s\S]*?(?=\n// Payme Webhook Contract Tests)',
        r'// Payme Webhook Contract Tests[.\s\S]*?(?=\n// Global Pay Webhook Contract Tests)'
    ]
)

