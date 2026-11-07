package bootstrap

import "testing"

func TestNewLoginFirebaseVerifier_OffOrEmptyProject(t *testing.T) {
	if v := newLoginFirebaseVerifier(nil, nil); v != nil {
		t.Fatal("nil cfg must return nil")
	}
	if v := newLoginFirebaseVerifier(&Config{FirebaseAuthEnabled: false, FirebaseProjectID: "p"}, nil); v != nil {
		t.Fatal("flag off must return nil")
	}
	if v := newLoginFirebaseVerifier(&Config{FirebaseAuthEnabled: true, FirebaseProjectID: ""}, nil); v != nil {
		t.Fatal("empty project must return nil")
	}
}

func TestNewLoginFirebaseVerifier_Enabled(t *testing.T) {
	v := newLoginFirebaseVerifier(&Config{
		FirebaseAuthEnabled: true,
		FirebaseProjectID:   "demo-project",
	}, nil)
	if v == nil {
		t.Fatal("enabled + project must construct verifier")
	}
}
