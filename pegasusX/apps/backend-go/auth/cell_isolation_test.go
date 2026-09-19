package auth

import (
	"errors"
	"testing"
)

func TestCellIsolation_UZJWTRejectedByEUSecret(t *testing.T) {
	tok, err := Issue(Claims{Role: RoleAdmin, SupplierID: "sup-uz", HomeCell: "cell-uz", MarketCode: "UZ"}, IssueOptions{Secret: "uz-cell-jwt"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Parse(tok, "eu-cell-jwt")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("UZ JWT on EU secret: %v", err)
	}
}

func TestCellIsolation_UZHomeCellRejectedOnEUAPI(t *testing.T) {
	t.Setenv("CELL_JWT_ENFORCE", "true")
	t.Setenv("HOME_CELL", "cell-eu")
	tok, err := Issue(Claims{Role: RoleAdmin, SupplierID: "sup-uz", HomeCell: "cell-uz", MarketCode: "UZ"}, IssueOptions{Secret: "shared-misconfig"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Parse(tok, "shared-misconfig")
	if !errors.Is(err, ErrWrongCell) {
		t.Fatalf("UZ home_cell on EU API: %v", err)
	}
}

func TestCellIsolation_EUTokenAcceptedOnEUAPI(t *testing.T) {
	t.Setenv("CELL_JWT_ENFORCE", "true")
	t.Setenv("HOME_CELL", "cell-eu")
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	tok, err := Issue(Claims{Role: RoleAdmin, SupplierID: "sup-eu", HomeCell: "cell-eu", MarketCode: "EU"}, IssueOptions{Secret: "eu-cell-jwt"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(tok, "eu-cell-jwt")
	if err != nil {
		t.Fatal(err)
	}
	if got.HomeCell != "cell-eu" {
		t.Fatalf("home_cell=%q", got.HomeCell)
	}
}

func TestCellIsolation_KafkaTopicsDisjoint(t *testing.T) {
	uz := []string{"staging.events.orders", "cell-uz.events.orders"}
	eu := []string{"cell-eu.events.orders", "cell-eu.events.spatial", "cell-eu.events.realtime", "cell-eu.events.webhooks"}
	set := map[string]struct{}{}
	for _, tpc := range uz {
		set[tpc] = struct{}{}
	}
	for _, tpc := range eu {
		if _, ok := set[tpc]; ok {
			t.Fatalf("topic %q overlaps UZ and EU", tpc)
		}
	}
	uzBoot := "bootstrap.pegasusx-staging-kafka.asia-south1.managedkafka.pegasus-503013.cloud.goog:9092"
	euBoot := "bootstrap.pegasusx-eu-kafka.europe-west1.managedkafka.pegasusx-cell-eu.cloud.goog:9092"
	if uzBoot == euBoot {
		t.Fatal("UZ and EU Kafka bootstrap must differ")
	}
}
