package cache

import (
	"fmt"
	"strings"
)

// SupplierScopedKey prefixes a supplier id with the cluster hash-tag convention.
func SupplierScopedKey(supplierID, suffix string) string {
	return fmt.Sprintf("{sup:%s}:%s", supplierID, strings.TrimPrefix(suffix, ":"))
}
