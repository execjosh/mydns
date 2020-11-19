// Copyright (C) 2021  execjosh
// SPDX-License-Identifier: AGPL-3.0-or-later

package blocklist

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/execjosh/mydns/internal/globtrie"
	"github.com/execjosh/mydns/internal/stringset"
	"github.com/miekg/dns"
)

type set interface {
	Insert(string) error
	Contains(string) bool
}

// Blocklist represents an immutable set of FQDNs to block.
type Blocklist struct {
	exact set
	glob  set
}

// Empty returns an empty Blocklist.
func Empty() *Blocklist {
	return &Blocklist{
		exact: stringset.New(),
		glob:  globtrie.New(),
	}
}

// Load loads a blocklist from an io.Reader.
func Load(r io.Reader) (*Blocklist, uint, error) {
	bl := Empty()

	var cnt uint
	s := bufio.NewScanner(r)
	for s.Scan() {
		l := s.Text()
		if _, ok := dns.IsDomainName(l); !ok {
			continue
		}
		l = dns.CanonicalName(l)

		var set set
		if strings.Contains(l, "*") {
			set = bl.glob
		} else {
			set = bl.exact
		}

		if err := set.Insert(l); err != nil {
			log.Println(err)
		} else {
			cnt++
		}
	}
	if err := s.Err(); err != nil {
		return bl, cnt, fmt.Errorf("loading blocklist: %w", err)
	}

	return bl, cnt, nil
}

// Contains returns whether the specified fqdn is included in the blocklist.
func (bl *Blocklist) Contains(fqdn string) bool {
	return bl.exact.Contains(fqdn) || bl.glob.Contains(fqdn)
}
