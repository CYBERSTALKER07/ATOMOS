package migration

import (
	"context"
	"fmt"
	"strings"
)

func applyCustomerRow(ctx context.Context, tx ApplyTx, rctx *applyRowContext, rowIdx int64, fields map[string]string) error {
	name := strings.TrimSpace(fields["customer_name"])
	inn := NormalizeINN(fields["inn"])
	phone := NormalizePhoneUZ(fields["phone"])
	extID := strings.TrimSpace(fields["external_customer_id"])

	if name == "" || (inn == "" && phone == "") {
		return fmt.Errorf("%w: customer requires name and INN or phone at row %d", ErrInvalidInput, rowIdx)
	}
	if rawINN := strings.TrimSpace(fields["inn"]); rawINN != "" && len(inn) != 9 && len(inn) != 14 {
		return fmt.Errorf("%w: invalid INN %q at row %d", ErrInvalidInput, rawINN, rowIdx)
	}

	dedupKey := "cust|" + firstNonEmpty(inn, phone)
	if rctx.SeenKeys[dedupKey] {
		rctx.addWarning(rowIdx, "duplicate_customer_skipped", fmt.Sprintf("duplicate customer identity %s skipped", firstNonEmpty(inn, phone)))
		rctx.Result.Skipped++
		return nil
	}
	rctx.SeenKeys[dedupKey] = true

	if inn == "" && phone != "" {
		rctx.addWarning(rowIdx, "informal_shop_phone_only", fmt.Sprintf("customer %q linked via phone-only informal identifier", name))
	}

	mappedID := ""
	if extID != "" {
		if id, ok, err := tx.LookupIdentity(ctx, PackCustomers, extID); err != nil {
			return err
		} else if ok {
			mappedID = id
		}
	}

	existing, found, err := tx.FindCustomer(ctx, rctx.HomeCell, firstNonEmpty(mappedID, extID), inn, phone)
	if err != nil {
		return err
	}

	retailerID := existing.RetailerID
	createRetailer := false
	var beforeLink *StoredCustomerLink
	if found {
		cp := existing.Link
		if cp.SupplierID != "" {
			beforeLink = &cp
		}
	} else {
		createRetailer = true
		retailerID = deterministicEntityID("ret", rctx.Session.SupplierID, string(PackCustomers), firstNonEmpty(inn, phone, extID))
	}

	effectivePhone := phone
	if effectivePhone == "" {
		if existing.Phone != "" {
			effectivePhone = existing.Phone
		} else {
			// Retailers.Phone has a NOT NULL + UNIQUE constraint in Spanner; synthesize a non-dialable placeholder
			// when counterparty came only with TIN/INN so we never fabricate user login credentials.
			effectivePhone = "+000" + inn
		}
	}

	next := StoredCustomer{
		RetailerID:   retailerID,
		Phone:        effectivePhone,
		Name:         name,
		CountryCode:  "UZ",
		DeliveryAddr: strings.TrimSpace(fields["address"]),
		Link: StoredCustomerLink{
			SupplierID:      rctx.Session.SupplierID,
			RetailerID:      retailerID,
			TaxIdTIN:        firstNonEmpty(inn, existing.Link.TaxIdTIN),
			NormalizedPhone: phone,
			LegalName:       name,
			DedupStatus:     "CLEAN",
		},
	}

	if err := tx.SaveCustomer(ctx, next, createRetailer, beforeLink); err != nil {
		return err
	}
	if extID != "" {
		if err := tx.PutIdentity(ctx, PackCustomers, extID, retailerID); err != nil {
			return err
		}
	}
	if inn != "" {
		if err := tx.PutIdentity(ctx, PackCustomers, "inn:"+inn, retailerID); err != nil {
			return err
		}
	}
	if found {
		rctx.Result.Updated++
	} else {
		rctx.Result.Created++
	}
	return nil
}
