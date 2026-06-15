package main

import (
	"errors"
	"strings"
	"testing"
)

// dcOf builds a datacenter lookup from a map; an address absent from the map is
// treated as unreachable.
func dcOf(byAddr map[string]string) func(string) (string, error) {
	return func(addr string) (string, error) {
		if dc, ok := byAddr[addr]; ok {
			return dc, nil
		}
		return "", errors.New("unreachable")
	}
}

func TestSelectHost(t *testing.T) {
	const want = "dc-verify"

	// A reachable host in the declared datacenter is chosen.
	if addr, err := selectHost([]string{"h1", "h2"}, want, dcOf(map[string]string{"h1": want, "h2": want})); err != nil || addr != "h1" {
		t.Fatalf("match: got %q, %v; want h1", addr, err)
	}

	// A reachable host in a DIFFERENT datacenter aborts -- and must NOT skip to
	// the next host, even though that one would match. This is the dangerous
	// case (a hosts entry that points at the wrong cluster).
	addr, err := selectHost([]string{"h1", "h2"}, want, dcOf(map[string]string{"h1": "dc1", "h2": want}))
	if err == nil || addr != "" || !strings.Contains(err.Error(), "datacenter guard") {
		t.Fatalf("mismatch must abort, not skip: got %q, %v", addr, err)
	}

	// An unreachable host falls through to the next reachable one.
	if addr, err := selectHost([]string{"down", "h2"}, want, dcOf(map[string]string{"h2": want})); err != nil || addr != "h2" {
		t.Fatalf("fallthrough: got %q, %v; want h2", addr, err)
	}

	// No reachable host at all.
	if _, err := selectHost([]string{"a", "b"}, want, dcOf(map[string]string{})); err == nil || !strings.Contains(err.Error(), "reachable") {
		t.Fatalf("all unreachable: want a 'reachable' error; got %v", err)
	}
}

func TestCollidingNode(t *testing.T) {
	external := []Node{{Name: "verify-node-01"}, {Name: "verify-node-02"}}

	// External names that do not match any live agent: no collision.
	if name, c := collidingNode(external, map[string]bool{"consul-0": true, "consul-1": true}); c {
		t.Errorf("no collision expected; got %q", name)
	}

	// A declared name that is a live agent: collision, reporting that name.
	if name, c := collidingNode([]Node{{Name: "consul-0"}, {Name: "verify-node-01"}}, map[string]bool{"consul-0": true}); !c || name != "consul-0" {
		t.Errorf("collision: got %q, %v; want consul-0, true", name, c)
	}

	// No live agents at all: no collision.
	if _, c := collidingNode(external, map[string]bool{}); c {
		t.Error("empty agent set; want no collision")
	}
}
