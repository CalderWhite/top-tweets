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

// NOTE: ONLY ACCESS USING .Get() and .GetUnlocked()
//       .Get() handles the merge between the SlimTrie and the map.

var TrieCodec16 encode.I16 = encode.I16{}
var TrieCodec64 encode.I64 = encode.I64{}

type WordDiff struct {
	Words map[string]int
	Trie  *trie.SlimTrie
	// Maps do not allow concurrent reads and writes in Go, so we must use a mutex
	mutex sync.Mutex
}

func NewWordDiff() *WordDiff {
	return &WordDiff{
		Words: make(map[string]int),
	}
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
	trieCount, ok2 := w.Trie.GetI16(word)
	if !ok2 {
		trieCount = 0
	}
	if !ok {
		return int(trieCount)
	} else {
		return count + int(trieCount)
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

	for word, count := range diff.Words {
		currentCount, ok := w.Words[word]
		if !ok {
			currentCount = 0
		}
		w.Words[word] = currentCount + count
	}
}

func (w *WordDiff) AddTrie16(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec16.Decode(value)
		currentCount, ok := w.Words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.Words[string(word)] = currentCount + int(count.(int16))
		return true
	})
}

func (w *WordDiff) AddTrie64(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec64.Decode(value)
		currentCount, ok := w.Words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.Words[string(word)] = currentCount + int(count.(int64))
		return true
	})
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

func (w *WordDiff) SubTrie16(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec16.Decode(value)
		currentCount, ok := w.Words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.Words[string(word)] = currentCount - int(count.(int16))
		return true
	})
}

func (w *WordDiff) SubTrie64(t *trie.SlimTrie) {
	w.Lock()
	defer w.Unlock()

	t.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec64.Decode(value)
		currentCount, ok := w.Words[string(word)]
		if !ok {
			currentCount = 0
		}

		w.Words[string(word)] = currentCount - int(count.(int64))
		return true
	})
}

func (w *WordDiff) GetSlimTrie16() *trie.SlimTrie {
	counts := NewCountSlice16()
	for word, _ := range w.Words {
		counts.Add(word, int16(w.Get(word)))
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

func (w *WordDiff) Compress() {
	w.Lock()
	defer w.Unlock()
	sTrie := w.GetSlimTrie16()
	w.Trie = sTrie
	w.Words = make(map[string]int)
}
