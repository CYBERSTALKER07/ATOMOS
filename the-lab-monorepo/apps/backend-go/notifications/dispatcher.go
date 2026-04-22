package notifications

import (
	"context"
	"log"

	"cloud.google.com/go/spanner"
)

// Dispatcher sends notifications through a config-driven fallback chain.
// Chain order is read from CountryConfigs.NotificationFallbackOrder (e.g. ["FCM","TELEGRAM","SMS"]).
type Dispatcher struct {
	FCM      *FCMClient
	Telegram *TelegramClient
	SMS      SMSProvider // nil if no SMS provider configured
	Spanner  *spanner.Client
}

// NotifyPayload is the unified notification payload.
type NotifyPayload struct {
	Title   string
	Body    string
	Data    map[string]string // FCM data payload
	Channel string            // Filled by dispatcher after delivery
}

// NotifyUser sends a notification to a user via the configured fallback chain.
// fallbackOrder is e.g. ["FCM", "TELEGRAM", "SMS"].
func (d *Dispatcher) NotifyUser(ctx context.Context, userID, role string, payload NotifyPayload, fallbackOrder []string) error {
	if len(fallbackOrder) == 0 {
		fallbackOrder = []string{"FCM", "TELEGRAM"}
	}

	// Resolve user contact info from Spanner based on role
	phone, fcmToken, telegramChatId := d.resolveUserContacts(ctx, userID, role)

	for _, channel := range fallbackOrder {
		switch channel {
		case "FCM":
			if fcmToken == "" || d.FCM == nil {
				continue
			}
			data := map[string]string{"title": payload.Title, "body": payload.Body}
			for k, v := range payload.Data {
				data[k] = v
			}
			err := d.FCM.SendDataMessage(fcmToken, data)
			if err == nil {
				payload.Channel = "FCM"
				return nil
			}
			log.Printf("[Dispatcher] FCM failed for %s: %v, trying next", userID, err)

		case "TELEGRAM":
			if telegramChatId == "" || d.Telegram == nil {
				continue
			}
			msg := payload.Title + "\n" + payload.Body
			err := d.Telegram.SendMessage(telegramChatId, msg)
			if err == nil {
				payload.Channel = "TELEGRAM"
				return nil
			}
			log.Printf("[Dispatcher] Telegram failed for %s: %v, trying next", userID, err)

		case "SMS":
			if phone == "" || d.SMS == nil {
				continue
			}
			msg := payload.Title + ": " + payload.Body
			err := d.SMS.Send(phone, msg)
			if err == nil {
				payload.Channel = "SMS"
				return nil
			}
			log.Printf("[Dispatcher] SMS failed for %s: %v, trying next", userID, err)
		}
	}

	log.Printf("[Dispatcher] All channels exhausted for user %s (role=%s)", userID, role)
	return nil // Best-effort — don't fail the caller
}

// resolveUserContacts fetches phone, FCM token, and Telegram chat ID based on role.
func (d *Dispatcher) resolveUserContacts(ctx context.Context, userID, role string) (phone, fcmToken, telegramChatId string) {
	switch role {
	case "RETAILER":
		row, err := d.Spanner.Single().ReadRow(ctx, "Retailers",
			spanner.Key{userID}, []string{"Phone", "FcmToken", "TelegramChatId"})
		if err != nil {
			return
		}
		var p, f, t spanner.NullString
		_ = row.Columns(&p, &f, &t)
		phone = p.StringVal
		fcmToken = f.StringVal
		telegramChatId = t.StringVal

	case "DRIVER":
		row, err := d.Spanner.Single().ReadRow(ctx, "Drivers",
			spanner.Key{userID}, []string{"Phone"})
		if err != nil {
			return
		}
		var p spanner.NullString
		_ = row.Columns(&p)
		phone = p.StringVal
		// Driver FCM token from DeviceTokens table
		fcmToken = d.getDeviceToken(ctx, userID)

	case "SUPPLIER", "ADMIN":
		row, err := d.Spanner.Single().ReadRow(ctx, "SupplierUsers",
			spanner.Key{userID}, []string{"Phone"})
		if err != nil {
			return
		}
		var p spanner.NullString
		_ = row.Columns(&p)
		phone = p.StringVal
		fcmToken = d.getDeviceToken(ctx, userID)
	}
	return
}

func (d *Dispatcher) getDeviceToken(ctx context.Context, userID string) string {
	stmt := spanner.Statement{
		SQL:    "SELECT Token FROM DeviceTokens WHERE UserId = @uid ORDER BY CreatedAt DESC LIMIT 1",
		Params: map[string]interface{}{"uid": userID},
	}
	iter := d.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return ""
	}
	var token string
	if err := row.Columns(&token); err != nil {
		return ""
	}
	return token
}
