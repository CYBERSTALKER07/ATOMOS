import re

with open("apps/backend-go/kafka/notification_dispatcher.go", "r") as f:
    content = f.read()

content = content.replace('case events.EventOrderValidationFailed:\n', 'case events.EventOrderValidationFailed, events.EventReassignHandshakeCompleted:\n')

with open("apps/backend-go/kafka/notification_dispatcher.go", "w") as f:
    f.write(content)
