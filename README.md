# consul-fixture-seed

> **WRITE TOOL: non-production clusters only.** This registers external catalog
> nodes, service instances, and KV keys on a Consul cluster. Point it at a
> staging or lab cluster, never at production. See [Write controls](#write-controls).

Make a declared inventory real on a Consul cluster, idempotently. Given a
reviewed `inventory.json` (a closed set of nodes, services, and KV keys), this
tool registers exactly that set and nothing else. Re-running reads the current
state at Consul's default (leader-forwarded) consistency, never `?stale`, so it
sees its own writes even on a cluster still settling replication, and registers
only what is missing.

Use it to put a Consul cluster into a known, reproducible state for testing,
demos, or rehearsing operational tooling against a fixed fixture.

## inventory.json

The whole format is the `Inventory` struct at the top of `main.go`:

```json
{
  "datacenter": "dc2-verify",
  "hosts": ["10.0.0.1:8500", "10.0.0.2:8500", "10.0.0.3:8500"],
  "nodes": [
    {"name": "verify-node-01", "address": "192.0.2.11"},
    {"name": "verify-node-02", "address": "192.0.2.12"}
  ],
  "services": [
    {"name": "verify-web", "node": "verify-node-01"},
    {"name": "verify-web", "node": "verify-node-02"}
  ],
  "kv": ["verify/config/db", "verify/jobs/cleanup"]
}
```

- **datacenter**: the verification cluster's datacenter name (checked; see Guards).
- **hosts**: the Consul HTTP addresses to write to (see Guards). A write goes to
  the first reachable host; raft replicates.
- **nodes**: registered as external catalog nodes (no agent); addresses should
  be fictional (TEST-NET, e.g. `192.0.2.0/24`).
- **services**: attach to a declared node; the same name on several nodes builds
  a healthy count.
- **kv**: keys only; values are generated (their content is meaningless to a
  hash comparison).

## Usage

```
consul-fixture-seed --inventory examples/inventory.json
```

A summary of what changed is printed to stderr; because the tool is idempotent, a
second run reporting `0 operation(s)` is the observable proof that the inventory
already matches. Exit 0 means the realized state matches the declaration;
non-zero means a write failed.

## Write controls

- **Reviewed destination**: the write host comes from `inventory.json`'s `hosts`,
  not a command-line argument, so a mistyped host can't reach production.
- **datacenter guard**: the chosen host's `/v1/agent/self` must report the
  inventory's `datacenter`. Give the verification cluster a distinct name like
  `dc2-verify`, and a `hosts` entry that turns out to be production (`dc1`) aborts
  the run.
- **agent collision**: a declared node name that collides with a live agent is
  refused (anti-entropy would otherwise delete the external registration and
  manufacture a phantom loss).

## License

This project is licensed under the [MIT License](./LICENSE).
