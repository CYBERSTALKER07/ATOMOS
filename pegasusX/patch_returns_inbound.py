import re

with open("apps/backend-go/returns/inbound.go", "r") as f:
    content = f.read()

# I want to add an EventCreditNoteRequested emission.
# After the EventReturnReceivedAtWarehouse event mutation, I'll add another.
pattern = re.compile(r'\t\t\tmutations = append\(mutations, spanner\.InsertOrUpdateMap\("OutboxEvents", outbox\.EventRowMap\(outbox\.Event\{\n\t\t\t\tEventID:\s+eventID,\n\t\t\t\tAggregateType:\s+events\.AggregateOrder,\n\t\t\t\tAggregateID:\s+orderID,\n\t\t\t\tTopicName:\s+events\.TopicMain,\n\t\t\t\tPayload:\s+payload,\n\t\t\t\tCreatedAt:\s+s\.now\(\)\.UTC\(\),\n\t\t\t\tSupplierID:\s+supplierID,\n\t\t\t\}\)\)\)')

replacement = r"""			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(outbox.Event{
				EventID:       eventID,
				AggregateType: events.AggregateOrder,
				AggregateID:   orderID,
				TopicName:     events.TopicMain,
				Payload:       payload,
				CreatedAt:     s.now().UTC(),
				SupplierID:    supplierID,
			})))

			// TRK4-009: Enqueue automatic credit note generation upon warehouse return confirmation
			cnPayload, _ := json.Marshal(map[string]any{
				"type":        "CREDIT_NOTE_REQUESTED",
				"order_id":    orderID,
				"return_id":   returnID,
				"supplier_id": supplierID,
				"quantity":    creditQty,
				"sku_id":      skuID,
				"reason":      reason,
				"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
			})
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(outbox.Event{
				EventID:       "cn:" + returnID,
				AggregateType: events.AggregateOrder,
				AggregateID:   orderID,
				TopicName:     events.TopicMain,
				Payload:       cnPayload,
				CreatedAt:     s.now().UTC(),
				SupplierID:    supplierID,
			})))"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/returns/inbound.go", "w") as f:
    f.write(content)

