cat pegasusX/apps/backend-go/order/repository_spanner.go | sed '/func (r \*SpannerRepository) GetOrder/i\
\/\/ UpdateOrder overrides an order state and emits outbox events automatically.\\
func (r *SpannerRepository) UpdateOrder(ctx context.Context, o Order, emit func(outbox.TxnBuffer) error) error {\\
	if r == nil || r.client == nil {\\
		return fmt.Errorf("spanner order repository: nil client")\\
	}\\
\\
	lineItemsRaw, err := json.Marshal(o.LineItems)\\
	if err \!= nil {\\
		return fmt.Errorf("marshal order line items: %w", err)\\
	}\\
\\
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {\\
		// Check for Version match to ensure optimistic concurrency\\
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{o.OrderID}, []string{"Version"})\\
		if err \!= nil {\\
			return err\\
		}\\
		var version int64\\
		if err := row.Columns(&version); err \!= nil {\\
			return err\\
		}\\
		if version \!= o.Version {\\
			return fmt.Errorf("optimistic concurrency conflict: expected %d, got %d", o.Version, version)\\
		}\\
\\
		o.Version++\\
		o.UpdatedAt = time.Now().UTC()\\
\\
		buf := &spannerTxnBuffer{}\\
		if emit \!= nil {\\
			if err := emit(buf); err \!= nil {\\
				return err\\
			}\\
		}\\
\\
		mutations := []*spanner.Mutation{\\
			spanner.UpdateMap("Orders", map[string]any{\\
				"OrderId":       o.OrderID,\\
				"Status":        string(o.Status),\\
				"LineItemsJson": lineItemsRaw,\\
				"TotalMinor":    o.TotalMinor,\\
				"Currency":      o.Currency,\\
				"H3Cell":        o.H3Cell,\\
				"Lat":           o.Lat,\\
				"Lng":           o.Lng,\\
				"Version":       o.Version,\\
				"UpdatedAt":     o.UpdatedAt,\\
			}),\\
		}\\
\\
		for _, e := range buf.events {\\
			createdAt := e.CreatedAt.UTC()\\
			if createdAt.IsZero() {\\
				createdAt = time.Now().UTC()\\
			}\\
\\
			row := map[string]any{\\
				"EventId":       e.EventID,\\
				"AggregateType": e.AggregateType,\\
				"AggregateId":   e.AggregateID,\\
				"TopicName":     e.TopicName,\\
				"Payload":       e.Payload,\\
				"CreatedAt":     createdAt,\\
				"PublishedAt":   nil,\\
			}\\
			if e.PublishedAt \!= nil {\\
				row["PublishedAt"] = e.PublishedAt.UTC()\\
			}\\
\\
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))\\
		}\\
\\
		return txn.BufferWrite(mutations)\\
	})\\
	if err \!= nil {\\
		return fmt.Errorf("update order transaction: %w", err)\\
	}\\
\\
	return nil\\
}\\
' > new_repo.go
mv new_repo.go pegasusX/apps/backend-go/order/repository_spanner.go
