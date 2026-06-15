// Command consul-fixture-seed registers a declared inventory (inventory.json)
// on a Consul cluster, idempotently.
//
// WRITE tool, non-production clusters only. The write destination is the
// reviewed "hosts" list in inventory.json, never a command-line argument, so a
// mistyped host cannot reach production. See DESIGN.md / README.md.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Build information, injected at release time by GoReleaser via -ldflags
// (-X main.version / main.commit / main.date). The defaults apply to a plain
// `go build`.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	inventoryPath := flag.String("inventory", "", "inventory.json: the declared inventory, hosts, and datacenter")
	timeout := flag.Duration("timeout", 5*time.Second, "per-request HTTP timeout")
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("consul-fixture-seed %s (commit %s, built %s)\n", version, commit, date)
		return
	}
	if *inventoryPath == "" {
		fmt.Fprintln(os.Stderr, "consul-fixture-seed: --inventory is required")
		os.Exit(2)
	}

	if err := run(*inventoryPath, *timeout); err != nil {
		fmt.Fprintf(os.Stderr, "consul-fixture-seed: %v\n", err)
		os.Exit(1)
	}
}

func run(inventoryPath string, timeout time.Duration) error {
	inv, err := loadInventory(inventoryPath)
	if err != nil {
		return err
	}

	c := &client{http: &http.Client{Timeout: timeout}}
	addr, err := chooseHost(c, inv.Hosts, inv.Datacenter)
	if err != nil {
		return err
	}
	if err := checkAgentCollision(c, addr, inv.Nodes); err != nil {
		return err
	}

	changed := 0
	n, err := seedNodes(c, addr, inv)
	if err != nil {
		return err
	}
	changed += n
	if n, err = seedServices(c, addr, inv); err != nil {
		return err
	}
	changed += n
	if n, err = seedKV(c, addr, inv); err != nil {
		return err
	}
	changed += n

	fmt.Fprintf(os.Stderr, "seed: %d operation(s); inventory matches the declaration\n", changed)
	return nil
}
