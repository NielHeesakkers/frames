// internal/auth/password_test.go
package auth

import "testing"

func TestHashAndVerify(t *testing.T) {
	h, err := HashPassword("hunter2")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword(h, "hunter2")
	if err != nil || !ok {
		t.Fatalf("want ok, got ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword(h, "wrong")
	if err != nil || ok {
		t.Fatalf("want not-ok, got ok=%v err=%v", ok, err)
	}
}

func TestVerify_MalformedHash(t *testing.T) {
	_, err := VerifyPassword("not-a-hash", "pw")
	if err == nil {
		t.Fatal("expected error on malformed hash")
	}
}
