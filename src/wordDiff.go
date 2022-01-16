package main

import (
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
