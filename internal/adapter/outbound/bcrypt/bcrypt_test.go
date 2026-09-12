package bcrypt

import "testing"

func TestHashAndCompare(t *testing.T) {
	h := New()

	hashed, err := h.Hash("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if hashed == "secret123" {
		t.Error("Hash() returned the plain-text password")
	}

	if !h.Compare(hashed, "secret123") {
		t.Error("Compare() = false, want true for matching password")
	}
	if h.Compare(hashed, "wrong") {
		t.Error("Compare() = true for wrong password")
	}
}

func TestHashProducesDifferentSalt(t *testing.T) {
	h := New()

	a, _ := h.Hash("secret123")
	b, _ := h.Hash("secret123")
	if a == b {
		t.Error("same password produced identical hashes; missing salt")
	}
}
