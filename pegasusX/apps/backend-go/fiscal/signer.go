package fiscal

import (
	"context"
	"encoding/json"
	"fmt"
)

// EDSSigner is the contract for generating PKCS7 cryptographic signatures
// over canonical JSON payloads (EHF).
type EDSSigner interface {
	// Sign returns the PKCS7 attached signature (or base64 signed payload) for the given canonical bytes.
	Sign(ctx context.Context, payload []byte) ([]byte, error)
}

// AttachSignature takes an arbitrary struct, marshals it deterministically,
// signs it using the provided EDSSigner, and returns a JSON payload containing
// both the original struct fields and a "signature" field.
func AttachSignature(ctx context.Context, signer EDSSigner, payload any) ([]byte, error) {
	// 1. Generate canonical JSON bytes.
	canonical, err := MarshalCanonical(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical: %w", err)
	}

	// 2. Cryptographically sign the canonical bytes.
	signature, err := signer.Sign(ctx, canonical)
	if err != nil {
		return nil, fmt.Errorf("sign payload: %w", err)
	}

	// 3. Attach the signature to the payload.
	// We unmarshal the canonical payload into a generic map so we can inject the signature field.
	var generic map[string]any
	if err := json.Unmarshal(canonical, &generic); err != nil {
		return nil, fmt.Errorf("unmarshal for attach: %w", err)
	}

	generic["signature"] = string(signature)

	// 4. Return the final signed JSON.
	// We don't strictly need this to be canonical, but it's safe to use standard Marshal.
	finalPayload, err := json.Marshal(generic)
	if err != nil {
		return nil, fmt.Errorf("marshal final: %w", err)
	}

	return finalPayload, nil
}
