package main

import (
    "log"
    trie "github.com/dghubble/trie"
)

// one definition file since this is used in 2 places
type WordDiff struct {
    trie *trie.RuneTrie
}

func NewWordDiff() *WordDiff {
    wordDiff := &WordDiff{}
    wordDiff.trie = trie.NewRuneTrie()

    return wordDiff
}

// adds all the counts from src to dst
func (dst *WordDiff) Add(src *WordDiff) {
    src.trie.Walk(func(word string, _count interface{}) error {
        count, ok := _count.(int)
        if !ok {
            log.Fatal("Couldn't convert count to intereger when adding 2 tries.")
        }
        currentCount, ok := dst.trie.Get(word).(int)
        if !ok {
            currentCount = 0
        }

        dst.trie.Put(word, count + currentCount)
        return nil
    })
}


// subtracts all the counts from src to dst
func (dst *WordDiff) Sub(src *WordDiff) {
    src.trie.Walk(func(word string, _count interface{}) error {
        count, ok := _count.(int)
        if !ok {
            log.Fatal("Couldn't convert count to intereger when subtracting 2 tries.")
        }
        currentCount, ok := dst.trie.Get(word).(int)
        if !ok {
            currentCount = 0
        }

        dst.trie.Put(word, currentCount - count)
        return nil
    })
}

func (dst *WordDiff) IncWord(word string) {
    count, ok := dst.trie.Get(word).(int)
    if !ok {
        count = 0
    }

    dst.trie.Put(word, count + 1)
}
