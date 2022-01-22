package main

import (
    "log"
    "sync"
    "sort"
    mtrie "github.com/dghubble/trie"
    "github.com/openacid/slim/trie"
    "github.com/openacid/slim/encode"
)

// use global codec instead of adding it to each WordDiff object
var trieCodec encode.I32 = encode.I32{}

// one definition file since this is used in 2 places
type WordDiff struct {
    trie *mtrie.RuneTrie
    // the trie package won't let me concurrently read and write, so I must use a mutux
    // NOTE: I don't actually care about this race condition. It will become correct over time.
    Lock sync.Mutex
}

func NewWordDiff() *WordDiff {
    wordDiff := &WordDiff{}
    wordDiff.trie = mtrie.NewRuneTrie()

    return wordDiff
}

// adds all the counts from src to dst
func (dst *WordDiff) Add(src *trie.SlimTrie) {
    dst.Lock.Lock()
    defer dst.Lock.Unlock()
    src.ScanFrom("", true, true, func(word []byte, value []byte) bool {
        _, count := trieCodec.Decode(value)
        countInt := int(count.(int32))
        currentCount, ok := dst.trie.Get(string(word)).(int)
        if !ok {
            currentCount = 0
        }


        dst.trie.Put(string(word), currentCount + countInt)
        return true
    })
}


// subtracts all the counts from src to dst
func (dst *WordDiff) Sub(src *trie.SlimTrie) {
    dst.Lock.Lock()
    defer dst.Lock.Unlock()
    src.ScanFrom("", true, true, func(word []byte, value []byte) bool {
        _, count := trieCodec.Decode(value)
        countInt := int(count.(int32))
        currentCount, ok := dst.trie.Get(string(word)).(int)
        if !ok {
            currentCount = 0
        }

        dst.trie.Put(string(word), currentCount - countInt)
        return true
    })
}

func (dst *WordDiff) IncWord(word string) {
    dst.Lock.Lock()
    defer dst.Lock.Unlock()

    count, ok := dst.trie.Get(word).(int)
    if !ok {
        count = 0
    }

    dst.trie.Put(word, count + 1)
}

func (dst *WordDiff) GetStrie() *trie.SlimTrie {
    // should make this 1 allocation to optimize speed.
    // also don't add if count = 0
    counts := NewCountSlice()
    dst.trie.Walk(func(word string, _count interface{}) error {
        count, ok := _count.(int)
        if !ok {
            return nil
        }
        counts.Add(word, int32(count))
        return nil
    })

    sort.Sort(counts)

    out, err := trie.NewSlimTrie(trieCodec, counts.StringSlice, counts.Counts, trie.Opt{
        Complete: trie.Bool(true),
    })

    if err != nil {
        log.Fatal("Failed to create SlimTrie", err)
    }

    return out
}
