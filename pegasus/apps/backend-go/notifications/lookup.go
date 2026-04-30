package notifications

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/spanner"
)

// LookupTelegramChatID retrieves the TelegramChatId for a user by role.
// Currently only Retailers have TelegramChatId in their table.
// Returns empty string if not found or not configured.
func LookupTelegramChatID(client *spanner.Client, userID, role string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var sql string
	switch role {
	case "RETAILER":
		sql = `SELECT TelegramChatId FROM Retailers WHERE RetailerId = @id LIMIT 1`
	default:
		// Drivers and Suppliers do not have TelegramChatId columns yet
		return ""
	}

	stmt := spanner.Statement{
		SQL:    sql,
		Params: map[string]interface{}{"id": userID},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return ""
	}

	var chatID spanner.NullString
	if err := row.Columns(&chatID); err != nil {
		log.Printf("[TELEGRAM LOOKUP] Column parse failed for %s/%s: %v", role, userID, err)
		return ""
	}

	return chatID.StringVal
}

// LookupRetailerIDsForOrders resolves RetailerIds for a batch of order IDs.
// Used by the notification dispatcher to find recipients for dispatch events.
func LookupRetailerIDsForOrders(client *spanner.Client, orderIDs []string) map[string]string {
	if len(orderIDs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stmt := spanner.Statement{
		SQL:    `SELECT OrderId, RetailerId FROM Orders WHERE OrderId IN UNNEST(@ids)`,
		Params: map[string]interface{}{"ids": orderIDs},
	}

	result := make(map[string]string)
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var orderID, retailerID string
		if err := row.Columns(&orderID, &retailerID); err != nil {
			continue
		}
		result[orderID] = retailerID
	}

	return result
}
