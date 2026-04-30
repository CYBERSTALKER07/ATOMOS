package vault

// OnboardingMode describes how a supplier can configure a gateway.
type OnboardingMode string

const (
	ManualOnly         OnboardingMode = "MANUAL_ONLY"
	RedirectPlusManual OnboardingMode = "REDIRECT_PLUS_MANUAL"
)

type ManualField struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	InputType   string `json:"input_type,omitempty"`
	HelperText  string `json:"helper_text,omitempty"`
}

// ProviderCapability describes what a gateway supports for supplier onboarding.
type ProviderCapability struct {
	Gateway        string         `json:"gateway"`
	DisplayName    string         `json:"display_name"`
	OnboardingMode OnboardingMode `json:"onboarding_mode"`
	RequiredFields []string       `json:"required_fields"`
	ManualFields   []ManualField  `json:"manual_fields,omitempty"`
	ConnectLabel   string         `json:"connect_label,omitempty"`
	ConnectHint    string         `json:"connect_hint,omitempty"`
	ManualHint     string         `json:"manual_hint"`
}

// providerRegistry is the single source of truth for gateway capabilities.
// These providers start as MANUAL_ONLY until official merchant-connect docs arrive.
var providerRegistry = map[string]ProviderCapability{
	"CASH": {
		Gateway:        "CASH",
		DisplayName:    "Cash",
		OnboardingMode: ManualOnly,
		RequiredFields: []string{"merchant_id", "service_id", "secret_key"},
		ManualFields: []ManualField{
			{Name: "merchant_id", Label: "Merchant ID", Placeholder: "e.g. 12345"},
			{Name: "service_id", Label: "Service ID", Placeholder: "e.g. 23456"},
			{Name: "secret_key", Label: "Secret Key", Placeholder: "Enter your Cash secret key", InputType: "password"},
		},
		ManualHint: "Enter your Cash merchant credentials from the Cash merchant dashboard.",
	},
	"GLOBAL_PAY": {
		Gateway:        "GLOBAL_PAY",
		DisplayName:    "Global Pay",
		OnboardingMode: ManualOnly,
		RequiredFields: []string{"merchant_id", "service_id", "secret_key"},
		ManualFields: []ManualField{
			{Name: "merchant_id", Label: "OAuth Username", Placeholder: "e.g. gp_checkout_user", HelperText: "Mapped to the stored merchant_id field for backward compatibility."},
			{Name: "service_id", Label: "Service ID", Placeholder: "e.g. 200123", HelperText: "Required to create hosted checkout service tokens."},
			{Name: "secret_key", Label: "OAuth Password", Placeholder: "Enter your Global Pay OAuth password", InputType: "password", HelperText: "Stored encrypted and used for Checkout Service authentication."},
		},
		ManualHint: "Enter the Global Pay Checkout Service OAuth username, OAuth password, and service ID issued for your supplier account.",
	},
}

// providerOrder enforces display order in the UI.
var providerOrder = []string{"CASH", "GLOBAL_PAY"}

// GetProviderCapabilities returns the capability metadata for all supported gateways.
func GetProviderCapabilities() []ProviderCapability {
	caps := make([]ProviderCapability, 0, len(providerRegistry))
	for _, gateway := range providerOrder {
		if c, ok := providerRegistry[gateway]; ok {
			caps = append(caps, c)
		}
	}
	return caps
}

// GetProviderCapability returns capability for a specific gateway, or nil.
func GetProviderCapability(gateway string) *ProviderCapability {
	cap, ok := providerRegistry[gateway]
	if !ok {
		return nil
	}
	return &cap
}

// SupportsRedirect returns true if the gateway has redirect onboarding enabled.
func SupportsRedirect(gateway string) bool {
	cap, ok := providerRegistry[gateway]
	return ok && cap.OnboardingMode == RedirectPlusManual
}

func labelForField(cap ProviderCapability, field string) string {

	for _, mf := range cap.ManualFields {
		if mf.Name == field {
			return mf.Label
		}
	}
	return field
}
