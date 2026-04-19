// internal/thumbnail/queue.go
package thumbnail

import "sync"

type Priority int

const (
	PrioBackground Priority = 0
	PrioForeground Priority = 1
)

// Queue is a two-bucket priority queue. Foreground items pop first.
// Deduplicates by file id.
type Queue struct {
	mu       sync.Mutex
	fg, bg   []int64
	inQueue  map[int64]Priority
	notifyCh chan struct{}
}

func NewQueue(initialCap int) *Queue {
	return &Queue{
		inQueue:  make(map[int64]Priority, initialCap),
		notifyCh: make(chan struct{}, 1),
	}
}

func (q *Queue) Push(id int64, p Priority) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if cur, ok := q.inQueue[id]; ok {
		// already queued; only boost
		if p > cur {
			q.boostLocked(id)
			q.inQueue[id] = p
		}
		return
	}
	if p == PrioForeground {
		q.fg = append(q.fg, id)
	} else {
		q.bg = append(q.bg, id)
	}
	q.inQueue[id] = p
	q.notifyOne()
}

func (q *Queue) Boost(id int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if cur, ok := q.inQueue[id]; !ok || cur == PrioForeground {
		return
	}
	q.boostLocked(id)
	q.inQueue[id] = PrioForeground
}

func (q *Queue) boostLocked(id int64) {
	for i, v := range q.bg {
		if v == id {
			q.bg = append(q.bg[:i], q.bg[i+1:]...)
			q.fg = append(q.fg, id)
			return
		}
	}
}

// Pop returns -1 if empty.
func (q *Queue) Pop() int64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.fg) > 0 {
		id := q.fg[0]
		q.fg = q.fg[1:]
		delete(q.inQueue, id)
		return id
	}
	if len(q.bg) > 0 {
		id := q.bg[0]
		q.bg = q.bg[1:]
		delete(q.inQueue, id)
		return id
	}
	return -1
}

func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.fg) + len(q.bg)
}

// Notify returns a channel that receives a signal when items are pushed.
func (q *Queue) Notify() <-chan struct{} { return q.notifyCh }

func (q *Queue) notifyOne() {
	select {
	case q.notifyCh <- struct{}{}:
	default:
	}
}
