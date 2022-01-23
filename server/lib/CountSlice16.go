package lib

import (
	"sort"
)

type CountSlice16 struct {
	sort.StringSlice
	Counts []int16
}

func (s CountSlice16) Swap(i, j int) {
	s.StringSlice.Swap(i, j)
	s.Counts[i], s.Counts[j] = s.Counts[j], s.Counts[i]
}

func NewCountSlice16() *CountSlice16 {
	return &CountSlice16{}
}

func (s *CountSlice16) Add(word string, count int16) {
	s.StringSlice = append(s.StringSlice, word)
	s.Counts = append(s.Counts, count)
}
