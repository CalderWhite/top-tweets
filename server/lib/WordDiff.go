package lib

import (
	"sync"
    "bytes"
    "log"
    "encoding/gob"
    "github.com/CalderWhite/top-tweets/server/lib/radix"
)

type WordDiff struct {
    Tree *radix.Tree
	// Maps do not allow concurrent reads and writes in Go, so we must use a mutex
	mutex sync.Mutex
}

func NewWordDiff() *WordDiff {
	w := &WordDiff{}
    w.Tree = radix.New()

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

	count, ok := w.Tree.Get(word)
	if !ok {
        w.Tree.Insert(word, 1)
	} else {
        w.Tree.Insert(word, count.(int) + 1)
	}
}

func (w *WordDiff) GetUnlocked(word string) int {
	count, ok := w.Tree.Get(word)
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
		currentCount, ok := w.Tree.Get(word)
		if !ok {
            w.Tree.Insert(word, count)
		} else {
            w.Tree.Insert(word, currentCount.(int) + count)
        }
    })
}

func (w *WordDiff) Sub(diff *WordDiff) {
	diff.Lock()
	w.Lock()
	defer diff.Unlock()
	defer w.Unlock()

    diff.WalkUnlocked(func (word string, count int) {
		currentCount, ok := w.Tree.Get(word)
		if !ok {
            w.Tree.Insert(word, -count)
		} else {
            w.Tree.Insert(word, currentCount.(int) - count)
        }
    })
}

func (w *WordDiff) WalkUnlocked(walkFunc func(string, int)) {
    w.Tree.Walk(func (word string, count interface{}) bool {
        walkFunc(word, count.(int))
        return false
    })
}

func (w *WordDiff) Walk(walkFunc func(string, int)) {
	w.Lock()
	defer w.Unlock()

	w.WalkUnlocked(walkFunc)
}

func (w *WordDiff) Serialize() *bytes.Buffer {
    buffer := bytes.NewBuffer([]byte{})
    encoder := gob.NewEncoder(buffer)
    err := encoder.Encode(w)
    if err != nil {
        log.Fatal(err)
    }

    return buffer
}
