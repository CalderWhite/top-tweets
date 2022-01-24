package lib

import (
	"log"
	"sort"
	"sync"

	"github.com/openacid/slim/trie"
)

/**
 * After much research, it turns out that a Go hashmap is sufficient for our word diff.
 * We can save a lot of space by serializing them as SlimTries, but when in memory using a
 * trie has far too much memory overhead for almost not performance gain. A HAT-Trie could be used,
 * but the existing libraries were lacking 64 bit integers, and a port of tessil's HAT-Trie would be too much work.
 * Thus, a map[string]int and map[string]int64 will suffice.
 */

// TODO: Add a Prune() function that deletes all words with count 1.
//      ==> If they are really supposed to turn up often, if we do this only once every day
//          or so it should prevent from long term memory growth. (Since eventually we will still run out of memory)

// NOTE: ONLY ACCESS USING .Get() and .GetUnlocked()
//       .Get() handles the merge between the SlimTrie and the map.

type WordDiff64 struct {
	Words map[string]int64
	Trie  *trie.SlimTrie
	// Maps do not allow concurrent reads and writes in Go, so we must use a mutex
	mutex sync.Mutex
}

func NewWordDiff64() *WordDiff64 {
	return &WordDiff64{
		Words: make(map[string]int64),
	}
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
	trieCount, ok2 := w.Trie.GetI64(word)
	if !ok2 {
		trieCount = 0
	}
	if !ok {
		return trieCount
	} else {
		return count + trieCount
	}
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

func (w *WordDiff64) AddTrie16(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec16.Decode(value)
		currentCount, ok := w.Words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.Words[string(word)] = currentCount + int64(count.(int16))
		return true
	})
}

func (w *WordDiff64) AddTrie64(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec64.Decode(value)
		currentCount, ok := w.Words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.Words[string(word)] = currentCount + count.(int64)
		return true
	})
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

func (w *WordDiff64) SubTrie16(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec16.Decode(value)
		currentCount, ok := w.Words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.Words[string(word)] = currentCount - int64(count.(int16))
		return true
	})
}

func (w *WordDiff64) SubTrie64(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec64.Decode(value)
		currentCount, ok := w.Words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.Words[string(word)] = currentCount - count.(int64)
		return true
	})
}

func (w *WordDiff64) GetSlimTrie64() *trie.SlimTrie {
	counts := NewCountSlice64()
	for word, _ := range w.Words {
		counts.Add(word, w.Get(word))
	}

	sort.Sort(counts)

	out, err := trie.NewSlimTrie(TrieCodec64, counts.StringSlice, counts.Counts, trie.Opt{
		Complete: trie.Bool(true),
	})
	if err != nil {
		log.Fatal("Failed to create SlimTrie", err)
	}
	return out
}

func (w *WordDiff64) Compress() {
	w.Lock()
	defer w.Unlock()
	sTrie := w.GetSlimTrie64()
	w.Trie = sTrie
	w.Words = make(map[string]int64)
}
