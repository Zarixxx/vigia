package clock

import (
	"testing"
	"time"
)

func TestFakeClock(t *testing.T) {
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	fake := NewFake(start)
	if got := fake.Now(); !got.Equal(start) {
		t.Fatalf("Now() = %v, want %v", got, start)
	}

	fake.Advance(2 * time.Hour)
	want := start.Add(2 * time.Hour)
	if got := fake.Now(); !got.Equal(want) {
		t.Fatalf("Now() after Advance = %v, want %v", got, want)
	}

	fake.Set(start)
	if got := fake.Now(); !got.Equal(start) {
		t.Fatalf("Now() after Set = %v, want %v", got, start)
	}
}
