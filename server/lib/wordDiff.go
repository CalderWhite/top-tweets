package lib

import (
	"log"
	"sort"
	"sync"

	mtrie "github.com/dghubble/trie"
	"github.com/openacid/slim/encode"
	"github.com/openacid/slim/trie"
)

/**
 * I understand the double implementations for int16 and int64 in this file are not the best.
 * Please tell me your better solution and I will happily implement it. As it stands now, this is the lowest
 * effort way to do this while also creating the smallest design impact.
 */



// use global codec instead of adding it to each WordDiff object
var Trie16Codec encode.I16 = encode.I16{}
var Trie64Codec encode.I64 = encode.I64{}

// one definition file since this is used in 2 places
type WordDiff struct {
	Trie *mtrie.RuneTrie
	// the trie package won't let me concurrently read and write, so I must use a mutux
	// NOTE: I don't actually care about this race condition. It will become correct over time.
	Lock sync.Mutex
	// 0 --> int16
	// 1 --> int64
	// we do care about this optimization as it reduces storage + memory by 4x when using high granularity.
	IntType int
}

func NewWordDiff16() *WordDiff {
	wordDiff := &WordDiff{}
	wordDiff.Trie = mtrie.NewRuneTrie()
	wordDiff.IntType = 0

	return wordDiff
}

func NewWordDiff64() *WordDiff {
	wordDiff := &WordDiff{}
	wordDiff.Trie = mtrie.NewRuneTrie()
	wordDiff.IntType = 1

	return wordDiff
}

// adds all the counts from src to dst
func (dst *WordDiff) Add16(src *trie.SlimTrie) {
	dst.Lock.Lock()
	defer dst.Lock.Unlock()
	src.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		if dst.IntType == 0 {
			_, count := Trie16Codec.Decode(value)
			countInt := int(count.(int16))
			currentCount, ok := dst.Trie.Get(string(word)).(int)
			if !ok {
				currentCount = 0
			}

			dst.Trie.Put(string(word), currentCount+countInt)
	   } else {
			_, count := Trie16Codec.Decode(value)
		   countInt := int64(count.(int16))
		   currentCount, ok := dst.Trie.Get(string(word)).(int64)
			if !ok {
				currentCount = 0
			}

			dst.Trie.Put(string(word), currentCount+countInt)
		}
		return true
	})
}

// subtracts all the counts from src to dst
func (dst *WordDiff) Sub16(src *trie.SlimTrie) {
	dst.Lock.Lock()
	defer dst.Lock.Unlock()
	src.ScanFrom("", true, true, func(word []byte, value []byte) bool {
		if dst.IntType == 0 {
			_, count := Trie16Codec.Decode(value)
			countInt := int(count.(int16))
			currentCount, ok := dst.Trie.Get(string(word)).(int)
			if !ok {
				currentCount = 0
			}

			dst.Trie.Put(string(word), currentCount-countInt)
	   } else {
			_, count := Trie16Codec.Decode(value)
		   countInt := int64(count.(int16))
		   currentCount, ok := dst.Trie.Get(string(word)).(int64)
			if !ok {
				currentCount = 0
			}

			dst.Trie.Put(string(word), currentCount-countInt)
		}
		return true
	})
}

// XXX: For some reason this function causes a memory leak but Add() doesn't
// if the trie is deleted and recreated it gets rid of the memory leak though???
func (dst *WordDiff) IncWord(word string) {
	dst.Lock.Lock()
	defer dst.Lock.Unlock()

	if dst.IntType == 0 {
		count, ok := dst.Trie.Get(word).(int16)
		if !ok {
			count = 0
		}

		dst.Trie.Put(word, count + 1)
	} else {
		count, ok := dst.Trie.Get(word).(int64)
		if !ok {
			count = 0
		}

		dst.Trie.Put(word, count + 1)
	}
}

func (dst *WordDiff) GetStrie() *trie.SlimTrie {
	if dst.IntType == 0 {
		// should make this 1 allocation to optimize speed.
		// also don't add if count = 0
		counts := NewCountSlice16()
		dst.Trie.Walk(func(word string, _count interface{}) error {
			count, ok := _count.(int16)
			if !ok {
				return nil
			}
			counts.Add(word, count)
			return nil
		})

		sort.Sort(counts)

		out, err := trie.NewSlimTrie(Trie16Codec, counts.StringSlice, counts.Counts, trie.Opt{
			Complete: trie.Bool(true),
		})
		if err != nil {
			log.Fatal("Failed to create SlimTrie", err)
		}
		return out
	} else {
		// should make this 1 allocation to optimize speed.
		// also don't add if count = 0
		counts := NewCountSlice64()
		dst.Trie.Walk(func(word string, _count interface{}) error {
			count, ok := _count.(int64)
			if !ok {
				return nil
			}
			counts.Add(word, count)
			return nil
		})

		sort.Sort(counts)

		out, err := trie.NewSlimTrie(Trie64Codec, counts.StringSlice, counts.Counts, trie.Opt{
			Complete: trie.Bool(true),
		})
		if err != nil {
			log.Fatal("Failed to create SlimTrie", err)
		}
		return out
	}

}
