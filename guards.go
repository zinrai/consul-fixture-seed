package main

import "fmt"

// chooseHost returns the first reachable reviewed host, confirmed to be in the
// declared datacenter, by querying each host's agent for its datacenter.
func chooseHost(c *client, hosts []string, datacenter string) (string, error) {
	return selectHost(hosts, datacenter, func(addr string) (string, error) {
		var self agentSelf
		if err := c.getJSON(addr, "/v1/agent/self", &self); err != nil {
			return "", err
		}
		return self.Config.Datacenter, nil
	})
}

// selectHost is the host-selection decision, separated from the I/O so it can be
// tested directly. The first reachable host must be in the declared datacenter;
// a reachable host in a different datacenter is a misconfigured hosts list and
// aborts the run rather than being written to (it is NOT skipped to the next
// host). dcOf reports a host's live datacenter, or an error if unreachable.
func selectHost(hosts []string, datacenter string, dcOf func(string) (string, error)) (string, error) {
	var last error
	for _, addr := range hosts {
		live, err := dcOf(addr)
		if err != nil {
			last = err
			continue
		}
		if live != datacenter {
			return "", fmt.Errorf("datacenter guard: %s is in %q, inventory declares %q; refusing to write",
				addr, live, datacenter)
		}
		return addr, nil
	}
	return "", fmt.Errorf(`no host in "hosts" is reachable (last error: %v)`, last)
}

// checkAgentCollision refuses a declared node name that collides with a live
// agent: registering an external node under an agent's name would be undone by
// anti-entropy and manufacture a phantom loss.
func checkAgentCollision(c *client, addr string, nodes []Node) error {
	var members []agentMember
	if err := c.getJSON(addr, "/v1/agent/members", &members); err != nil {
		return err
	}
	live := map[string]bool{}
	for _, m := range members {
		live[m.Name] = true
	}
	if name, collides := collidingNode(nodes, live); collides {
		return fmt.Errorf("agent collision: declared node %s is a live agent; "+
			"registering it as an external node would be undone by anti-entropy", name)
	}
	return nil
}

// collidingNode reports the first declared node whose name is a live agent, if
// any. Separated from the I/O so the decision can be tested directly.
func collidingNode(nodes []Node, live map[string]bool) (string, bool) {
	for _, n := range nodes {
		if live[n.Name] {
			return n.Name, true
		}
	}
	return "", false
}
