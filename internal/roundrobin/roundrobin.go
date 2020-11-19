// Copyright (C) 2021  execjosh
// SPDX-License-Identifier: AGPL-3.0-or-later

package roundrobin

import "sync"

// RoundRobin represents a set of strings that are chosen one-after-another in a
// concurrency-safe manner.
type RoundRobin struct {
	list []string
	idx  int
	mu   sync.Mutex
}

// New returns a new RoundRobin instance.
// `ss` is copied to ensure immutability.
func New(ss []string) *RoundRobin {
	r := &RoundRobin{}

	r.list = make([]string, len(ss))
	copy(r.list, ss)

	return r
}

// Next returns the next element.
func (r *RoundRobin) Next() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := r.list[r.idx]
	r.idx = (r.idx + 1) % len(r.list)

	return s
}
