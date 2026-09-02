import re

with open("/Users/shakhzod/.gemini/antigravity-cli/brain/859745ac-ab51-4564-8a5a-3a6c246aac63/backend_remediation_plan.md", "r") as f:
    content = f.read()

content = content.replace("[ ] **TRK4-005**", "[x] **TRK4-005**")
content = content.replace("[ ] **TRK4-006**", "[x] **TRK4-006**")
content = content.replace("[ ] **TRK4-007**", "[x] **TRK4-007**")
content = content.replace("[ ] **TRK4-008**", "[x] **TRK4-008**")
content = content.replace("[ ] **TRK4-009**", "[x] **TRK4-009**")
content = content.replace("[ ] **TRK4-010**", "[x] **TRK4-010**")

with open("/Users/shakhzod/.gemini/antigravity-cli/brain/859745ac-ab51-4564-8a5a-3a6c246aac63/backend_remediation_plan.md", "w") as f:
    f.write(content)
