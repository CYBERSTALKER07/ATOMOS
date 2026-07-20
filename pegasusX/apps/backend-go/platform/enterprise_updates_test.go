package platform

import (
	"strings"
	"testing"
)

func TestNormalizeChannel_EnterpriseAliases(t *testing.T) {
	for _, in := range []string{"enterprise", "production-web", "website", "sideload"} {
		if got := NormalizeChannel(in); got != EnterpriseChannel {
			t.Fatalf("NormalizeChannel(%q)=%q", in, got)
		}
	}
	if got := NormalizeChannel(""); got != "production" {
		t.Fatalf("empty → %q", got)
	}
}

func TestDefaultEnterpriseManifestURL_Supplier(t *testing.T) {
	android := DefaultEnterpriseManifestURL("ADMIN", "android")
	ios := DefaultEnterpriseManifestURL("ADMIN", "ios")
	desktop := DefaultEnterpriseManifestURL("ADMIN", "desktop")
	if android == "" || ios == "" || desktop == "" {
		t.Fatal("expected supplier URLs")
	}
	if !strings.Contains(android, "/android/supplier/updater.json") {
		t.Fatalf("android url %q", android)
	}
	if !strings.Contains(ios, "/ios/supplier/updater.json") {
		t.Fatalf("ios url %q", ios)
	}
	if !strings.Contains(desktop, "/supplier-desktop/windows/x86_64/updater.json") {
		t.Fatalf("desktop url %q", desktop)
	}
}

func TestDefaultEnterpriseManifestURL_DesktopRoles(t *testing.T) {
	cases := []struct {
		role string
		want string
	}{
		{"RETAILER", "retailer-desktop/windows/x86_64/updater.json"},
		{"WAREHOUSE", "warehouse-desktop/windows/x86_64/updater.json"},
		{"FACTORY", "factory-desktop/windows/x86_64/updater.json"},
	}
	for _, tc := range cases {
		got := DefaultEnterpriseManifestURL(tc.role, "desktop")
		if !strings.Contains(got, tc.want) {
			t.Fatalf("%s desktop url %q want substring %q", tc.role, got, tc.want)
		}
	}
}

func TestEvaluate_EnterpriseFillsUpdateURL(t *testing.T) {
	svc := NewService(NewMemoryPolicyRepository(), NoopSessionChecker{}, nil)
	resp, err := svc.Evaluate(t.Context(), "ADMIN", "android", "enterprise", "1.0.0", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Channel != EnterpriseChannel {
		t.Fatalf("channel=%q", resp.Channel)
	}
	if resp.UpdateURL == "" {
		t.Fatal("expected default enterprise update_url")
	}
	if !strings.Contains(resp.UpdateURL, "android/supplier/updater.json") {
		t.Fatalf("update_url=%q", resp.UpdateURL)
	}
}

func TestNormalizeChannel_StoreAliases(t *testing.T) {
	for _, in := range []string{"production", "store", "appstore", "play"} {
		if got := NormalizeChannel(in); got != StoreChannel {
			t.Fatalf("NormalizeChannel(%q)=%q want production", in, got)
		}
	}
}

func TestDefaultStoreUpdateURL_Android(t *testing.T) {
	url := DefaultStoreUpdateURL("ADMIN", "android")
	if !strings.Contains(url, "play.google.com") || !strings.Contains(url, "com.pegasusx.supplier") {
		t.Fatalf("store url %q", url)
	}
	driver := DefaultStoreUpdateURL("DRIVER", "android")
	if !strings.Contains(driver, "com.pegasusx.driver") {
		t.Fatalf("driver store url %q", driver)
	}
}

func TestDefaultStoreUpdateURL_iOSRequiresID(t *testing.T) {
	t.Setenv("APP_STORE_ID_SUPPLIER", "")
	t.Setenv("STORE_URL_IOS_SUPPLIER", "")
	if got := DefaultStoreUpdateURL("ADMIN", "ios"); got != "" {
		t.Fatalf("expected empty without App Store id, got %q", got)
	}
	t.Setenv("APP_STORE_ID_SUPPLIER", "id999888777")
	got := DefaultStoreUpdateURL("ADMIN", "ios")
	if !strings.Contains(got, "apps.apple.com") || !strings.Contains(got, "id999888777") {
		t.Fatalf("ios store url %q", got)
	}
}

func TestEvaluate_StoreFillsPlayURL(t *testing.T) {
	svc := NewService(NewMemoryPolicyRepository(), NoopSessionChecker{}, nil)
	resp, err := svc.Evaluate(t.Context(), "RETAILER", "android", "production", "1.0.0", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Channel != StoreChannel {
		t.Fatalf("channel=%q", resp.Channel)
	}
	if !strings.Contains(resp.UpdateURL, "com.pegasusx.retailer") {
		t.Fatalf("update_url=%q", resp.UpdateURL)
	}
}

func TestDefaultStoreUpdateURL_DesktopMSStore(t *testing.T) {
	t.Setenv("STORE_URL_DESKTOP_SUPPLIER", "")
	t.Setenv("MS_STORE_URL_SUPPLIER", "")
	t.Setenv("MAC_APP_STORE_ID_DESKTOP_SUPPLIER", "")
	t.Setenv("MAC_APP_STORE_URL_DESKTOP_SUPPLIER", "")
	t.Setenv("MS_STORE_PRODUCT_ID_SUPPLIER", "9NBLGGH4NNS1")
	got := DefaultStoreUpdateURL("ADMIN", "desktop")
	if !strings.Contains(got, "ms-windows-store://") || !strings.Contains(got, "9NBLGGH4NNS1") {
		t.Fatalf("desktop ms store url %q", got)
	}
}

func TestEvaluate_StoreFillsDesktopURL(t *testing.T) {
	t.Setenv("MS_STORE_PRODUCT_ID_WAREHOUSE", "9WZDNCRFJ3Q2")
	svc := NewService(NewMemoryPolicyRepository(), NoopSessionChecker{}, nil)
	resp, err := svc.Evaluate(t.Context(), "WAREHOUSE", "desktop", "production", "0.1.0", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Channel != StoreChannel {
		t.Fatalf("channel=%q", resp.Channel)
	}
	if !strings.Contains(resp.UpdateURL, "ms-windows-store://") {
		t.Fatalf("update_url=%q", resp.UpdateURL)
	}
}
