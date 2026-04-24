import os
import re

def process_file(path, replacements):
    try:
        with open(path, "r") as f:
            text = f.read()
        for pattern, repl in replacements:
            text = re.sub(pattern, repl, text, flags=re.DOTALL | re.MULTILINE)
        with open(path, "w") as f:
            f.write(text)
    except Exception as e:
        print(f"Skipped {path}: {e}")

# payment/gateway_client.go
process_file(
    "the-lab-monorepo/apps/backend-go/payment/gateway_client.go",
    [
        (r'case "PAYME":.*?case "GLOBAL_PAY":', 'case "GLOBAL_PAY":'),
        (r'case "CLICK":.*?case "GLOBAL_PAY":', 'case "GLOBAL_PAY":'),
        (r'case "PAYME":.*?(?=return fmt\.Errorf)', ''),
        (r'case "CLICK":.*?(?=return fmt\.Errorf)', ''),
    ]
)
