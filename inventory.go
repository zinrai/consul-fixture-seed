package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Inventory is the declared set of objects that may exist, plus where to write
// them. It holds no run parameters, so it can be read identically by any tool
// that operates on the same fixture (e.g. consul-fixture-churn). The whole shape
// of inventory.json is this struct.
type Inventory struct {
	Datacenter string    `json:"datacenter"` // verification cluster's DC name (guard)
	Hosts      []string  `json:"hosts"`      // Consul HTTP addresses to write to
	Nodes      []Node    `json:"nodes"`
	Services   []Service `json:"services"`
	KV         []string  `json:"kv"` // keys only; values are generated
}

// Node is one external catalog node: a name and a (fictional, TEST-NET) address.
type Node struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// Service is one service instance: a name attached to a declared node. The same
// Name on several nodes builds a healthy count.
type Service struct {
	Name string `json:"name"`
	Node string `json:"node"`
}

func loadInventory(path string) (*Inventory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var inv Inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if inv.Datacenter == "" {
		return nil, fmt.Errorf(`%s: "datacenter" must be a non-empty string`, path)
	}
	if len(inv.Hosts) == 0 {
		return nil, fmt.Errorf(`%s: "hosts" must list at least one Consul HTTP address`, path)
	}
	declared := map[string]bool{}
	for _, n := range inv.Nodes {
		if n.Name == "" || n.Address == "" {
			return nil, fmt.Errorf(`%s: each node needs a non-empty "name" and "address"`, path)
		}
		declared[n.Name] = true
	}
	for _, s := range inv.Services {
		if s.Name == "" || s.Node == "" {
			return nil, fmt.Errorf(`%s: each service needs a non-empty "name" and "node"`, path)
		}
		if !declared[s.Node] {
			return nil, fmt.Errorf("%s: service %s placed on undeclared node %s", path, s.Name, s.Node)
		}
	}
	for _, k := range inv.KV {
		if k == "" {
			return nil, fmt.Errorf(`%s: each kv entry must be a non-empty key string`, path)
		}
	}
	return &inv, nil
}
