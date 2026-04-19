// internal/thumbnail/queue_test.go
package thumbnail

import "testing"

func TestQueue_OrderAndBoost(t *testing.T) {
	q := NewQueue(8)
	q.Push(1, PrioBackground)
	q.Push(2, PrioBackground)
	q.Push(3, PrioForeground)
	q.Boost(1) // 1 moves to foreground

	got := []int64{q.Pop(), q.Pop(), q.Pop()}
	// Foreground first: 3 and 1 (order within priority by insertion).
	if got[0] != 3 || got[1] != 1 || got[2] != 2 {
		t.Errorf("pop order = %v", got)
	}
}

func TestQueue_Dedup(t *testing.T) {
	q := NewQueue(8)
	q.Push(1, PrioBackground)
	q.Push(1, PrioBackground) // dedup
	if n := q.Len(); n != 1 {
		t.Errorf("len=%d want 1", n)
	}
}
