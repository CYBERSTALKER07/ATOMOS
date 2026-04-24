import re

with open("the-lab-monorepo/apps/backend-go/payment/refund.go", "r") as f:
    text = f.read()

pattern = r'return txn\.BufferWrite\(mutations\)\n\s*\}\)\n\n\s*if txErr \!= nil \{\n\s*return nil, fmt\.Errorf\("refund transaction failed: %w", txErr\)\n\s*\}\n\n\s*// 5\. Emit Kafka event for notification dispatcher\n\s*if rs\.kafkaWriter \!= nil \{\n\s*rs\.emitRefundEvent\(orderID, retailerID, supplierID, refundID, refundAmount, refundStatus\)\n\s*\}'

replacement = r'''if err := txn.BufferWrite(mutations); err \!= nil {
                        return err
                }

                // 5. Emit OUTBOX event atomically
                payload := map[string]interface{}{
                        "order_id":    orderID,
                        "retailer_id": retailerID,
                        "supplier_id": supplierID,
                        "refund_id":   refundID,
                        "amount":      refundAmount,
                        "status":      refundStatus,
                        "timestamp":   now.Format(time.RFC3339),
                }

                return outbox.EmitJSON(txn, "Refund", refundID, string("PAYMENT_REFUNDED"), "lab-logistics-events", payload, telemetry.TraceIDFromContext(ctx))
        })

        if txErr \!= nil {
                return nil, fmt.Errorf("refund transaction failed: %w", txErr)
        }

        // Cache invalidate
        // NOTE: we skip invalidating if txErr \!= nil handled above
        cache.Invalidate(ctx, cache.PrefixActiveOrders+retailerID, cache.SupplierProfile(supplierID))'''
        
text = re.sub(pattern, replacement, text, flags=re.DOTALL)

with open("the-lab-monorepo/apps/backend-go/payment/refund.go", "w") as f:
    f.write(text)

