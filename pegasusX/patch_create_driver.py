import re

with open("apps/backend-go/driver/crud_handlers.go", "r") as f:
    content = f.read()

# Make sure bcrypt is imported
if '"golang.org/x/crypto/bcrypt"' not in content:
    content = content.replace('import (', 'import (\n\t"golang.org/x/crypto/bcrypt"\n', 1)

pattern = re.compile(r'\tvar req Driver\n\tif err := json\.Unmarshal\(body, &req\); err != nil \{\n\t\tweb\.JSONError\(w, "invalid request body", http\.StatusBadRequest\)\n\t\treturn\n\t\}')

replacement = r"""	var reqPayload struct {
		Driver
		Pin string `json:"pin"`
	}
	if err := json.Unmarshal(body, &reqPayload); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req := reqPayload.Driver
	if reqPayload.Pin != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(reqPayload.Pin), bcrypt.DefaultCost)
		if err == nil {
			req.PinHash = string(hash)
		}
	}"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/driver/crud_handlers.go", "w") as f:
    f.write(content)

