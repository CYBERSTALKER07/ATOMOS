with open('the-lab-monorepo/apps/backend-go/vault/capabilities.go', 'r') as f:
    text = f.read()

text = text.replace('if \\!ok {', 'if \!ok {')

with open('the-lab-monorepo/apps/backend-go/vault/capabilities.go', 'w') as f:
    f.write(text)

with open('the-lab-monorepo/apps/backend-go/vault/vault.go', 'r') as f:
    v_text = f.read()
    
# Let's add labelForField back to capabilities.go since vault depends on it.
# Actually I'll append it to capabilities.go
text += """
func labelForField(gateway, field string) string {
        cap, ok := providerRegistry[gateway]
        if \!ok {
                return field
        }
        for _, mf := range cap.ManualFields {
                if mf.Name == field {
                        return mf.Label
                }
        }
        return field
}
"""

with open('the-lab-monorepo/apps/backend-go/vault/capabilities.go', 'w') as f:
    f.write(text)
