package ratelimit

import (
	"testing"
	"time"
)

func TestAllow(t *testing.T) {
	l := New(50 * time.Millisecond)

	if !l.Allow("a") {
		t.Fatal("first request for a should be allowed")
	}
	if l.Allow("a") {
		t.Fatal("second request for a within the interval should be rejected")
	}
	if !l.Allow("b") {
		t.Fatal("first request for b should be allowed independently of a")
	}

	time.Sleep(60 * time.Millisecond)
	if !l.Allow("a") {
		t.Fatal("request for a after the interval should be allowed")
	}
}
