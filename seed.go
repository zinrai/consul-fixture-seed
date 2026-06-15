package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// --- deterministic identities and values ---

// nodeID derives a stable node id from the node name, so re-running stays
// idempotent. consul-fixture-churn replaces this id when it rehearses a
// re-registration.
func nodeID(name string) string {
	h := sha256.Sum256([]byte(name))
	s := fmt.Sprintf("%x", h[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", s[0:8], s[8:12], s[12:16], s[16:20], s[20:32])
}

// kvValue is generated, not declared: its content is meaningless to a hash
// comparison, so it only needs to be deterministic and to change on a rewrite.
func kvValue(key string, rev int, now string) []byte {
	return []byte(fmt.Sprintf("key=%s\nrev=%d\nts=%s\n", key, rev, now))
}

// --- seed steps (each returns the number of objects it registered) ---

func seedNodes(c *client, addr string, inv *Inventory) (int, error) {
	var current []catalogNode
	if err := c.getJSON(addr, "/v1/catalog/nodes", &current); err != nil {
		return 0, err
	}
	have := map[string]string{} // name -> address
	for _, n := range current {
		have[n.Node] = n.Address
	}
	changed := 0
	for _, n := range inv.Nodes {
		if have[n.Name] == n.Address {
			continue
		}
		body, _ := json.Marshal(registerRequest{
			Datacenter: inv.Datacenter, Node: n.Name, Address: n.Address, ID: nodeID(n.Name),
		})
		if _, err := c.do(http.MethodPut, addr, "/v1/catalog/register", body); err != nil {
			return changed, err
		}
		changed++
	}
	return changed, nil
}

func seedServices(c *client, addr string, inv *Inventory) (int, error) {
	address := map[string]string{}
	for _, n := range inv.Nodes {
		address[n.Name] = n.Address
	}
	onNode := map[string]map[string]bool{} // node -> set of service names
	changed := 0
	for _, s := range inv.Services {
		if onNode[s.Node] == nil {
			names, err := nodeServices(c, addr, s.Node)
			if err != nil {
				return changed, err
			}
			onNode[s.Node] = names
		}
		if onNode[s.Node][s.Name] {
			continue
		}
		body, _ := json.Marshal(registerRequest{
			Datacenter: inv.Datacenter, Node: s.Node, Address: address[s.Node], ID: nodeID(s.Node),
			Service: &registerService{Service: s.Name, ID: s.Name, Port: 80},
		})
		if _, err := c.do(http.MethodPut, addr, "/v1/catalog/register", body); err != nil {
			return changed, err
		}
		onNode[s.Node][s.Name] = true
		changed++
	}
	return changed, nil
}

func nodeServices(c *client, addr, node string) (map[string]bool, error) {
	var doc catalogNodeServices
	if err := c.getJSON(addr, "/v1/catalog/node/"+node, &doc); err != nil {
		if statusOf(err) == http.StatusNotFound {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	names := map[string]bool{}
	for id, svc := range doc.Services {
		name := svc.Service
		if name == "" {
			name = id
		}
		names[name] = true
	}
	return names, nil
}

func seedKV(c *client, addr string, inv *Inventory) (int, error) {
	now := time.Now().Format(time.RFC3339)
	changed := 0
	for _, key := range inv.KV {
		exists, err := kvExists(c, addr, key)
		if err != nil {
			return changed, err
		}
		if exists {
			continue
		}
		if _, err := c.do(http.MethodPut, addr, "/v1/kv/"+key, kvValue(key, 1, now)); err != nil {
			return changed, err
		}
		changed++
	}
	return changed, nil
}

func kvExists(c *client, addr, key string) (bool, error) {
	_, err := c.do(http.MethodGet, addr, "/v1/kv/"+key, nil)
	if err == nil {
		return true, nil
	}
	if statusOf(err) == http.StatusNotFound {
		return false, nil
	}
	return false, err
}
