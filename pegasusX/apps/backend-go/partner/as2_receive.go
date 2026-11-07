package partner

import (
	"io"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/partner/as2"
)

const maxAS2Body = 8 << 20 // 8 MiB

// HandleAS2Receive POST /partner/v1/as2 — unauthenticated AS2 receive (RFC 4130).
func (h *Handlers) HandleAS2Receive(w http.ResponseWriter, r *http.Request) {
	if !PartnerAS2Enabled() && !PartnerAS2InsecurePlain() {
		writePartnerError(w, http.StatusServiceUnavailable, "as2_disabled")
		return
	}
	if h == nil || h.Svc == nil || h.EdiInbound == nil {
		writePartnerError(w, http.StatusServiceUnavailable, "as2_unavailable")
		return
	}

	hdrs := as2.ParseHeaders(r.Header)
	if hdrs.AS2To == "" || hdrs.AS2From == "" {
		http.Error(w, "missing AS2-From/AS2-To", http.StatusBadRequest)
		return
	}

	cfg, found, err := h.Svc.LookupAs2ByOurID(r.Context(), hdrs.AS2To)
	if err != nil || !found || !cfg.As2Enabled {
		_ = as2.WriteSyncMDN(w, hdrs.AS2To, hdrs.AS2From, hdrs, "", as2.MDNFailed, "unknown or disabled AS2-To")
		return
	}
	if as2.NormalizeAS2ID(cfg.PartnerAs2Id) != hdrs.AS2From {
		_ = as2.WriteSyncMDN(w, cfg.OurAs2Id, hdrs.AS2From, hdrs, "", as2.MDNFailed, "AS2-From mismatch")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxAS2Body))
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	_ = r.Body.Close()

	insecure := PartnerAS2InsecurePlain()

	var our as2.Material
	var partnerCertPEM []byte
	if !insecure {
		loader := LoadSecretRef
		certPEM, err1 := loader(cfg.OurCertSecretRef)
		keyPEM, err2 := loader(cfg.OurKeySecretRef)
		partnerPEM, err3 := loader(cfg.PartnerCertSecretRef)
		if err1 != nil || err2 != nil || err3 != nil {
			_ = as2.WriteSyncMDN(w, cfg.OurAs2Id, cfg.PartnerAs2Id, hdrs, "", as2.MDNFailed, "cert material unavailable")
			return
		}
		our, err = as2.LoadMaterial([]byte(certPEM), []byte(keyPEM))
		if err != nil {
			_ = as2.WriteSyncMDN(w, cfg.OurAs2Id, cfg.PartnerAs2Id, hdrs, "", as2.MDNFailed, "cert load failed")
			return
		}
		partnerCertPEM = []byte(partnerPEM)
	}

	payload, err := as2.UnwrapInbound(r.Header, body, our, partnerCertPEM, insecure)
	if err != nil {
		_ = as2.WriteSyncMDN(w, cfg.OurAs2Id, cfg.PartnerAs2Id, hdrs, "", as2.MDNFailed, err.Error())
		return
	}

	remoteName := payload.Filename
	if remoteName == "" {
		remoteName = "as2:" + hdrs.MessageID
	}
	if !looksLikeORDERS(string(payload.Content), remoteName) {
		_ = as2.WriteSyncMDN(w, cfg.OurAs2Id, cfg.PartnerAs2Id, hdrs, payload.MIC, as2.MDNFailed, "only ORDERS inbound supported")
		return
	}

	if err := h.EdiInbound.IngestORDERSBytes(r.Context(), cfg.TenantType, cfg.TenantID, remoteName, payload.Content); err != nil {
		_ = as2.WriteSyncMDN(w, cfg.OurAs2Id, cfg.PartnerAs2Id, hdrs, payload.MIC, as2.MDNFailed, err.Error())
		return
	}
	_ = as2.WriteSyncMDN(w, cfg.OurAs2Id, cfg.PartnerAs2Id, hdrs, payload.MIC, as2.MDNProcessed, "")
}

func looksLikeORDERS(body, filename string) bool {
	n := strings.ToUpper(strings.TrimSpace(filename))
	if strings.Contains(n, "ORDERS") {
		return true
	}
	u := strings.ToUpper(body)
	return strings.Contains(u, "UNH+") && strings.Contains(u, "ORDERS")
}
