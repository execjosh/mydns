// Copyright (C) 2021  execjosh
// SPDX-License-Identifier: AGPL-3.0-or-later

package iplist

import (
	"flag"
	"fmt"
	"net"
	"strings"
)

// IPList represents a comma-separated list of IP addresses to be used with the
// `flag` package.
type IPList struct {
	values []string
}

var _ flag.Value = (*IPList)(nil)

// New returns a new instance of IPList
func New() *IPList {
	return &IPList{}
}

func (l *IPList) String() string {
	var s strings.Builder
	for idx, addr := range l.values {
		if idx > 0 {
			s.Write([]byte(","))
		}
		s.WriteString(addr)
	}
	return s.String()
}

// Set implements `flag.Value`
func (l *IPList) Set(s string) error {
	ss := strings.Split(s, ",")

	seen := map[string]struct{}{}
	for _, ipStr := range ss {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return fmt.Errorf("invalid nameserver IP: %q", ipStr)
		}

		addr := ip.String()
		if _, ok := seen[addr]; ok {
			continue
		}
		seen[addr] = struct{}{}
		l.values = append(l.values, addr)
	}

	return nil
}

// Uniq returns the list in original order, with duplicates removed, keeping the
// first occurrence only.
func (l *IPList) Uniq() []string {
	seen := map[string]struct{}{}
	var uniq []string
	for _, v := range l.values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		uniq = append(uniq, v)
	}
	return uniq
}
