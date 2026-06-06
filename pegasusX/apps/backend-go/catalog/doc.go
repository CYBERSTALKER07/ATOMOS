// Package catalog owns the product and category domain. It provides Spanner-
// backed CRUD for the Products and ProductCategories tables and exposes HTTP
// handlers consumed by catalogroutes. It does NOT own inventory levels (see
// inventory package) or cart state (see retailer package).
package catalog
