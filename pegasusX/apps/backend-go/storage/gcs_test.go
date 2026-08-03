package storage

import (
	"errors"
	"testing"
)

func TestIsPlaceholderMediaURL(t *testing.T) {
	if !IsPlaceholderMediaURL("") {
		t.Fatal("empty should be placeholder")
	}
	if !IsPlaceholderMediaURL("https://placehold.co/400x400?text=x") {
		t.Fatal("placehold.co should be placeholder")
	}
	if IsPlaceholderMediaURL("https://storage.googleapis.com/bucket/obj.jpg") {
		t.Fatal("gcs should not be placeholder")
	}
}

func TestValidateEvidenceURI(t *testing.T) {
	BucketName = "pegasusx-evidence"
	t.Cleanup(func() { BucketName = "" })

	if err := ValidateEvidenceURI("https://placehold.co/x"); !errors.Is(err, ErrInvalidEvidenceURI) {
		t.Fatalf("want invalid for placehold, got %v", err)
	}
	if err := ValidateEvidenceURI("https://storage.googleapis.com/pegasusx-evidence/evidence/claims/a.jpg"); err != nil {
		t.Fatalf("want ok for bucket path: %v", err)
	}
	if err := ValidateEvidenceURI("https://storage.googleapis.com/other-bucket/a.jpg"); !errors.Is(err, ErrInvalidEvidenceURI) {
		t.Fatalf("want invalid for wrong bucket, got %v", err)
	}
}

func TestEvidenceFailClosed_RequireInfra(t *testing.T) {
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "true")
	t.Setenv("PEGASUSX_ENV", "local")
	if !EvidenceFailClosed() {
		t.Fatal("REQUIRE_INFRA_ADAPTERS=true must fail closed")
	}
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "false")
	t.Setenv("PEGASUSX_ENV", "ssmr")
	if !EvidenceFailClosed() {
		t.Fatal("PEGASUSX_ENV=ssmr must fail closed")
	}
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "false")
	t.Setenv("PEGASUSX_ENV", "local")
	if EvidenceFailClosed() {
		t.Fatal("local + adapters false must allow placeholders")
	}
}

func TestGenerateUploadTicketFor_FailClosedNoClient(t *testing.T) {
	prev := Client
	Client = nil
	t.Cleanup(func() { Client = prev })
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "true")
	_, _, err := GenerateUploadTicketFor("evidence/claims/x", "jpg")
	if !errors.Is(err, ErrMediaStorageUnavailable) {
		t.Fatalf("want media_storage_unavailable, got %v", err)
	}
}
