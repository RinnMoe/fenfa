package config

import "testing"

func TestTrustedProxiesDefaultToDisabled(t *testing.T) {
	c := Default()
	if got := c.GetTrustedProxies(); got != nil {
		t.Fatalf("expected trusted proxies to be disabled by default, got %v", got)
	}
}

func TestTrustedProxiesLoadFromEnvironment(t *testing.T) {
	t.Setenv("FENFA_TRUSTED_PROXIES", " 127.0.0.1, 10.0.0.0/8 ,, ::1 ")
	c := Default()
	applyEnv(c)

	got := c.GetTrustedProxies()
	want := []string{"127.0.0.1", "10.0.0.0/8", "::1"}
	if len(got) != len(want) {
		t.Fatalf("expected %d trusted proxies, got %v", len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("expected trusted proxy %q at index %d, got %q", want[index], index, got[index])
		}
	}
}

func TestSetupTokenLoadsFromEnvironment(t *testing.T) {
	t.Setenv("FENFA_SETUP_TOKEN", "bootstrap-secret")
	c := Default()
	applyEnv(c)
	if c.Auth.SetupToken != "bootstrap-secret" {
		t.Fatalf("expected setup token from environment, got %q", c.Auth.SetupToken)
	}
}

func TestNormalizeOrigin(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "HTTPS://Example.COM/", want: "https://example.com"},
		{input: "https://example.com:443", want: "https://example.com"},
		{input: "http://example.com:8080", want: "http://example.com:8080"},
	} {
		got, err := NormalizeOrigin(test.input)
		if err != nil || got != test.want {
			t.Fatalf("NormalizeOrigin(%q) = %q, %v; want %q", test.input, got, err, test.want)
		}
	}
	for _, input := range []string{"", "https://example.com/path", "https://user@example.com", "ftp://example.com", "*"} {
		if _, err := NormalizeOrigin(input); err == nil {
			t.Fatalf("expected NormalizeOrigin(%q) to reject input", input)
		}
	}
}

func TestAdminPathIsNotOverriddenByEnvironment(t *testing.T) {
	t.Setenv("FENFA_ADMIN_PATH", "from-env")
	c := Default()
	applyEnv(c)
	if got := c.GetAdminPath(); got != "admin" {
		t.Fatalf("expected admin path environment variable to be ignored, got %q", got)
	}
}

func TestValidateAdminPath(t *testing.T) {
	for _, path := range []string{"admin", "release-console", "admin_2"} {
		if err := ValidateAdminPath(path); err != nil {
			t.Fatalf("expected %q to be valid, got %v", path, err)
		}
	}
	for _, path := range []string{"", "api", "foo/bar", "admin path"} {
		if err := ValidateAdminPath(path); err == nil {
			t.Fatalf("expected %q to be invalid", path)
		}
	}
}
