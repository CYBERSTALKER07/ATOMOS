import re

with open("apps/backend-go/outbox/spanner_store.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(s \*SpannerStore\) Fetch\(ctx context\.Context, claimant string, limit int\) \(\[\]Event, error\) \{.*?return events, err\n\}', re.DOTALL)

replacement = r"""func (s *SpannerStore) Fetch(ctx context.Context, claimant string, limit int) ([]Event, error) {
	if limit <= 0 {
		return nil, nil
	}
	now := time.Now().UTC()
	leaseUntil := now.Add(2 * time.Minute)
	
	fetchLimit := limit * 4
	if fetchLimit < 100 {
		fetchLimit = 100
	}
	if fetchLimit > 500 {
		fetchLimit = 500
	}

	stmt := spanner.Statement{
		SQL: `
SELECT EventId, AggregateType, AggregateId, TopicName, Payload, CreatedAt, PublishedAt, SupplierId
FROM OutboxEvents@{FORCE_INDEX=Idx_OutboxEvents_Unpublished}
WHERE PublishedAt IS NULL
  AND (ClaimedUntil IS NULL OR ClaimedUntil < @now)
ORDER BY CreatedAt
LIMIT @limit`,
		Params: map[string]interface{}{
			"limit": int64(fetchLimit),
			"now":   now,
		},
	}
	
	iter := s.client.Single().Query(ctx, stmt)
	candidates := make([]Event, 0, fetchLimit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iter.Stop()
			return nil, fmt.Errorf("outbox spanner store: fetch: %w", err)
		}

		var (
			eventID       string
			aggregateType string
			aggregateID   string
			topicName     string
			payload       []byte
			createdAt     time.Time
			publishedAt   spanner.NullTime
			supplierID    spanner.NullString
		)
		if err := row.Columns(&eventID, &aggregateType, &aggregateID, &topicName, &payload, &createdAt, &publishedAt, &supplierID); err != nil {
			iter.Stop()
			return nil, fmt.Errorf("outbox spanner store: fetch scan: %w", err)
		}

		stored := ""
		if supplierID.Valid {
			stored = supplierID.StringVal
		}
		e := Event{
			EventID:       eventID,
			AggregateType: aggregateType,
			AggregateID:   aggregateID,
			TopicName:     topicName,
			Payload:       payload,
			CreatedAt:     createdAt.UTC(),
			SupplierID:    ResolveSupplierID(stored, payload),
		}
		if publishedAt.Valid {
			ts := publishedAt.Time.UTC()
			e.PublishedAt = &ts
		}
		candidates = append(candidates, e)
	}
	iter.Stop()

	if len(candidates) == 0 {
		return nil, nil
	}

	claimed := FairInterleave(candidates, limit)
	if len(claimed) == 0 {
		return nil, nil
	}
	
	var events []Event
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		events = nil // reset on retry
		ids := make([]string, 0, len(claimed))
		eventMap := make(map[string]Event)
		for _, e := range claimed {
			ids = append(ids, e.EventID)
			eventMap[e.EventID] = e
		}
		
		chkStmt := spanner.Statement{
			SQL: `SELECT EventId FROM OutboxEvents 
			      WHERE EventId IN UNNEST(@ids) 
			      AND PublishedAt IS NULL 
			      AND (ClaimedUntil IS NULL OR ClaimedUntil < @now)`,
			Params: map[string]any{"ids": ids, "now": now},
		}
		chkIter := txn.Query(ctx, chkStmt)
		var validIDs []string
		for {
			row, err := chkIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				chkIter.Stop()
				return err
			}
			var id string
			if err := row.Columns(&id); err != nil {
				chkIter.Stop()
				return err
			}
			validIDs = append(validIDs, id)
		}
		chkIter.Stop()
		
		mutations := make([]*spanner.Mutation, 0, len(validIDs))
		for _, id := range validIDs {
			mutations = append(mutations, spanner.UpdateMap("OutboxEvents", map[string]interface{}{
				"EventId":      id,
				"ClaimedBy":    claimant,
				"ClaimedUntil": leaseUntil,
			}))
			events = append(events, eventMap[id])
		}
		
		if len(mutations) > 0 {
			return txn.BufferWrite(mutations)
		}
		return nil
	})
	
	return events, err
}"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/outbox/spanner_store.go", "w") as f:
    f.write(content)
