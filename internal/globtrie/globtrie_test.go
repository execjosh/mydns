// Copyright (C) 2021  execjosh
// SPDX-License-Identifier: AGPL-3.0-or-later

package globtrie_test

import (
	"testing"

	"github.com/execjosh/mydns/internal/globtrie"
)

func TestInsert(t *testing.T) {
	lm := globtrie.New()
	lm.Insert("example.com.")
	lm.Insert("sub1.example.com.")
	lm.Insert("*.example.com.")
	lm.Insert("sub2.*.example.com.")
	lm.Insert("sub3.sub1.example.com.")

	if err := lm.Insert("1234567890123456789012345678901234567890123456789012345678901234.example.com"); err == nil {
		t.Error("expected label longer than 63 to give error")
	}

	if err := lm.Insert("!example.com"); err == nil {
		t.Error("expected `!` to give error")
	}

	if err := lm.Insert("com"); err == nil {
		t.Error("expected TLD to give error")
	}
}

func TestContains(t *testing.T) {
	lm := globtrie.New()
	lm.Insert("sub1.example.com.")
	lm.Insert("*.example.com.")
	lm.Insert("sub2.*.example.com.")
	lm.Insert("sub3.sub1.example.com.")

	if !lm.Contains("sub2.ss.example.com") {
		t.Error("expected sub2.ss.example.com to be contained")
	}

	if !lm.Contains("jubilee.example.com") {
		t.Error("expected jubilee.example.com to be contained")
	}

	if lm.Contains("example.com") {
		t.Error("expected example.com NOT to be contained")
	}

	if lm.Contains("1234567890123456789012345678901234567890123456789012345678901234.example.com") {
		t.Error("expected label longer than 63 to be false")
	}

	if lm.Contains("!example.com") {
		t.Error("expected `!` to be false")
	}

	if lm.Contains("com") {
		t.Error("expected TLD to be false")
	}

	if !lm.Contains("sub2.sub1.example.com") {
		t.Error("expected sub2.sub1.example.com to be contained")
	}

	if !lm.Contains("sub4.example.com") {
		t.Error("expected sub4.example.com to be contained")
	}
}
