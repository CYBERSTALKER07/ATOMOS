package driver

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/grpc/codes"
)

type commandAckRequest struct {
	CommandID string `json:"command_id"`
	TraceID   string `json:"trace_id"`
	EventType string `json:"event_type"`
}

var (
	errCommandAckUnavailable = errors.New("command_ack_unavailable")
	errCommandAckInvalid     = errors.New("command_ack_invalid")
)

func (s *Service) recordCommandAck(ctx context.Context, driverID string, req commandAckRequest) error {
	commandID := strings.TrimSpace(req.CommandID)
	eventType := strings.TrimSpace(req.EventType)
	traceID := strings.TrimSpace(req.TraceID)
	if commandID == "" || len(commandID) > 128 || eventType == "" || len(eventType) > 64 || len(traceID) > 128 {
		return errCommandAckInvalid
	}
	if s == nil || s.spanner == nil {
		return errCommandAckUnavailable
	}
	_, err := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, readErr := txn.ReadRow(ctx, "DriverCommandAcks", spanner.Key{driverID, commandID}, []string{"CommandId"})
		if readErr == nil {
			// Already acknowledged by this driver; repeat command id must not emit a second event.
			return nil
		}
		if spanner.ErrCode(readErr) != codes.NotFound && !errors.Is(readErr, spanner.ErrRowNotFound) {
			return readErr
		}
		row := map[string]any{
			"DriverId":  driverID,
			"CommandId": commandID,
			"EventType": eventType,
			"AckedAt":   spanner.CommitTimestamp,
		}
		if traceID != "" {
			row["TraceId"] = traceID
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("DriverCommandAcks", row)}); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, "DriverCommandAck", commandID, events.TopicMain, map[string]any{
			"type":       "driver.command.acked",
			"driver_id":  driverID,
			"command_id": commandID,
			"trace_id":   traceID,
			"event_type": eventType,
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
	return err
}

func writeCommandAckError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errCommandAckUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "command_ack_unavailable"})
	case errors.Is(err, errCommandAckInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "command_id_and_event_type_required"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "command_ack_failed"})
	}
}
