package order

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"
)

//go:embed receipt_assets/logo.svg
var receiptLogoSVG []byte

//go:embed receipt_templates/receipt.html.tmpl
var receiptHTMLTemplateSrc string

var receiptHTMLTmpl = template.Must(template.New("pegasus_receipt").Funcs(template.FuncMap{
	"money": formatMinorMoney,
}).Parse(receiptHTMLTemplateSrc))

// ReceiptLine is one printable line on the branded document.
type ReceiptLine struct {
	SKU            string `json:"sku"`
	Name           string `json:"name"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	LineTotalMinor int64  `json:"line_total_minor"`
}

// ReceiptDocument is the full branded Pegasus settlement receipt model.
type ReceiptDocument struct {
	ReceiptID     string           `json:"receipt_id"`
	Provider      string           `json:"provider"`
	LegalClass    string           `json:"legal_class"`
	TaxOFD        bool             `json:"tax_ofd"`
	TaxOFDNote    string           `json:"tax_ofd_note,omitempty"`
	Status        string           `json:"status"`
	OrderID       string           `json:"order_id"`
	SupplierID    string           `json:"supplier_id,omitempty"`
	RetailerID    string           `json:"retailer_id,omitempty"`
	AmountMinor   int64            `json:"amount_minor"`
	Currency      string           `json:"currency"`
	PaymentMethod string           `json:"payment_method,omitempty"`
	QRURL         string           `json:"qr_url,omitempty"`
	HTMLURL       string           `json:"html_url,omitempty"`
	PDFURL        string           `json:"pdf_url,omitempty"`
	IssuedAt      time.Time        `json:"issued_at,omitempty"`
	CountryCode   string           `json:"country_code"`
	PartyCopy     ReceiptPartyCopy `json:"party_copy"`
	PartyLabel    string           `json:"party_label"`
	CompanyName   string           `json:"company_name"`
	IssuerTIN     string           `json:"issuer_tin,omitempty"`
	LogoDataURI   string           `json:"-"`
	LineItems     []ReceiptLine    `json:"line_items"`
	Layout        ReceiptLayout    `json:"-"`
	Title         string           `json:"title"`
	Subtitle      string           `json:"subtitle"`
	FooterNote    string           `json:"footer_note"`
}

type storedReceiptPayload struct {
	Provider      string `json:"provider"`
	LegalClass    string `json:"legal_class"`
	TaxOFD        bool   `json:"tax_ofd"`
	TaxOFDNote    string `json:"tax_ofd_note"`
	ReceiptID     string `json:"receipt_id"`
	OrderID       string `json:"order_id"`
	SupplierID    string `json:"supplier_id"`
	RetailerID    string `json:"retailer_id"`
	AmountMinor   int64  `json:"amount_minor"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method"`
	QRURL         string `json:"qr_url"`
	IssuedAt      string `json:"issued_at"`
	CountryCode   string `json:"country_code"`
	CompanyName   string `json:"company_name"`
	IssuerTIN     string `json:"issuer_tin"`
	LogoPath      string `json:"logo_path"`
	LineItems     []struct {
		SKU            string `json:"sku"`
		Name           string `json:"name"`
		Quantity       int64  `json:"quantity"`
		UnitPriceMinor int64  `json:"unit_price_minor"`
	} `json:"line_items"`
	Branding map[string]any `json:"branding"`
}

func formatMinorMoney(amountMinor int64, currency string) string {
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" {
		cur = "UZS"
	}
	// Integer minor units (tiyin/tiyn style) → major with 2 decimals.
	neg := amountMinor < 0
	if neg {
		amountMinor = -amountMinor
	}
	major := amountMinor / 100
	frac := amountMinor % 100
	s := fmt.Sprintf("%d.%02d %s", major, frac, cur)
	if neg {
		return "-" + s
	}
	return s
}

func logoDataURI() string {
	if len(receiptLogoSVG) == 0 {
		return ""
	}
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(receiptLogoSVG)
}

func publicReceiptBaseURL(qrURL string) string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	if base == "" && qrURL != "" {
		// Derive from stored QR: …/v1/platform/receipts/{id}
		if i := strings.Index(qrURL, "/v1/platform/receipts/"); i >= 0 {
			base = strings.TrimRight(qrURL[:i], "/")
		}
	}
	if base == "" {
		base = "https://api-ssmr.pegasusx.app"
	}
	return base
}

func receiptFormatURLs(receiptID, qrURL string) (htmlURL, pdfURL, canonicalQR string) {
	base := publicReceiptBaseURL(qrURL)
	path := base + "/v1/platform/receipts/" + strings.TrimSpace(receiptID)
	return path + "?format=html", path + "?format=pdf", path + "?format=html"
}

// BuildReceiptDocument assembles a branded document from a fiscal row (+ optional order lines).
func BuildReceiptDocument(fr FiscalReceiptRow, order *Order, party ReceiptPartyCopy) ReceiptDocument {
	if party == "" {
		party = PartyCopyPublic
	}
	var stored storedReceiptPayload
	if len(fr.ProviderPayload) > 0 {
		_ = json.Unmarshal(fr.ProviderPayload, &stored)
	}

	currency := strings.TrimSpace(fr.Currency)
	if currency == "" {
		currency = strings.TrimSpace(stored.Currency)
	}
	if currency == "" {
		currency = "UZS"
	}
	country := strings.TrimSpace(stored.CountryCode)
	if country == "" {
		country = countryFromCurrency(currency)
	}
	layout := receiptLayoutForCountry(country)

	company := strings.TrimSpace(stored.CompanyName)
	if company == "" {
		if b, ok := stored.Branding["company_name"].(string); ok {
			company = strings.TrimSpace(b)
		}
	}
	if company == "" {
		company = pegasusCompanyName()
	}
	tin := strings.TrimSpace(stored.IssuerTIN)
	if tin == "" {
		if b, ok := stored.Branding["issuer_tin"].(string); ok {
			tin = strings.TrimSpace(b)
		}
	}
	if tin == "" {
		tin = pegasusIssuerTIN()
	}

	receiptID := strings.TrimSpace(fr.FiscalReceiptID)
	if receiptID == "" {
		receiptID = strings.TrimSpace(stored.ReceiptID)
	}
	htmlURL, pdfURL, canonicalQR := receiptFormatURLs(receiptID, firstNonEmpty(fr.FiscalQR, stored.QRURL))

	issued := fr.CreatedAt.UTC()
	if stored.IssuedAt != "" {
		if t, err := time.Parse(time.RFC3339, stored.IssuedAt); err == nil {
			issued = t.UTC()
		}
	}

	legal := "platform_receipt"
	taxOFD := false
	taxNote := layout.OFDDeferredNote
	if strings.EqualFold(fr.Provider, FiscalProviderMySoliq) || stored.TaxOFD {
		legal = "tax_ofd_receipt"
		taxOFD = true
		taxNote = "Soliq/OFD tax fiscal receipt"
	}
	if stored.LegalClass != "" && taxOFD {
		legal = stored.LegalClass
	}
	if stored.TaxOFDNote != "" && !taxOFD {
		taxNote = stored.TaxOFDNote
	}

	lines := make([]ReceiptLine, 0, len(stored.LineItems))
	for _, li := range stored.LineItems {
		qty := li.Quantity
		if qty <= 0 {
			qty = 1
		}
		lines = append(lines, ReceiptLine{
			SKU:            li.SKU,
			Name:           firstNonEmpty(li.Name, li.SKU),
			Quantity:       qty,
			UnitPriceMinor: li.UnitPriceMinor,
			LineTotalMinor: qty * li.UnitPriceMinor,
		})
	}
	if len(lines) == 0 && order != nil {
		for _, li := range order.LineItems {
			qty := li.Quantity
			if qty <= 0 {
				qty = 1
			}
			lines = append(lines, ReceiptLine{
				SKU:            li.SKU,
				Name:           firstNonEmpty(li.Name, li.SKU),
				Quantity:       qty,
				UnitPriceMinor: li.UnitPrice,
				LineTotalMinor: qty * li.UnitPrice,
			})
		}
	}

	amount := fr.AmountMinor
	if amount == 0 {
		amount = stored.AmountMinor
	}

	return ReceiptDocument{
		ReceiptID:     receiptID,
		Provider:      firstNonEmpty(fr.Provider, stored.Provider, FiscalProviderPegasus),
		LegalClass:    legal,
		TaxOFD:        taxOFD,
		TaxOFDNote:    taxNote,
		Status:        fr.Status,
		OrderID:       firstNonEmpty(fr.OrderID, stored.OrderID),
		SupplierID:    firstNonEmpty(fr.SupplierID, stored.SupplierID),
		RetailerID:    firstNonEmpty(fr.RetailerID, stored.RetailerID),
		AmountMinor:   amount,
		Currency:      currency,
		PaymentMethod: firstNonEmpty(fr.PaymentMethod, stored.PaymentMethod),
		QRURL:         canonicalQR,
		HTMLURL:       htmlURL,
		PDFURL:        pdfURL,
		IssuedAt:      issued,
		CountryCode:   country,
		PartyCopy:     party,
		PartyLabel:    partyCopyLabel(party, layout),
		CompanyName:   company,
		IssuerTIN:     tin,
		LogoDataURI:   logoDataURI(),
		LineItems:     lines,
		Layout:        layout,
		Title:         layout.Title,
		Subtitle:      layout.Subtitle,
		FooterNote:    layout.FooterNote,
	}
}

// RenderReceiptHTML returns a full HTML page for the document.
func RenderReceiptHTML(doc ReceiptDocument) ([]byte, error) {
	var buf bytes.Buffer
	if err := receiptHTMLTmpl.Execute(&buf, doc); err != nil {
		return nil, fmt.Errorf("receipt html: %w", err)
	}
	return buf.Bytes(), nil
}

// RenderReceiptPDF returns a simple branded PDF for the document.
func RenderReceiptPDF(doc ReceiptDocument) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(doc.CompanyName+" receipt "+doc.ReceiptID, false)
	pdf.SetAuthor(doc.CompanyName, false)
	pdf.AddPage()
	pdf.SetMargins(16, 16, 16)

	pdf.SetFillColor(11, 31, 23)
	pdf.Rect(0, 0, 210, 28, "F")
	pdf.SetTextColor(232, 245, 233)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(16, 8)
	pdf.CellFormat(0, 10, doc.CompanyName, "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetX(16)
	pdf.SetTextColor(167, 243, 208)
	pdf.CellFormat(0, 5, doc.Subtitle, "", 1, "L", false, 0, "")

	pdf.Ln(10)
	pdf.SetTextColor(15, 23, 42)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, doc.Title, "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(0, 6, doc.PartyLabel+" · "+doc.CountryCode, "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.SetTextColor(15, 23, 42)
	pdf.SetFont("Helvetica", "", 10)
	writePDFKV(pdf, doc.Layout.Labels.ReceiptID, doc.ReceiptID)
	writePDFKV(pdf, doc.Layout.Labels.OrderID, doc.OrderID)
	writePDFKV(pdf, doc.Layout.Labels.IssuedAt, doc.IssuedAt.UTC().Format(time.RFC3339))
	writePDFKV(pdf, doc.Layout.Labels.PaymentMethod, firstNonEmpty(doc.PaymentMethod, "—"))
	writePDFKV(pdf, doc.Layout.Labels.Supplier, firstNonEmpty(doc.SupplierID, "—"))
	writePDFKV(pdf, doc.Layout.Labels.Retailer, firstNonEmpty(doc.RetailerID, "—"))
	writePDFKV(pdf, doc.Layout.Labels.LegalClass, doc.LegalClass)
	if doc.IssuerTIN != "" {
		writePDFKV(pdf, "TIN / STIR", doc.IssuerTIN)
	}
	pdf.Ln(4)

	// Table header
	pdf.SetFillColor(236, 253, 245)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(32, 7, doc.Layout.Labels.SKU, "1", 0, "L", true, 0, "")
	pdf.CellFormat(70, 7, doc.Layout.Labels.Item, "1", 0, "L", true, 0, "")
	pdf.CellFormat(18, 7, doc.Layout.Labels.Qty, "1", 0, "R", true, 0, "")
	pdf.CellFormat(30, 7, doc.Layout.Labels.UnitPrice, "1", 0, "R", true, 0, "")
	pdf.CellFormat(30, 7, doc.Layout.Labels.LineTotal, "1", 1, "R", true, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	if len(doc.LineItems) == 0 {
		pdf.CellFormat(180, 7, "—", "1", 1, "C", false, 0, "")
	}
	for _, li := range doc.LineItems {
		name := li.Name
		if len(name) > 40 {
			name = name[:37] + "..."
		}
		pdf.CellFormat(32, 7, truncatePDF(li.SKU, 14), "1", 0, "L", false, 0, "")
		pdf.CellFormat(70, 7, truncatePDF(name, 36), "1", 0, "L", false, 0, "")
		pdf.CellFormat(18, 7, fmt.Sprintf("%d", li.Quantity), "1", 0, "R", false, 0, "")
		pdf.CellFormat(30, 7, formatMinorMoney(li.UnitPriceMinor, doc.Currency), "1", 0, "R", false, 0, "")
		pdf.CellFormat(30, 7, formatMinorMoney(li.LineTotalMinor, doc.Currency), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, doc.Layout.Labels.Total+": "+formatMinorMoney(doc.AmountMinor, doc.Currency), "", 1, "R", false, 0, "")

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(71, 85, 105)
	pdf.MultiCell(0, 4, doc.FooterNote, "", "L", false)
	if !doc.TaxOFD {
		pdf.Ln(2)
		pdf.MultiCell(0, 4, doc.TaxOFDNote, "", "L", false)
	}
	pdf.Ln(4)
	pdf.SetTextColor(15, 23, 42)
	pdf.MultiCell(0, 4, doc.Layout.Labels.VerifyQR+": "+doc.QRURL, "", "L", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("receipt pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func writePDFKV(pdf *gofpdf.Fpdf, k, v string) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(45, 6, k, "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 6, v, "", 1, "L", false, 0, "")
}

func truncatePDF(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
