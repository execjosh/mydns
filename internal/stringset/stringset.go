// Copyright (C) 2021  execjosh
// SPDX-License-Identifier: AGPL-3.0-or-later

package stringset

// StringSet is a quick-and-dirty implementation of a set of strings.
type StringSet map[string]struct{}

// New returns a new instance of StringSet.
func New() StringSet {
	return StringSet{}
}

// Insert inserts a string s into the set.
func (set StringSet) Insert(s string) error {
	set[s] = struct{}{}
	return nil
}

// Contains returns whether the set contains a string s.
func (set StringSet) Contains(s string) bool {
	_, ok := set[s]
	return ok
}
