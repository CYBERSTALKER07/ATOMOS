package fiscal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
)

// Explicit fail-closed errors returned when Soliq gateway or EDS credentials are absent.
var (
	ErrSoliqGatewayNotConfigured = errors.New("fiscal: soliq gateway client is not configured")
	ErrEDSSignerNotConfigured    = errors.New("fiscal: EDS signer is not configured")
	ErrInvalidFiscalDocument     = errors.New("fiscal: invalid fiscal document payload")
)

// UzbekistanStrategy implements FiscalStrategy for Uzbekistan (UZ) via Soliq EHF + EDS.
type UzbekistanStrategy struct {
	soliqClient soliq.SoliqClient
	signer      EDSSigner
	tin         string
	operator    string
}

// NewUzbekistanStrategy creates an unconfigured UzbekistanStrategy (fail-closed on submit/cancel
// until Soliq client and EDS signer are supplied via NewUzbekistanStrategyWithDeps).
func NewUzbekistanStrategy() *UzbekistanStrategy {
	return &UzbekistanStrategy{operator: "direct"}
}

// NewUzbekistanStrategyWithDeps creates a production-ready UzbekistanStrategy backed by SoliqClient and EDSSigner.
func NewUzbekistanStrategyWithDeps(client soliq.SoliqClient, signer EDSSigner, tin, operator string) *UzbekistanStrategy {
	op := strings.TrimSpace(operator)
	if op == "" {
		op = "direct"
	}
	return &UzbekistanStrategy{
		soliqClient: client,
		signer:      signer,
		tin:         strings.TrimSpace(tin),
		operator:    op,
	}
}

// FormatDocument builds canonical JSON for an Uzbekistan EHF submission envelope.
func (s *UzbekistanStrategy) FormatDocument(ctx context.Context, docID string) ([]byte, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	docID = strings.TrimSpace(docID)
	if docID == "" {
		return nil, fmt.Errorf("%w: doc_id is required", ErrInvalidFiscalDocument)
	}
	op := "direct"
	var tin string
	if s != nil {
		if strings.TrimSpace(s.operator) != "" {
			op = strings.TrimSpace(s.operator)
		}
		tin = strings.TrimSpace(s.tin)
	}
	envelope := map[string]any{
		"action":   "SUBMIT",
		"country":  "UZ",
		"doc_id":   docID,
		"operator": op,
	}
	if tin != "" {
		envelope["tin"] = tin
	}
	return MarshalCanonical(envelope)
}

// SubmitDocument cryptographically EDS-signs formattedDoc and submits it to Soliq.
func (s *UzbekistanStrategy) SubmitDocument(ctx context.Context, docID string, formattedDoc []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	docID = strings.TrimSpace(docID)
	if docID == "" {
		return fmt.Errorf("%w: doc_id is required", ErrInvalidFiscalDocument)
	}
	trimmed := bytes.TrimSpace(formattedDoc)
	if len(trimmed) == 0 {
		return fmt.Errorf("%w: empty document payload", ErrInvalidFiscalDocument)
	}
	if s == nil || s.soliqClient == nil {
		return ErrSoliqGatewayNotConfigured
	}
	if s.signer == nil {
		return ErrEDSSignerNotConfigured
	}

	var payload map[string]any
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil || len(payload) == 0 {
		return fmt.Errorf("%w: formattedDoc must be a non-empty JSON object", ErrInvalidFiscalDocument)
	}
	if _, ok := payload["doc_id"]; !ok {
		payload["doc_id"] = docID
	}
	if _, ok := payload["tin"]; !ok && strings.TrimSpace(s.tin) != "" {
		payload["tin"] = strings.TrimSpace(s.tin)
	}
	if _, ok := payload["operator"]; !ok && strings.TrimSpace(s.operator) != "" {
		payload["operator"] = strings.TrimSpace(s.operator)
	}

	signedPayload, err := AttachSignature(ctx, s.signer, payload)
	if err != nil {
		return fmt.Errorf("sign uzbekistan fiscal document: %w", err)
	}

	resp, err := s.soliqClient.Submit(ctx, signedPayload, "uz-submit:"+docID)
	if err != nil {
		return fmt.Errorf("soliq submit document %s: %w", docID, err)
	}
	if resp == nil || !resp.Success {
		code := ""
		msg := "empty response"
		if resp != nil {
			code = resp.ErrorCode
			msg = resp.ErrorMessage
		}
		return fmt.Errorf("soliq rejected document %s (code=%s): %s", docID, code, msg)
	}
	return nil
}

// CancelDocument signs and submits a cancellation request for docID to Soliq.
func (s *UzbekistanStrategy) CancelDocument(ctx context.Context, docID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	docID = strings.TrimSpace(docID)
	if docID == "" {
		return fmt.Errorf("%w: doc_id is required", ErrInvalidFiscalDocument)
	}
	if s == nil || s.soliqClient == nil {
		return ErrSoliqGatewayNotConfigured
	}
	if s.signer == nil {
		return ErrEDSSignerNotConfigured
	}

	cancelPayload := map[string]any{
		"action":   "CANCEL",
		"country":  "UZ",
		"doc_id":   docID,
		"operator": strings.TrimSpace(s.operator),
	}
	if strings.TrimSpace(s.tin) != "" {
		cancelPayload["tin"] = strings.TrimSpace(s.tin)
	}
	signedPayload, err := AttachSignature(ctx, s.signer, cancelPayload)
	if err != nil {
		return fmt.Errorf("sign uzbekistan fiscal cancellation: %w", err)
	}

	resp, err := s.soliqClient.Submit(ctx, signedPayload, "uz-cancel:"+docID)
	if err != nil {
		return fmt.Errorf("soliq cancel document %s: %w", docID, err)
	}
	if resp == nil || !resp.Success {
		code := ""
		msg := "empty response"
		if resp != nil {
			code = resp.ErrorCode
			msg = resp.ErrorMessage
		}
		return fmt.Errorf("soliq rejected cancellation %s (code=%s): %s", docID, code, msg)
	}
	return nil
}
