import re

with open("apps/backend-go/order/consumer.go", "r") as f:
    content = f.read()

content = content.replace(
    """"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafka"
)""",
    """"github.com/pegasusx/pegasusx/apps/backend-go/events"
	pegasuskafka "github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	kafka "github.com/segmentio/kafka-go"
)"""
)

content = content.replace(
    "envelope, err := kafka.ParseEnvelope(msg.Value)",
    "envelope, err := pegasuskafka.ParseEnvelope(msg.Value)"
)

with open("apps/backend-go/order/consumer.go", "w") as f:
    f.write(content)

with open("apps/backend-go/order/external_payment.go", "r") as f:
    ext_content = f.read()

ext_content = ext_content.replace(
    "case StatusCompleted, StatusCancelled, StatusRejected:",
    "case StatusCompleted, StatusCancelled:"
)

with open("apps/backend-go/order/external_payment.go", "w") as f:
    f.write(ext_content)
