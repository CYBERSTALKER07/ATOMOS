package order

import (
	"os"
	"strings"
)

// ReceiptPartyCopy labels which party is viewing the shared fiscal document.
type ReceiptPartyCopy string

const (
	PartyCopyPublic    ReceiptPartyCopy = "public"
	PartyCopyRetailer  ReceiptPartyCopy = "retailer"
	PartyCopySupplier  ReceiptPartyCopy = "supplier"
	PartyCopyWarehouse ReceiptPartyCopy = "warehouse"
)

// ReceiptLayout holds country-specific labels and legal footer copy for
// Pegasus platform settlement receipts (not Soliq/OFD tax documents).
type ReceiptLayout struct {
	CountryCode     string
	Locale          string
	Title           string
	Subtitle        string
	CompanyFallback string
	FooterNote      string
	OFDDeferredNote string
	Labels          ReceiptLabels
}

// ReceiptLabels are UI strings on the branded document.
type ReceiptLabels struct {
	ReceiptID     string
	OrderID       string
	IssuedAt      string
	PaymentMethod string
	Supplier      string
	Retailer      string
	SKU           string
	Item          string
	Qty           string
	UnitPrice     string
	LineTotal     string
	Total         string
	PartyCopy     string
	VerifyQR      string
	LegalClass    string
	DownloadPDF   string
}

func countryFromCurrency(currency string) string {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "KZT":
		return "KZ"
	default:
		return "UZ"
	}
}

func receiptLayoutForCountry(country string) ReceiptLayout {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "KZ":
		return ReceiptLayout{
			CountryCode:     "KZ",
			Locale:          "kk-KZ",
			Title:           "PegasusX settlement receipt",
			Subtitle:        "Official platform fiscal document",
			CompanyFallback: "PegasusX",
			FooterNote:      "This is an official PegasusX commercial settlement receipt. State OFD (Soliq) tax fiscalization is deferred until credentials are connected.",
			OFDDeferredNote: "Tax OFD not attached — platform receipt only.",
			Labels: ReceiptLabels{
				ReceiptID:     "Receipt ID",
				OrderID:       "Order",
				IssuedAt:      "Issued",
				PaymentMethod: "Payment",
				Supplier:      "Supplier",
				Retailer:      "Retailer",
				SKU:           "SKU",
				Item:          "Item",
				Qty:           "Qty",
				UnitPrice:     "Unit",
				LineTotal:     "Total",
				Total:         "Amount due",
				PartyCopy:     "Copy",
				VerifyQR:      "Verify",
				LegalClass:    "Legal class",
				DownloadPDF:   "Download PDF",
			},
		}
	default:
		return ReceiptLayout{
			CountryCode:     "UZ",
			Locale:          "uz-UZ",
			Title:           "PegasusX soliq uslubidagi kvitansiya",
			Subtitle:        "Rasmiy platforma fiskal hujjati / Official settlement receipt",
			CompanyFallback: "PegasusX",
			FooterNote:      "Bu PegasusX rasmiy tijorat hisob-faktura / settlement kvitansiyasi. Soliq/OFD soliq fiskalizatsiyasi keyinroq ulanadi — hozirda platforma hujjati.",
			OFDDeferredNote: "Soliq/OFD ulanmagan — faqat PegasusX platforma kvitansiyasi.",
			Labels: ReceiptLabels{
				ReceiptID:     "Kvitansiya ID / Receipt ID",
				OrderID:       "Buyurtma / Order",
				IssuedAt:      "Berilgan / Issued",
				PaymentMethod: "To‘lov / Payment",
				Supplier:      "Yetkazib beruvchi / Supplier",
				Retailer:      "Chakana / Retailer",
				SKU:           "SKU",
				Item:          "Mahsulot / Item",
				Qty:           "Soni / Qty",
				UnitPrice:     "Narx / Unit",
				LineTotal:     "Jami / Line",
				Total:         "Jami summa / Total",
				PartyCopy:     "Nusxa / Copy",
				VerifyQR:      "Tekshirish / Verify",
				LegalClass:    "Huquqiy tur / Legal class",
				DownloadPDF:   "PDF yuklash / Download PDF",
			},
		}
	}
}

func partyCopyLabel(copy ReceiptPartyCopy, _ ReceiptLayout) string {
	switch copy {
	case PartyCopyRetailer:
		return "Buyer / Retailer copy"
	case PartyCopySupplier:
		return "Supplier copy"
	case PartyCopyWarehouse:
		return "Warehouse / Fulfillment copy"
	default:
		return "Public verification copy"
	}
}

func pegasusIssuerTIN() string {
	if v := strings.TrimSpace(os.Getenv("PEGASUS_ISSUER_TIN")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_TIN")); v != "" {
		return v
	}
	return ""
}

func pegasusCompanyName() string {
	if v := strings.TrimSpace(os.Getenv("PEGASUS_COMPANY_NAME")); v != "" {
		return v
	}
	return "PegasusX"
}
