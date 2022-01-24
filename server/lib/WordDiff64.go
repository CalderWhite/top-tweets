package lib

import (
	"log"
	"sort"
	"sync"

	"github.com/openacid/slim/trie"
)

/**
 * The current design for this module is: hashmap (map[string]int) for hot data, and periodic compression to a SlimTrie
 * to prevent heap large heap growth. A hashmap already doesn't have a huge memory footprint, but by using this hybrid approach
 * the heap is extremely stable.
 *
 * Using a HAT-Trie would be the best here, but I have yet to find a Go port that supports 64-bit integers.
 */

type WordDiff64 struct {
	words map[string]int64
	// Maps do not allow concurrent reads and writes in Go, so we must use a mutex
	mutex sync.Mutex
}

func NewWordDiff64() *WordDiff64 {
	w := &WordDiff64{}
	w.words = make(map[string]int64)
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

	count, ok := w.words[word]
	if !ok {
		w.words[word] = 1
	} else {
		w.words[word] = count + 1
	}
}

func (w *WordDiff64) GetUnlocked(word string) int64 {
	count, ok := w.words[word]
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

	for word, count := range diff.words {
		currentCount, ok := w.words[word]
		if !ok {
			currentCount = 0
		}
		w.words[word] = currentCount + count
	}
}

func (w *WordDiff64) AddTrie16(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec16.Decode(value)
		currentCount, ok := w.words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.words[string(word)] = currentCount + int64(count.(int16))
		return true
	})
}

func (w *WordDiff64) AddTrie64(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec64.Decode(value)
		currentCount, ok := w.words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.words[string(word)] = currentCount + count.(int64)
		return true
	})
}

func (w *WordDiff64) Sub(diff *WordDiff64) {
	diff.Lock()
	w.Lock()
	defer diff.Unlock()
	defer w.Unlock()

	for word, count := range diff.words {
		currentCount, ok := w.words[word]
		if !ok {
			currentCount = 0
		}

		w.words[word] = currentCount - count
	}
}

func (w *WordDiff64) SubTrie16(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec16.Decode(value)
		currentCount, ok := w.words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.words[string(word)] = currentCount - int64(count.(int16))
		return true
	})
}

func (w *WordDiff64) SubTrie64(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec64.Decode(value)
		currentCount, ok := w.words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.words[string(word)] = currentCount - count.(int64)
		return true
	})
}

func (w *WordDiff64) WalkUnlocked(walkFunc func(string, int64)) {
	for word := range w.words {
		walkFunc(word, w.GetUnlocked(word))
	}
}

func (w *WordDiff64) Walk(walkFunc func(string, int64)) {
	w.Lock()
	defer w.Unlock()

	w.WalkUnlocked(walkFunc)
}

func (w *WordDiff64) GetSlimTrie64Unlocked() *trie.SlimTrie {
	counts := NewCountSlice64()
	w.WalkUnlocked(func(word string, count int64) {
		counts.Add(word, count)
	})

	sort.Sort(counts)

	out, err := trie.NewSlimTrie(TrieCodec64, counts.StringSlice, counts.Counts, trie.Opt{
		Complete: trie.Bool(true),
	})
	if err != nil {
		log.Fatal("Failed to create SlimTrie", err)
	}
	return out
}

func (w *WordDiff64) GetSlimTrie64() *trie.SlimTrie {
	w.Lock()
	defer w.Unlock()

	return w.GetSlimTrie64Unlocked()
}

func (w *WordDiff64) Compress() {
	w.Lock()
	defer w.Unlock()
	w.words = make(map[string]int64)
}
