package lib

import (
    "encoding/gob"
    "bytes"
	"sync"
    "log"
)

type WordDiff64 struct {
	Words map[string]int64
	// Maps do not allow concurrent reads and writes in Go, so we must use a mutex
	mutex sync.Mutex
}

func NewWordDiff64() *WordDiff64 {
	w := &WordDiff64{}
	w.Words = make(map[string]int64)
	return w
}

func (w *WordDiff64) Lock() {
	w.mutex.Lock()
}

func (w *WordDiff64) Unlock() {
	w.mutex.Unlock()
}

func (w *WordDiff64) IncWord(word string) {
	w.Lock()
	defer w.Unlock()

	count, ok := w.Words[word]
	if !ok {
		w.Words[word] = 1
	} else {
		w.Words[word] = count + 1
	}
}

func (w *WordDiff64) GetUnlocked(word string) int64 {
	count, ok := w.Words[word]
	if !ok {
		count = 0
	}

	return count
}

func (w *WordDiff64) Get(word string) int64 {
	w.Lock()
	defer w.Unlock()

	return w.GetUnlocked(word)
}

func (w *WordDiff64) Add(diff *WordDiff64) {
	diff.Lock()
	w.Lock()
	defer diff.Unlock()
	defer w.Unlock()

	for word, count := range diff.Words {
		currentCount, ok := w.Words[word]
		if !ok {
			currentCount = 0
		}
		w.Words[word] = currentCount + count
	}
}

func (w *WordDiff64) Sub(diff *WordDiff64) {
	diff.Lock()
	w.Lock()
	defer diff.Unlock()
	defer w.Unlock()

	for word, count := range diff.Words {
		currentCount, ok := w.Words[word]
		if !ok {
			currentCount = 0
		}

		w.Words[word] = currentCount - count
	}
}

func (w *WordDiff64) WalkUnlocked(walkFunc func(string, int64)) {
	for word := range w.Words {
		walkFunc(word, w.GetUnlocked(word))
	}
}

func (w *WordDiff64) Walk(walkFunc func(string, int64)) {
	w.Lock()
	defer w.Unlock()

	w.WalkUnlocked(walkFunc)
}

// Note: An INCLUSIVE minimum count
func (w *WordDiff64) Prune(minCount int64) {
    w.Walk(func (word string, count int64) {
        if count <= minCount {
            delete(w.Words, word)
        }
    })
}

func (w *WordDiff64) Serialize() []byte {
    w.Lock()
    defer w.Unlock()
    buffer := bytes.NewBuffer([]byte{})
    encoder := gob.NewEncoder(buffer)
    err := encoder.Encode(w)
    if err != nil {
        log.Fatal(err)
    }

    return buffer.Bytes()
}
