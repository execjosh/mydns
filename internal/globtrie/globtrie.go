// Copyright (C) 2021  execjosh
// SPDX-License-Identifier: AGPL-3.0-or-later

package globtrie

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

type node map[string]node

// GlobTrie represents a trie of domain name labels that can be globbed. Some
// examples of valid values to insert are:
//   - `example.com.`
//   - `sub1.example.com.`
//   - `*.example.com.`
//   - `sub2.*.example.com.`
//   - `sub3.sub1.example.com.`
//
// The trie if these examples had been inserted would be:
//
//    com --> example --+--> sub1 --+--> !
//                      |           |
//                      |           `--> sub3 --> !
//                      |
//                      |--> * --+--> !
//                      |        |
//                      |        `--> sub2 --> !
//                      `--> !
type GlobTrie struct {
	root node
}

// New returns a new instance of GlobTrie.
func New() *GlobTrie {
	return &GlobTrie{
		root: node{},
	}
}

// Insert attempts to insert the given FQDN into the GlobTrie.
// The FQDN can contain globs.
// `*.example.com` matches `sub1.example.com` and `www.example.com` but not
// `example.com` nor `sub2.sub1.example.com`.
func (lm *GlobTrie) Insert(s string) error {
	s = strings.ToLower(s)

	if _, ok := dns.IsDomainName(s); !ok {
		return fmt.Errorf("invalid domain")
	}

	if strings.Contains(s, "!") {
		return fmt.Errorf("invalid domain: cannot contain exclamation point (`!`)")
	}

	labels := dns.SplitDomainName(s)
	if len(labels) < 2 {
		return fmt.Errorf("invalid domain: must have at least two levels")
	}

	// prepend
	labels = append(labels, "")
	copy(labels[1:], labels[0:])
	labels[0] = "!"

	n := lm.root
	for i := len(labels) - 1; i >= 0; i-- {
		label := labels[i]
		if _, ok := n[label]; !ok {
			n[label] = node{}
		}
		n = n[label]
	}

	return nil
}

// Contains returns whether the GlobTrie contains the fqdn.
func (lm *GlobTrie) Contains(s string) bool {
	s = strings.ToLower(s)

	if _, ok := dns.IsDomainName(s); !ok {
		return false
	}

	if strings.ContainsAny(s, "!*") {
		return false
	}

	labels := dns.SplitDomainName(s)
	if len(labels) < 2 {
		return false
	}

	currNode := lm.root
	var prevNode node
	for i := len(labels) - 1; i >= 0; i-- {
		label := labels[i]

		// exact match
		if nextNode, ok := currNode[label]; ok {
			prevNode = currNode
			currNode = nextNode
			continue
		}

		// check curr glob match
		glob, hasGlob := currNode["*"]
		if hasGlob {
			prevNode = nil
			currNode = glob
			continue
		}

		// check prev glob match

		if prevNode == nil {
			return false // no previous node; no match
		}

		glob, hasGlob = prevNode["*"]
		if !hasGlob {
			return false // no glob; no match
		}

		prevNode = nil
		currNode = glob
		i++ // keep the label position
	}

	// `!` means full stop:
	//   - `com-->example-->!` means `example.com` and
	//   - `com-->example-->www-->!` means `www.example.com`
	// If there is no full stop record at this node in the trie, then there is
	// no match. For example, if the trie contained only
	// `com-->example-->www-->!`, then it would not match `example.com`. In
	// order to match `example.com` and `www.example.com`, the node at
	// `com-->example` would need to have both `!` and `www-->!` (or `*-->!`)
	// subtries.
	_, ok := currNode["!"]

	return ok
}
