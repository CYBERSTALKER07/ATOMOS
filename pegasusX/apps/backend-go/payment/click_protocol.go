package payment

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Click SHOP API (Click → merchant Prepare/Complete) and Merchant API
// (merchant → Click invoice/status/reversal).
// Ledger amounts stay int64 minor (tiyin for UZS). Click SHOP `amount` is so'm
// at the HTTP edge and is converted immediately.
// Official: https://docs.click.uz/en/click-api/shop/
//
// Live checkout still uses catalogHonestyExecutorFor("CLICK") until wired.

const (
	clickActionPrepare  = "0"
	clickActionComplete = "1"

	clickErrOK           = 0
	clickErrSign         = -1
	clickErrAmount       = -2
	clickErrAction       = -3
	clickErrAlreadyPaid  = -4
	clickErrUserMissing  = -5
	clickErrTxMissing    = -6
	clickErrUpdateFailed = -7
	clickErrRequest      = -8
	clickErrCancelled    = -9

	clickUZSDecimals = 2
)

func clickShopSign(req clickWebhookRequest, secret string) string {
	secret = strings.TrimSpace(secret)
	payload := strings.TrimSpace(req.ClickTransID) +
		strings.TrimSpace(req.ServiceID) +
		secret +
		strings.TrimSpace(req.MerchantTransID)
	if clickNormalizedAction(req.Action) == clickActionComplete {
		payload += strings.TrimSpace(req.MerchantPrepareID)
	}
	payload += strings.TrimSpace(req.Amount) +
		strings.TrimSpace(req.Action) +
		strings.TrimSpace(req.SignTime)
	sum := md5.Sum([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func verifyClickSignature(req clickWebhookRequest, secret string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	provided := strings.TrimSpace(req.SignString)
	if provided == "" {
		return false
	}
	expected := clickShopSign(req, secret)
	bProvided := []byte(strings.ToLower(provided))
	bExpected := []byte(strings.ToLower(expected))
	if len(bProvided) != len(bExpected) {
		return false
	}
	return subtle.ConstantTimeCompare(bProvided, bExpected) == 1
}

func clickNormalizedAction(action string) string {
	return strings.TrimSpace(action)
}

func clickMerchantAuthHeader(merchantUserID, secret string, unixSeconds int64) string {
	ts := strconv.FormatInt(unixSeconds, 10)
	sum := sha1.Sum([]byte(ts + strings.TrimSpace(secret)))
	digest := hex.EncodeToString(sum[:])
	return strings.TrimSpace(merchantUserID) + ":" + digest + ":" + ts
}

func clickHostedPayURL(serviceID, merchantID, orderID, amountSom, returnURL string) string {
	q := url.Values{}
	q.Set("service_id", strings.TrimSpace(serviceID))
	q.Set("merchant_id", strings.TrimSpace(merchantID))
	q.Set("amount", strings.TrimSpace(amountSom))
	q.Set("transaction_param", strings.TrimSpace(orderID))
	if strings.TrimSpace(returnURL) != "" {
		q.Set("return_url", strings.TrimSpace(returnURL))
	}
	return "https://my.click.uz/services/pay?" + q.Encode()
}

func clickMerchantBaseURL() string {
	return "https://api.click.uz/v2/merchant"
}

// parseDecimalToMinor converts a decimal string (major units) to int64 minor
// without float64. "1000.00" with 2 decimals → 100000.
func parseDecimalToMinor(amount string, decimals int) (int64, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return 0, fmt.Errorf("empty amount")
	}
	if strings.ContainsAny(amount, "eE") {
		return 0, fmt.Errorf("scientific notation not allowed")
	}
	if strings.HasPrefix(amount, "-") {
		return 0, fmt.Errorf("negative amount")
	}
	amount = strings.TrimPrefix(amount, "+")
	whole, frac, cut := strings.Cut(amount, ".")
	if !cut {
		frac = ""
	}
	if whole == "" {
		whole = "0"
	}
	if decimals < 0 {
		return 0, fmt.Errorf("invalid decimals")
	}
	if len(frac) > decimals {
		return 0, fmt.Errorf("too many decimal places")
	}
	for len(frac) < decimals {
		frac += "0"
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %w", err)
	}
	var f int64
	if decimals > 0 {
		f, err = strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid amount fraction: %w", err)
		}
	}
	pow := int64(1)
	for i := 0; i < decimals; i++ {
		if pow > (1<<63-1)/10 {
			return 0, fmt.Errorf("overflow")
		}
		pow *= 10
	}
	if w > 0 && w > (1<<63-1-f)/pow {
		return 0, fmt.Errorf("overflow")
	}
	return w*pow + f, nil
}

func formatMinorAsDecimal(minor int64, decimals int) string {
	if decimals <= 0 {
		return strconv.FormatInt(minor, 10)
	}
	neg := minor < 0
	if neg {
		minor = -minor
	}
	pow := int64(1)
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	whole := minor / pow
	frac := minor % pow
	out := fmt.Sprintf("%d.%0*d", whole, decimals, frac)
	if neg {
		return "-" + out
	}
	return out
}

func clickSomToMinor(amount string) (int64, error) {
	return parseDecimalToMinor(amount, clickUZSDecimals)
}

func clickMinorToSom(amountMinor int64) string {
	return formatMinorAsDecimal(amountMinor, clickUZSDecimals)
}

func decodeClickShopBody(contentType string, body []byte) (clickWebhookRequest, error) {
	var req clickWebhookRequest
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "application/x-www-form-urlencoded") {
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return req, err
		}
		req.ClickTransID = strings.TrimSpace(values.Get("click_trans_id"))
		req.MerchantTransID = strings.TrimSpace(values.Get("merchant_trans_id"))
		req.MerchantPrepareID = strings.TrimSpace(values.Get("merchant_prepare_id"))
		req.ServiceID = strings.TrimSpace(values.Get("service_id"))
		req.Amount = strings.TrimSpace(values.Get("amount"))
		req.Action = strings.TrimSpace(values.Get("action"))
		req.SignTime = strings.TrimSpace(values.Get("sign_time"))
		req.SignString = strings.TrimSpace(values.Get("sign_string"))
		req.Error = strings.TrimSpace(values.Get("error"))
		req.ErrorNote = strings.TrimSpace(values.Get("error_note"))
		return req, nil
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return req, err
	}
	req.ClickTransID = jsonValueString(m["click_trans_id"])
	req.MerchantTransID = jsonValueString(m["merchant_trans_id"])
	req.MerchantPrepareID = jsonValueString(m["merchant_prepare_id"])
	req.ServiceID = jsonValueString(m["service_id"])
	req.Amount = jsonValueString(m["amount"])
	req.Action = jsonValueString(m["action"])
	req.SignTime = jsonValueString(m["sign_time"])
	req.SignString = jsonValueString(m["sign_string"])
	req.Error = jsonValueString(m["error"])
	req.ErrorNote = jsonValueString(m["error_note"])
	return req, nil
}

func clickPrepareID(session SessionRecord) string {
	if id := strings.TrimSpace(session.SessionID); id != "" {
		return id
	}
	return strings.TrimSpace(session.OrderID)
}
