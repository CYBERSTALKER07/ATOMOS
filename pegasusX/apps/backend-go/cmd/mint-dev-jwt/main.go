package main

import (
	"fmt"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func main() {
	t, err := auth.Issue(auth.Claims{
		Subject:      "prodsim",
		Role:         auth.RoleAdmin,
		SupplierID:   "sup_61d822c6ab9714ca11f20db9",
		IsConfigured: true,
	}, auth.IssueOptions{
		Secret: "dev-only-change-me",
		Issuer: "pegasusx-dev",
	})
	if err != nil {
		panic(err)
	}
	fmt.Print(t)
}
