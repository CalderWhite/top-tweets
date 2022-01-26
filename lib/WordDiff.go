package lib

import (
    "encoding/gob"
    "bytes"
	"sync"
    "log"
)

type WordDiff struct {
	Words map[string]int
	// Maps do not allow concurrent reads and writes in Go, so we must use a mutex
	mutex sync.Mutex
}

func NewWordDiff() *WordDiff {
	w := &WordDiff{}
	w.Words = make(map[string]int)

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

	count, ok := w.Words[word]
	if !ok {
		w.Words[word] = 1
	} else {
		w.Words[word] = count + 1
	}
}

func (w *WordDiff) GetUnlocked(word string) int {
	count, ok := w.Words[word]
	if !ok {
		count = 0
	}

	return count
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

	for word, count := range diff.Words {
		currentCount, ok := w.Words[word]
		if !ok {
			currentCount = 0
		}
		w.Words[word] = currentCount + count
	}
}

func (w *WordDiff) Sub(diff *WordDiff) {
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

func (w *WordDiff) WalkUnlocked(walkFunc func(string, int)) {
	for word := range w.Words {
		walkFunc(word, w.GetUnlocked(word))
	}
}

func (w *WordDiff) Walk(walkFunc func(string, int)) {
	w.Lock()
	defer w.Unlock()

	w.WalkUnlocked(walkFunc)
}

// Note: An INCLUSIVE minimum count
func (w *WordDiff) Prune(minCount int) {
    w.Walk(func (word string, count int) {
        if count <= minCount {
            delete(w.Words, word)
        }
    })
}

func (w *WordDiff) Serialize() []byte {
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
