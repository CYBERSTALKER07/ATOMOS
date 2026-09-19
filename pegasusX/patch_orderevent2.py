import re

with open("apps/backend-go/events/types.go", "r") as f:
    content = f.read()

content = content.replace('\tOrderValidationErrs   []string `json:"order_validation_errs,omitempty"`\n}', '\tOrderValidationErrs   []string `json:"order_validation_errs,omitempty"`\n\tMessage               string   `json:"message,omitempty"`\n}')

with open("apps/backend-go/events/types.go", "w") as f:
    f.write(content)
