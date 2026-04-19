// internal/auth/ratelimit_test.go
package auth

import (
	"testing"
	"time"
)

func TestLoginLimiter(t *testing.T) {
	l := NewLoginLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("allow %d expected true", i)
		}
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("expected block after limit")
	}
	// Different IP unaffected.
	if !l.Allow("5.6.7.8") {
		t.Fatal("different ip should be allowed")
	}
}

func TestLoginLimiter_WindowReset(t *testing.T) {
	l := NewLoginLimiter(2, 20*time.Millisecond)
	l.Allow("1.1.1.1")
	l.Allow("1.1.1.1")
	if l.Allow("1.1.1.1") {
		t.Fatal("should block")
	}
	time.Sleep(30 * time.Millisecond)
	if !l.Allow("1.1.1.1") {
		t.Fatal("should reset")
	}
}
