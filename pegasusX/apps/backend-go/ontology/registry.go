package ontology

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// AllObjectTypes returns the ordered list of all 17 canonical v1 ontology types.
func AllObjectTypes() []string {
	return []string{
		TypeSupplier,
		TypeLegalEntity,
		TypeWarehouse,
		TypeFactory,
		TypeSKU,
		TypeLot,
		TypeShop,
		TypeOrder,
		TypeOrderLine,
		TypeManifest,
		TypeVehicle,
		TypeDriver,
		TypeNasiyaInvoice,
		TypeCreditLine,
		TypePayment,
		TypeFiscalDoc,
		TypeProposal,
	}
}

// AllLinkTypes returns all 7 canonical v1 ontology link types.
func AllLinkTypes() []string {
	return []string{
		LinkSupplies,
		LinkStocks,
		LinkOrdered,
		LinkShippedOn,
		LinkOwes,
		LinkSubstitutes,
		LinkRecalledWith,
	}
}

// AllSecurityMarkings returns all canonical security markings recognized by the ontology layer.
func AllSecurityMarkings() []string {
	return []string{
		MarkingFinanceOnly,
		MarkingPII,
		MarkingAMLInvestigation,
		MarkingDiscrepantQuarantine,
		MarkingThermalQuarantineSpoiled,
		MarkingMicroSkimmingSuspect,
		MarkingLotRecallQuarantine,
	}
}

var typeDescriptors = []TypeDescriptor{
	{
		ObjectType:           TypeSupplier,
		Version:              "v1",
		SchemaFile:           "supplier.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"supplier_id", "name"},
		AllowedOutgoingLinks: []string{LinkSupplies, LinkStocks},
		Description:          "SSMR supplier tenant root entity",
	},
	{
		ObjectType:           TypeLegalEntity,
		Version:              "v1",
		SchemaFile:           "legal_entity.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"supplier_id", "legal_name"},
		MoneyFieldsMinor:     []string{"daily_cash_accumulated_minor", "daily_cash_ceiling_minor"},
		SecurityMarkings:     []string{MarkingFinanceOnly, MarkingPII},
		AllowedOutgoingLinks: []string{LinkSupplies, LinkOwes},
		Description:          "Registered legal entity with TIN/INN and banking settlement coordinates",
	},
	{
		ObjectType:           TypeWarehouse,
		Version:              "v1",
		SchemaFile:           "warehouse.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"warehouse_id", "name"},
		AllowedOutgoingLinks: []string{LinkStocks, LinkSupplies},
		Description:          "Supplier distribution depot or fulfillment warehouse",
	},
	{
		ObjectType:           TypeFactory,
		Version:              "v1",
		SchemaFile:           "factory.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"factory_id", "name"},
		AllowedOutgoingLinks: []string{LinkSupplies, LinkStocks},
		Description:          "Manufacturing or production facility feeding supplier warehouses",
	},
	{
		ObjectType:           TypeSKU,
		Version:              "v1",
		SchemaFile:           "sku.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"product_id", "name", "price_minor", "currency"},
		MoneyFieldsMinor:     []string{"price_minor"},
		AllowedOutgoingLinks: []string{LinkSubstitutes, LinkStocks},
		Description:          "Catalog SKU with integer minor-unit pricing and handling classification",
	},
	{
		ObjectType:           TypeLot,
		Version:              "v1",
		SchemaFile:           "lot.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"lot_id", "warehouse_id", "product_id", "status"},
		AllowedOutgoingLinks: []string{LinkStocks, LinkRecalledWith},
		Description:          "Warehouse FEFO stock lot with quarantine/recall lineage",
	},
	{
		ObjectType:           TypeShop,
		Version:              "v1",
		SchemaFile:           "shop.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"retailer_id", "name"},
		SecurityMarkings:     []string{MarkingPII},
		AllowedOutgoingLinks: []string{LinkOrdered, LinkOwes},
		Description:          "Retailer store outlet ordering from supplier depots",
	},
	{
		ObjectType:           TypeOrder,
		Version:              "v1",
		SchemaFile:           "order.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"order_id", "retailer_id", "status", "total_minor", "currency"},
		MoneyFieldsMinor:     []string{"total_minor", "original_total_minor", "delivery_fee_minor"},
		SecurityMarkings:     []string{MarkingFinanceOnly},
		AllowedOutgoingLinks: []string{LinkOrdered, LinkShippedOn, LinkOwes},
		Description:          "Retailer fulfillment order with integer minor-unit totals",
	},
	{
		ObjectType:           TypeOrderLine,
		Version:              "v1",
		SchemaFile:           "order_line.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"order_id", "sku", "quantity", "unit_price_minor"},
		MoneyFieldsMinor:     []string{"unit_price_minor", "line_total_minor", "vat_minor", "price_gap_minor"},
		SecurityMarkings:     []string{MarkingFinanceOnly},
		AllowedOutgoingLinks: []string{LinkOrdered, LinkSubstitutes, LinkRecalledWith},
		Description:          "Individual SKU line item within an order",
	},
	{
		ObjectType:           TypeManifest,
		Version:              "v1",
		SchemaFile:           "manifest.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"manifest_id", "state"},
		AllowedOutgoingLinks: []string{LinkShippedOn},
		Description:          "Truck loading & dispatch manifest carrying orders",
	},
	{
		ObjectType:           TypeVehicle,
		Version:              "v1",
		SchemaFile:           "vehicle.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"vehicle_id", "license_plate"},
		AllowedOutgoingLinks: []string{LinkShippedOn},
		Description:          "Fleet truck or van with DVIR safety status",
	},
	{
		ObjectType:           TypeDriver,
		Version:              "v1",
		SchemaFile:           "driver.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"driver_id", "name"},
		SecurityMarkings:     []string{MarkingPII},
		AllowedOutgoingLinks: []string{LinkShippedOn},
		Description:          "Fleet driver operating delivery manifests",
	},
	{
		ObjectType:           TypeNasiyaInvoice,
		Version:              "v1",
		SchemaFile:           "nasiya_invoice.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"invoice_id", "retailer_id", "status", "principal_minor", "balance_minor", "currency"},
		MoneyFieldsMinor:     []string{"principal_minor", "balance_minor", "paid_minor", "amount_minor"},
		SecurityMarkings:     []string{MarkingFinanceOnly},
		AllowedOutgoingLinks: []string{LinkOwes},
		Description:          "Trade-credit (Nasiya) AR open invoice",
	},
	{
		ObjectType:           TypeCreditLine,
		Version:              "v1",
		SchemaFile:           "credit_line.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"retailer_id", "status", "credit_limit_minor", "current_balance_minor"},
		MoneyFieldsMinor:     []string{"credit_limit_minor", "current_balance_minor", "reserved_minor", "available_credit_minor"},
		SecurityMarkings:     []string{MarkingFinanceOnly},
		AllowedOutgoingLinks: []string{LinkOwes, LinkSupplies},
		Description:          "Per-retailer trade credit line & exposure state",
	},
	{
		ObjectType:           TypePayment,
		Version:              "v1",
		SchemaFile:           "payment.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"payment_id", "status", "amount_minor", "currency"},
		MoneyFieldsMinor:     []string{"amount_minor", "cash_minor", "card_minor", "refunded_minor"},
		SecurityMarkings:     []string{MarkingFinanceOnly},
		AllowedOutgoingLinks: []string{LinkOwes, LinkOrdered},
		Description:          "Payment capture, leg, or settlement event",
	},
	{
		ObjectType:           TypeFiscalDoc,
		Version:              "v1",
		SchemaFile:           "fiscal_doc.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"order_id", "provider", "status", "amount_minor", "currency"},
		MoneyFieldsMinor:     []string{"amount_minor", "net_minor", "vat_minor", "gross_minor"},
		SecurityMarkings:     []string{MarkingFinanceOnly},
		AllowedOutgoingLinks: []string{LinkOrdered},
		Description:          "OFD / Soliq EHF fiscal receipt document",
	},
	{
		ObjectType:           TypeProposal,
		Version:              "v1",
		SchemaFile:           "proposal.v1.schema.json",
		PrimaryKey:           []string{"cell_id", "object_type", "object_id"},
		RequiredProperties:   []string{"proposal_id", "proposal_kind", "status"},
		MoneyFieldsMinor:     []string{"proposed_amount_minor", "proposed_price_minor"},
		AllowedOutgoingLinks: []string{LinkOrdered, LinkStocks, LinkSubstitutes},
		Description:          "Replenishment, negotiation, or auto-order shadow proposal",
	},
}

var descriptorByType = buildDescriptorIndex()
var validLinkTypes = buildLinkSet()

func buildDescriptorIndex() map[string]TypeDescriptor {
	m := make(map[string]TypeDescriptor, len(typeDescriptors))
	for _, d := range typeDescriptors {
		m[d.ObjectType] = d
	}
	return m
}

func buildLinkSet() map[string]struct{} {
	s := make(map[string]struct{}, 7)
	for _, l := range AllLinkTypes() {
		s[l] = struct{}{}
	}
	return s
}

// ListTypes returns the full v1 ontology type catalog.
func ListTypes() TypesCatalog {
	cp := make([]TypeDescriptor, len(typeDescriptors))
	for i, d := range typeDescriptors {
		cp[i] = d
		cp[i].PrimaryKey = slices.Clone(d.PrimaryKey)
		cp[i].RequiredProperties = slices.Clone(d.RequiredProperties)
		cp[i].MoneyFieldsMinor = slices.Clone(d.MoneyFieldsMinor)
		cp[i].SecurityMarkings = slices.Clone(d.SecurityMarkings)
		cp[i].AllowedOutgoingLinks = slices.Clone(d.AllowedOutgoingLinks)
	}
	return TypesCatalog{
		Version:          SchemaVersion,
		Types:            cp,
		LinkTypes:        AllLinkTypes(),
		SecurityMarkings: AllSecurityMarkings(),
	}
}

// LookupType returns the descriptor for a canonical v1 object type.
func LookupType(objectType string) (TypeDescriptor, bool) {
	d, ok := descriptorByType[strings.TrimSpace(objectType)]
	return d, ok
}

// IsValidLinkType reports whether linkType is one of the 7 canonical v1 link types.
func IsValidLinkType(linkType string) bool {
	_, ok := validLinkTypes[strings.TrimSpace(linkType)]
	return ok
}

// ValidateObject validates PK coordinates, v1 type registration, required properties,
// and strict integer minor-unit money guarantees (zero floats).
func ValidateObject(obj *OntologyObject) error {
	if obj == nil {
		return ErrInvalidIdentity
	}
	obj.CellID = NormalizeCellID(obj.CellID)
	obj.ObjectType = strings.TrimSpace(obj.ObjectType)
	obj.ObjectID = strings.TrimSpace(obj.ObjectID)
	obj.SupplierID = strings.TrimSpace(obj.SupplierID)
	if obj.CellID == "" || obj.ObjectType == "" || obj.ObjectID == "" {
		return ErrInvalidIdentity
	}
	desc, ok := LookupType(obj.ObjectType)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownObjectType, obj.ObjectType)
	}
	if err := obj.PrepareForSpanner(); err != nil {
		return err
	}
	for _, req := range desc.RequiredProperties {
		val, exists := obj.Properties[req]
		if !exists || val == nil {
			return fmt.Errorf("%w: %s missing required property %q", ErrSchemaValidation, obj.ObjectType, req)
		}
		if s, isStr := val.(string); isStr && strings.TrimSpace(s) == "" {
			return fmt.Errorf("%w: %s property %q must not be empty", ErrSchemaValidation, obj.ObjectType, req)
		}
	}

	moneySet := make(map[string]struct{}, len(desc.MoneyFieldsMinor))
	for _, mf := range desc.MoneyFieldsMinor {
		moneySet[mf] = struct{}{}
	}
	if err := validatePropertiesNoFloats(obj.Properties, moneySet); err != nil {
		return err
	}
	if len(desc.SecurityMarkings) > 0 {
		obj.SecurityMarkings = MergeMarkings(obj.SecurityMarkings, desc.SecurityMarkings)
	} else {
		obj.SecurityMarkings = NormalizeMarkings(obj.SecurityMarkings)
	}
	return obj.PrepareForSpanner()
}

func validatePropertiesNoFloats(props map[string]any, moneyFields map[string]struct{}) error {
	for k, v := range props {
		_, isDeclaredMoney := moneyFields[k]
		isMinorSuffix := strings.HasSuffix(k, "_minor")
		if isDeclaredMoney || isMinorSuffix {
			if err := ensureInt64MinorValue(k, v); err != nil {
				return err
			}
			continue
		}
		if err := ensureNoFloatsValue(k, v); err != nil {
			return err
		}
	}
	return nil
}

func ensureInt64MinorValue(field string, v any) error {
	if v == nil {
		return fmt.Errorf("%w: %s must be integer minor units", ErrSchemaValidation, field)
	}
	switch tv := v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32:
		return nil
	case uint64:
		return nil
	case json.Number:
		s := tv.String()
		if strings.ContainsAny(s, ".eE") {
			return fmt.Errorf("%w: field %s (%s)", ErrFloatForbidden, field, s)
		}
		if _, err := strconv.ParseInt(s, 10, 64); err != nil {
			return fmt.Errorf("%w: field %s (%s)", ErrSchemaValidation, field, s)
		}
		return nil
	case float32, float64:
		return fmt.Errorf("%w: field %s", ErrFloatForbidden, field)
	default:
		return fmt.Errorf("%w: field %s expected integer, got %T", ErrSchemaValidation, field, v)
	}
}

func ensureNoFloatsValue(field string, v any) error {
	switch tv := v.(type) {
	case float32, float64:
		return fmt.Errorf("%w: field %s", ErrFloatForbidden, field)
	case json.Number:
		if strings.ContainsAny(tv.String(), ".eE") {
			return fmt.Errorf("%w: field %s (%s)", ErrFloatForbidden, field, tv.String())
		}
	case map[string]any:
		for subK, subV := range tv {
			if err := ensureNoFloatsValue(field+"."+subK, subV); err != nil {
				return err
			}
		}
	case []any:
		for idx, item := range tv {
			if err := ensureNoFloatsValue(fmt.Sprintf("%s[%d]", field, idx), item); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateEdge validates an OntologyGraphEdge against canonical v1 link types and coordinates.
func ValidateEdge(edge *OntologyGraphEdge) error {
	if edge == nil {
		return ErrInvalidIdentity
	}
	edge.CellID = NormalizeCellID(edge.CellID)
	edge.SourceObjectType = strings.TrimSpace(edge.SourceObjectType)
	edge.SourceObjectID = strings.TrimSpace(edge.SourceObjectID)
	edge.LinkType = strings.TrimSpace(edge.LinkType)
	edge.TargetObjectType = strings.TrimSpace(edge.TargetObjectType)
	edge.TargetObjectID = strings.TrimSpace(edge.TargetObjectID)
	edge.SupplierID = strings.TrimSpace(edge.SupplierID)
	edge.CreatedByActionID = strings.TrimSpace(edge.CreatedByActionID)
	if edge.CellID == "" || edge.SourceObjectType == "" || edge.SourceObjectID == "" ||
		edge.LinkType == "" || edge.TargetObjectType == "" || edge.TargetObjectID == "" {
		return ErrInvalidIdentity
	}
	if _, ok := LookupType(edge.SourceObjectType); !ok {
		return fmt.Errorf("%w: source %q", ErrUnknownObjectType, edge.SourceObjectType)
	}
	if _, ok := LookupType(edge.TargetObjectType); !ok {
		return fmt.Errorf("%w: target %q", ErrUnknownObjectType, edge.TargetObjectType)
	}
	if !IsValidLinkType(edge.LinkType) {
		return fmt.Errorf("%w: %q", ErrUnknownLinkType, edge.LinkType)
	}
	if edge.CreatedByActionID == "" {
		edge.CreatedByActionID = "ontology-projector"
	}
	return edge.PrepareForSpanner()
}

var piiSensitiveFields = map[string]bool{
	"phone":             true,
	"normalized_phone":  true,
	"email":             true,
	"tax_id":            true,
	"normalized_tax_id": true,
	"tin":               true,
	"tax_id_tin":        true,
	"inn":               true,
	"address":           true,
	"billing_address":   true,
	"delivery_address":  true,
	"contact_name":      true,
	"account_holder":    true,
	"legal_name":        true,
}

var financeSensitiveFields = map[string]bool{
	"price_minor":                  true,
	"total_minor":                  true,
	"original_total_minor":         true,
	"delivery_fee_minor":           true,
	"unit_price_minor":             true,
	"line_total_minor":             true,
	"vat_minor":                    true,
	"net_minor":                    true,
	"gross_minor":                  true,
	"price_gap_minor":              true,
	"principal_minor":              true,
	"balance_minor":                true,
	"paid_minor":                   true,
	"amount_minor":                 true,
	"cash_minor":                   true,
	"card_minor":                   true,
	"refunded_minor":               true,
	"credit_limit_minor":           true,
	"current_balance_minor":        true,
	"reserved_minor":               true,
	"available_credit_minor":       true,
	"proposed_price_minor":         true,
	"proposed_amount_minor":        true,
	"daily_cash_accumulated_minor": true,
	"daily_cash_ceiling_minor":     true,
	"bank_name":                    true,
	"account_number":               true,
	"iban":                         true,
	"swift_bic":                    true,
	"fiscal_qr":                    true,
	"provider_reference":           true,
}

const RedactedPlaceholder = "[REDACTED]"

// CanViewSensitiveMarkings returns true for finance and admin roles allowed unmasked
// MARKING:FINANCE_ONLY and MARKING:PII fields.
func CanViewSensitiveMarkings(viewerRole string) bool {
	return auth.IsFinanceOrAdminRole(viewerRole)
}

// RedactObjectForViewer returns a cloned OntologyObject with sensitive properties masked
// when the object carries MARKING:FINANCE_ONLY or MARKING:PII and the viewer is not
// a finance or admin role.
func RedactObjectForViewer(obj *OntologyObject, viewerRole string) (*OntologyObject, bool) {
	if obj == nil {
		return nil, false
	}
	cp := obj.Clone()
	if cp.Properties == nil && strings.TrimSpace(cp.PropertiesJSON) != "" {
		if m, err := DecodeJSONMapNoFloats([]byte(cp.PropertiesJSON)); err == nil {
			cp.Properties = m
		}
	}
	if cp.Properties == nil {
		cp.Properties = map[string]any{}
	}
	if CanViewSensitiveMarkings(viewerRole) {
		return cp, false
	}

	hasFinance := slices.Contains(cp.SecurityMarkings, MarkingFinanceOnly)
	hasPII := slices.Contains(cp.SecurityMarkings, MarkingPII)
	if !hasFinance && !hasPII {
		return cp, false
	}

	redactedAny := false
	for k := range cp.Properties {
		if hasPII && (piiSensitiveFields[k] || (k == "name" && (cp.ObjectType == TypeDriver || cp.ObjectType == TypeLegalEntity))) {
			cp.Properties[k] = RedactedPlaceholder
			redactedAny = true
			continue
		}
		if hasFinance && (financeSensitiveFields[k] || strings.HasSuffix(k, "_minor")) {
			cp.Properties[k] = RedactedPlaceholder
			redactedAny = true
			continue
		}
	}
	if redactedAny {
		cp.Properties["_redacted"] = true
		if b, err := json.Marshal(cp.Properties); err == nil {
			cp.PropertiesJSON = string(b)
		}
	}
	return cp, redactedAny
}

// RedactObjectsForViewer applies RedactObjectForViewer across a slice and reports whether any redaction occurred.
func RedactObjectsForViewer(objs []*OntologyObject, viewerRole string) ([]*OntologyObject, bool, bool) {
	maskedAny := false
	canView := CanViewSensitiveMarkings(viewerRole)
	out := make([]*OntologyObject, 0, len(objs))
	for _, o := range objs {
		if o == nil {
			continue
		}
		redacted, didRedact := RedactObjectForViewer(o, viewerRole)
		if didRedact {
			maskedAny = true
		}
		out = append(out, redacted)
	}
	return out, !canView || maskedAny, !canView || maskedAny
}
