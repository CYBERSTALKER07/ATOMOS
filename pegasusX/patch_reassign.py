import re

with open("apps/backend-go/order/reassign_handshake.go", "r") as f:
    content = f.read()

pattern = re.compile(r'if s\.driverHub != nil \{\n\t\tsiblings, err := s\.repo\.FindSiblingDriversForOrder\(r\.Context\(\), orderID\)\n\t\tif err == nil && len\(siblings\) > 1 \{\n\t\t\tfor _, sib := range siblings \{\n\t\t\t\tif sib != claims\.Subject \{\n\t\t\t\t\tpayload := map\[string\]any\{\n\t\t\t\t\t\t"type":     "REASSIGN_HANDSHAKE_COMPLETED",\n\t\t\t\t\t\t"order_id": orderID,\n\t\t\t\t\t\t"message":  "The other driver has started the reassigned order.",\n\t\t\t\t\t\}\n\t\t\t\t\tb, _ := json\.Marshal\(payload\)\n\t\t\t\t\tgo s\.driverHub\.Broadcast\(context\.Background\(\), "driver:"\+sib, b\)\n\t\t\t\t\}\n\t\t\t\}\n\t\t\}\n\t\}')

replacement = r"""siblings, err := s.repo.FindSiblingDriversForOrder(r.Context(), orderID)
	if err == nil && len(siblings) > 1 {
		_, _ = s.client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			buf := outbox.NewSpannerTxnBuffer(txn)
			for _, sib := range siblings {
				if sib != claims.Subject {
					outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.OrderEvent{
						BaseEvent: events.BaseEvent{Type: events.EventReassignHandshakeCompleted},
						OrderID:   orderID,
						DriverID:  sib,
						Message:   "The other driver has started the reassigned order.",
					})
				}
			}
			return buf.Flush(ctx)
		})
	}"""

content = pattern.sub(replacement, content)

# I need to make sure outbox, events, spanner imports are in reassign_handshake.go
content = content.replace('import (\n\t"context"\n\t"encoding/json"\n\t"net/http"\n\t"strings"\n\n\t"github.com/go-chi/chi/v5"\n\t"github.com/pegasusx/pegasusx/apps/backend-go/auth"\n)', 
'import (\n\t"context"\n\t"net/http"\n\t"strings"\n\n\t"cloud.google.com/go/spanner"\n\t"github.com/go-chi/chi/v5"\n\t"github.com/pegasusx/pegasusx/apps/backend-go/auth"\n\t"github.com/pegasusx/pegasusx/apps/backend-go/events"\n\t"github.com/pegasusx/pegasusx/apps/backend-go/outbox"\n)')

with open("apps/backend-go/order/reassign_handshake.go", "w") as f:
    f.write(content)
