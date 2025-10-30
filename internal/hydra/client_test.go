package hydra

import "testing"

func TestNewClientRequiresURL(t *testing.T) {
	if _, err := NewClient(Config{}); err == nil {
		t.Fatalf("expected error for missing admin url")
	}
}
