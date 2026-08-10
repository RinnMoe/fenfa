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
