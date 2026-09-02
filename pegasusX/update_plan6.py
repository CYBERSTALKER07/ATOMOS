import re

with open("/Users/shakhzod/.gemini/antigravity-cli/brain/859745ac-ab51-4564-8a5a-3a6c246aac63/backend_remediation_plan.md", "r") as f:
    content = f.read()

content = content.replace("- [ ] **Finding 5.13", "- [x] **Finding 5.13")
content = content.replace("- [ ] **Finding 5.14", "- [x] **Finding 5.14")

with open("/Users/shakhzod/.gemini/antigravity-cli/brain/859745ac-ab51-4564-8a5a-3a6c246aac63/backend_remediation_plan.md", "w") as f:
    f.write(content)
