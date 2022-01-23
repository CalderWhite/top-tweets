package lib

import (
	"log"
	"sort"
	"sync"

	mtrie "github.com/dghubble/trie"
	"github.com/openacid/slim/encode"
	"github.com/openacid/slim/trie"
)

// use global codec instead of adding it to each WordDiff object
var TrieCodec encode.I16 = encode.I16{}

// one definition file since this is used in 2 places
type WordDiff struct {
	Trie *mtrie.RuneTrie
	// the trie package won't let me concurrently read and write, so I must use a mutux
	// NOTE: I don't actually care about this race condition. It will become correct over time.
	Lock sync.Mutex
}

func NewWordDiff() *WordDiff {
	wordDiff := &WordDiff{}
	wordDiff.Trie = mtrie.NewRuneTrie()

	return wordDiff
}

// adds all the counts from src to dst
func (dst *WordDiff) Add(src *trie.SlimTrie) {
	dst.Lock.Lock()
	defer dst.Lock.Unlock()
	src.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec.Decode(value)
		countInt := int(count.(int16))
		currentCount, ok := dst.Trie.Get(string(word)).(int)
		if !ok {
			currentCount = 0
		}

		dst.Trie.Put(string(word), currentCount+countInt)
		return true
	})
}

// subtracts all the counts from src to dst
func (dst *WordDiff) Sub(src *trie.SlimTrie) {
	dst.Lock.Lock()
	defer dst.Lock.Unlock()
	src.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		_, count := TrieCodec.Decode(value)
		countInt := int(count.(int16))
		currentCount, ok := dst.Trie.Get(string(word)).(int)
		if !ok {
			currentCount = 0
		}

		dst.Trie.Put(string(word), currentCount-countInt)
		return true
	})
}

func (dst *WordDiff) IncWord(word string) {
	dst.Lock.Lock()
	defer dst.Lock.Unlock()

	count, ok := dst.Trie.Get(word).(int)
	if !ok {
		count = 0
	}

	dst.Trie.Put(word, count+1)
}

func (dst *WordDiff) GetStrie() *trie.SlimTrie {
	// should make this 1 allocation to optimize speed.
	// also don't add if count = 0
	counts := NewCountSlice()
	dst.Trie.Walk(func(word string, _count interface{}) error {
		count, ok := _count.(int)
		if !ok {
			return nil
		}
		counts.Add(word, int16(count))
		return nil
	})

	sort.Sort(counts)

	out, err := trie.NewSlimTrie(TrieCodec, counts.StringSlice, counts.Counts, trie.Opt{
		Complete: trie.Bool(true),
	})

	if err != nil {
		log.Fatal("Failed to create SlimTrie", err)
	}

	return out
}
