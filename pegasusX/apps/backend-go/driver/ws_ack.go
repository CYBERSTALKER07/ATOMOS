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
	errCommandAckUnavailable  = errors.New("command_ack_unavailable")
	errCommandAckInvalid      = errors.New("command_ack_invalid")
	errCommandNotIssued       = errors.New("command_not_issued")
	errCommandEventMismatch   = errors.New("command_event_type_mismatch")
)

func validateIssuedCommandMatch(issuedEventType, requestedEventType string) error {
	issued := strings.TrimSpace(issuedEventType)
	requested := strings.TrimSpace(requestedEventType)
	if issued == "" || requested == "" {
		return errCommandAckInvalid
	}
	if !strings.EqualFold(issued, requested) {
		return errCommandEventMismatch
	}
	return nil
}

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
		cmdRow, cmdErr := txn.ReadRow(ctx, "DriverCommands", spanner.Key{driverID, commandID}, []string{"EventType"})
		if spanner.ErrCode(cmdErr) == codes.NotFound || errors.Is(cmdErr, spanner.ErrRowNotFound) {
			return errCommandNotIssued
		}
		if cmdErr != nil {
			return cmdErr
		}
		var issuedEventType string
		if err := cmdRow.Columns(&issuedEventType); err != nil {
			return err
		}
		if err := validateIssuedCommandMatch(issuedEventType, eventType); err != nil {
			return err
		}

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
	case errors.Is(err, errCommandNotIssued):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "command_not_issued"})
	case errors.Is(err, errCommandEventMismatch):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "command_event_type_mismatch"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "command_ack_failed"})
	}
}
