import sys
with open("the-lab-monorepo/apps/backend-go/order/unified_checkout.go", "r") as f:
    text = f.read()

target = "return txn.BufferWrite(mutations)\n        })"

replacement = """if err := txn.BufferWrite(mutations); err \!= nil {
                        return err
                }

                now := time.Now().UTC()
                for _, plan := range processedPlans {
                        event := SupplierOrderCreatedEvent{
                                InvoiceID:     invoiceID,
                                OrderID:       plan.OrderID,
                                SupplierID:    plan.SupplierID,
                                RetailerID:    req.RetailerID,
                                WarehouseID:   plan.WarehouseID,
                                WarehouseName: plan.WarehouseName,
                                Total:         processedTotals[plan.OrderID],
                                Currency:      "UZS",
                                Items:         plan.Items,
                                Timestamp:     now,
                        }
                        if err := outbox.EmitJSON(txn, "Order", plan.OrderID, kafkaEvents.TopicMain, event, telemetry.TraceIDFromContext(ctx)); err \!= nil {
                                return fmt.Errorf("failed to emit SupplierOrderCreatedEvent: %w", err)
                        }
                }

                if err := outbox.EmitJSON(txn, "Invoice", invoiceID, kafkaEvents.TopicMain, struct {
                        InvoiceID  string    `json:"invoice_id"`
                        RetailerID string    `json:"retailer_id"`
                        Total      int64     `json:"total"`
                        Currency   string    `json:"currency"`
                        OrderCount int       `json:"order_count"`
                        Timestamp  time.Time `json:"timestamp"`
                }{
                        InvoiceID:  invoiceID,
                        RetailerID: req.RetailerID,
                        Total:      effectiveGrandTotal,
                        Currency:   "UZS",
                        OrderCount: len(processedPlans),
                        Timestamp:  now,
                }, telemetry.TraceIDFromContext(ctx)); err \!= nil {
                        return fmt.Errorf("failed to emit UnifiedCheckoutCompleted ev: %w", err)
                }

                return nil
        })"""

result = text.replace(target, replacement, 1)

with open("the-lab-monorepo/apps/backend-go/order/unified_checkout.go", "w") as f:
    f.write(result)
