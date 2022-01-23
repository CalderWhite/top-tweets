package lib

import (
	"sort"
)

type CountSlice64 struct {
	sort.StringSlice
	Counts []int64
}

func (s CountSlice64) Swap(i, j int) {
	s.StringSlice.Swap(i, j)
	s.Counts[i], s.Counts[j] = s.Counts[j], s.Counts[i]
}

func NewCountSlice64() *CountSlice64 {
	return &CountSlice64{}
}

func (s *CountSlice64) Add(word string, count int64) {
	s.StringSlice = append(s.StringSlice, word)
	s.Counts = append(s.Counts, count)
}
