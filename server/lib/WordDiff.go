package lib

import (
	"sync"
    "github.com/armon/go-radix"
)

type WordDiff struct {
    tree *radix.Tree
	// Maps do not allow concurrent reads and writes in Go, so we must use a mutex
	mutex sync.Mutex
}

func NewWordDiff() *WordDiff {
	w := &WordDiff{}
    w.tree = radix.New()

	return w
}

func (w *WordDiff) Lock() {
	w.mutex.Lock()
}

func (w *WordDiff) Unlock() {
	w.mutex.Unlock()
}

func (w *WordDiff) IncWord(word string) {
	w.Lock()
	defer w.Unlock()

	count, ok := w.tree.Get(word)
	if !ok {
        w.tree.Insert(word, 1)
	} else {
        w.tree.Insert(word, count.(int) + 1)
	}
}

func (w *WordDiff) GetUnlocked(word string) int {
	count, ok := w.tree.Get(word)
	if !ok {
        return 0
	} else {
        return count.(int)
    }
}

func (w *WordDiff) Get(word string) int {
	w.Lock()
	defer w.Unlock()

	return w.GetUnlocked(word)
}

func (w *WordDiff) Add(diff *WordDiff) {
	diff.Lock()
	w.Lock()
	defer diff.Unlock()
	defer w.Unlock()

    diff.WalkUnlocked(func (word string, count int) {
		currentCount, ok := w.tree.Get(word)
		if !ok {
            w.tree.Insert(word, count)
		} else {
            w.tree.Insert(word, currentCount.(int) + count)
        }
    })
}

func (w *WordDiff) Sub(diff *WordDiff) {
	diff.Lock()
	w.Lock()
	defer diff.Unlock()
	defer w.Unlock()

    diff.WalkUnlocked(func (word string, count int) {
		currentCount, ok := w.tree.Get(word)
		if !ok {
            w.tree.Insert(word, -count)
		} else {
            w.tree.Insert(word, currentCount.(int) - count)
        }
    })
}

func (w *WordDiff) WalkUnlocked(walkFunc func(string, int)) {
    w.tree.Walk(func (word string, count interface{}) bool {
        walkFunc(word, count.(int))
        return false
    })
}

func (w *WordDiff) Walk(walkFunc func(string, int)) {
	w.Lock()
	defer w.Unlock()

	w.WalkUnlocked(walkFunc)
}

func (w *WordDiff) Compress() {
	w.Lock()
	defer w.Unlock()
}
