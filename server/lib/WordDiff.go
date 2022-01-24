package lib

import (
	"log"
	"sort"
	"sync"

	"github.com/openacid/slim/encode"
	"github.com/openacid/slim/trie"
)

/**
 * The current design for this module is: hashmap (map[string]int) for hot data, and periodic compression to a SlimTrie
 * to prevent heap large heap growth. A hashmap already doesn't have a huge memory footprint, but by using this hybrid approach
 * the heap is extremely stable.
 *
 * Using a HAT-Trie would be the best here, but I have yet to find a Go port that supports 64-bit integers.
 */
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
	w.WalkUnlocked(func(word string, count int) {
		// could potentially delete word from the map at this point, reducing the memory overhead.
		// I think the heap would still be fragmented though, so there isn't actually a significant advantage to this.
		counts.Add(word, int16(count))
	})

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
