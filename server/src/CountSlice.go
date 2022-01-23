package main

import (
    "sort"
)

type CountSlice struct {
    sort.StringSlice
    Counts []int64
}

func (s CountSlice) Swap(i, j int) {
    s.StringSlice.Swap(i, j)
    s.Counts[i], s.Counts[j] = s.Counts[j], s.Counts[i]
}

func NewCountSlice() *CountSlice {
    return &CountSlice{}
}

func (s *CountSlice) Add(word string, count int64) {
    s.StringSlice = append(s.StringSlice, word)
    s.Counts = append(s.Counts, count)
}
