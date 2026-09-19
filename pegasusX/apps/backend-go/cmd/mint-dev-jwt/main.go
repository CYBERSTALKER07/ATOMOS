package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func main() {
	role := flag.String("role", "ADMIN", "JWT role (ADMIN or PLATFORM_ADMIN)")
	subject := flag.String("subject", "", "JWT sub (required for PLATFORM_ADMIN dual-control audit)")
	supplier := flag.String("supplier", "sup_61d822c6ab9714ca11f20db9", "supplier_id for ADMIN tokens")
	secret := flag.String("secret", "dev-only-change-me", "HS256 secret")
	mfaVerified := flag.Bool("mfa", false, "set mfa_verified=true (PLATFORM_ADMIN step-up bypass for local)")
	flag.Parse()

	claims := auth.Claims{
		Subject:      *subject,
		IsConfigured: true,
		MFAVerified:  *mfaVerified,
	}
	switch *role {
	case string(auth.RolePlatformAdmin):
		claims.Role = auth.RolePlatformAdmin
		if claims.Subject == "" {
			claims.Subject = "platform-admin-dev"
		}
	default:
		claims.Role = auth.RoleAdmin
		claims.SupplierID = *supplier
		if claims.Subject == "" {
			claims.Subject = "prodsim"
		}
	}

	t, err := auth.Issue(claims, auth.IssueOptions{
		Secret: *secret,
		Issuer: "pegasusx-dev",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(t)
}
