import re

with open("apps/backend-go/warehouse/repository_spanner.go", "r") as f:
    content = f.read()

outbox_helper = """
func outboxMutations(eventsList []outbox.Event) []*spanner.Mutation {
	mutations := make([]*spanner.Mutation, 0, len(eventsList))
	for _, e := range eventsList {
		createdAt := e.CreatedAt.UTC()
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		row := map[string]any{
			"EventId":       e.EventID,
			"AggregateType": e.AggregateType,
			"AggregateId":   e.AggregateID,
			"TopicName":     e.TopicName,
			"Payload":       e.Payload,
			"CreatedAt":     createdAt,
		}
		mutations = append(mutations, spanner.Insert("OutboxEvents", row))
	}
	return mutations
}
"""

content = re.sub(
    r"(func \(b \*spannerTxnBuffer\) BufferAudit[^\}]+\})",
    r"\1" + "\n\n" + outbox_helper,
    content
)

content = content.replace(
"""		// apply outbox/audits
		for _, e := range buf.events {
			mutations = append(mutations, outbox.InsertEventMutation(e))
		}
		for _, a := range buf.audits {
			mutations = append(mutations, outbox.InsertAuditMutation(a))
		}""",
"""		mutations = append(mutations, outboxMutations(buf.events)...)
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}"""
)

with open("apps/backend-go/warehouse/repository_spanner.go", "w") as f:
    f.write(content)

