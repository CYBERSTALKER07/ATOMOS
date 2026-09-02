import re

with open("apps/backend-go/payment/webhook_inbox.go", "r") as f:
    content = f.read()

pattern = re.compile(r'var recordJSON \[\]byte\n\t\tvar attempts int64\n\t\tif err := row\.Columns\(&webhookID, &gateway, &recordJSON, &source, &attempts\); err != nil \{\n\t\t\tcontinue\n\t\t\}\n\t\tvar record WebhookRecord\n\t\tif err := json\.Unmarshal\(recordJSON, &record\); err != nil \{')

replacement = r"""var recordJSON spanner.NullJSON
		var attempts int64
		if err := row.Columns(&webhookID, &gateway, &recordJSON, &source, &attempts); err != nil {
			continue
		}
		var record WebhookRecord
		
		recordBytes, marshalErr := json.Marshal(recordJSON.Value)
		if marshalErr != nil {
			_ = s.markDead(ctx, webhookID, attempts, marshalErr)
			continue
		}
		
		if err := json.Unmarshal(recordBytes, &record); err != nil {"""

content = pattern.sub(replacement, content)

with open("apps/backend-go/payment/webhook_inbox.go", "w") as f:
    f.write(content)
