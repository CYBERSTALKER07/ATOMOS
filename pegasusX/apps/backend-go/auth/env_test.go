package auth

import "testing"

func TestEnvClassFrom(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"sandbox", EnvClassSandbox},
		{"SSMR", EnvClassSandbox},
		{"ssmr", EnvClassSandbox},
		{"staging", EnvClassStaging},
		{"production", EnvClassProduction},
		{"prod", EnvClassProduction},
		{"", EnvClassLocal},
		{"dev", EnvClassLocal},
		{"local", EnvClassLocal},
		{"development", EnvClassLocal},
	}
	for _, tc := range cases {
		if got := EnvClassFrom(tc.in); got != tc.want {
			t.Fatalf("EnvClassFrom(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsSandboxAlias(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "ssmr")
	if !IsSandbox() || EnvClass() != EnvClassSandbox {
		t.Fatal("ssmr must alias sandbox")
	}
	t.Setenv("PEGASUSX_ENV", "sandbox")
	if !IsSandbox() {
		t.Fatal("sandbox must be sandbox")
	}
	t.Setenv("PEGASUSX_ENV", "production")
	if IsSandbox() || !IsProduction() {
		t.Fatal("production is not sandbox")
	}
}

func TestIsEnforcedEnv(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "sandbox")
	if !IsEnforcedEnv() {
		t.Fatal("sandbox is enforced")
	}
	t.Setenv("PEGASUSX_ENV", "ssmr")
	if !IsEnforcedEnv() {
		t.Fatal("ssmr alias is enforced")
	}
	t.Setenv("PEGASUSX_ENV", "production")
	if !IsEnforcedEnv() {
		t.Fatal("production is enforced")
	}
	t.Setenv("PEGASUSX_ENV", "")
	if IsEnforcedEnv() {
		t.Fatal("local is not enforced")
	}
}
