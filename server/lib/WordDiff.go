package lib

import (
	"log"
	"sort"
	"sync"

	"github.com/openacid/slim/encode"
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

var TrieCodec16 encode.I16 = encode.I16{}
var TrieCodec64 encode.I64 = encode.I64{}

type WordDiff struct {
	words map[string]int
	Trie  *trie.SlimTrie
	// Maps do not allow concurrent reads and writes in Go, so we must use a mutex
	mutex sync.Mutex
}

func NewWordDiff() *WordDiff {
	w := &WordDiff{}
	w.words = make(map[string]int)
	var err error
	w.Trie, err = trie.NewSlimTrie(TrieCodec16, []string{}, []int16{}, trie.Opt{
		Complete: trie.Bool(true),
	})
	if err != nil {
		log.Fatal(err)
	}

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

	count, ok := w.words[word]
	if !ok {
		w.words[word] = 1
	} else {
		w.words[word] = count + 1
	}
}

func (w *WordDiff) GetUnlocked(word string) int {
	count, ok := w.words[word]
	if !ok {
		count = 0
	}
	trieCount, ok2 := w.Trie.GetI16(word)
	if !ok2 {
		trieCount = 0
	}

	return count + int(trieCount)
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

	for word, count := range diff.words {
		currentCount, ok := w.words[word]
		if !ok {
			currentCount = 0
		}
		w.words[word] = currentCount + count
	}
}

func (w *WordDiff) AddTrie16(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec16.Decode(value)
		currentCount, ok := w.words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.words[string(word)] = currentCount + int(count.(int16))
		return true
	})
}

func (w *WordDiff) AddTrie64(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec64.Decode(value)
		currentCount, ok := w.words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.words[string(word)] = currentCount + int(count.(int64))
		return true
	})
}

func (w *WordDiff) Sub(diff *WordDiff) {
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

func (w *WordDiff) SubTrie16(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec16.Decode(value)
		currentCount, ok := w.words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.words[string(word)] = currentCount - int(count.(int16))
		return true
	})
}

func (w *WordDiff) SubTrie64(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec64.Decode(value)
		currentCount, ok := w.words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.words[string(word)] = currentCount - int(count.(int64))
		return true
	})
}

func (w *WordDiff) WalkUnlocked(walkFunc func(string, int)) {
	for word := range w.words {
		walkFunc(word, w.GetUnlocked(word))
	}

	w.Trie.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, ok := w.words[string(word)]
		if !ok {
			walkFunc(string(word), w.GetUnlocked(string(word)))
		}
		return true
	})
}

func (w *WordDiff) Walk(walkFunc func(string, int)) {
	w.Lock()
	defer w.Unlock()

	w.WalkUnlocked(walkFunc)
}

func (w *WordDiff) GetSlimTrie16Unlocked() *trie.SlimTrie {
	counts := NewCountSlice16()
	for word := range w.words {
		counts.Add(word, int16(w.GetUnlocked(word)))
	}

	sort.Sort(counts)

	out, err := trie.NewSlimTrie(TrieCodec16, counts.StringSlice, counts.Counts, trie.Opt{
		Complete: trie.Bool(true),
	})
	if err != nil {
		log.Fatal("Failed to create SlimTrie", err)
	}
	return out
}

func (w *WordDiff) GetSlimTrie16() *trie.SlimTrie {
	w.Lock()
	defer w.Unlock()

	return w.GetSlimTrie16Unlocked()
}

func (w *WordDiff) Compress() {
	w.Lock()
	defer w.Unlock()
	sTrie := w.GetSlimTrie16Unlocked()
	w.Trie = sTrie
	w.words = make(map[string]int)
}
