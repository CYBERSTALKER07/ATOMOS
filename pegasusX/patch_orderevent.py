import re

with open("apps/backend-go/events/types.go", "r") as f:
    content = f.read()

content = content.replace('Currency              string  `json:"currency,omitempty"`\n}', 'Currency              string  `json:"currency,omitempty"`\n\tMessage               string  `json:"message,omitempty"`\n}')

with open("apps/backend-go/events/types.go", "w") as f:
    f.write(content)
