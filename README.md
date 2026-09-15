# infraforge

CLI for infrastructure provisioning: it drives Ansible playbooks and applies
Linux cgroup v2 resource limits to hosts and VMs.

> Status: skeleton. Commands are stubs; Ansible and cgroups integration comes later.

## Commands

| Command              | Purpose                                        |
| -------------------- | ---------------------------------------------- |
| `infraforge provision` | Provision hosts from the config via Ansible  |
| `infraforge limit`     | Apply cgroup v2 resource limits to hosts     |
| `infraforge vm`        | Manage virtual machines                      |
| `infraforge report`    | Report host state and applied limits         |

## Global flags

| Flag            | Default                  | Description                          |
| --------------- | ------------------------ | ------------------------------------ |
| `--inventory`   | `deploy/inventory.yml`   | Path to the Ansible inventory        |
| `--config`      | `infraforge.yaml`        | Path to the infraforge config file   |
| `--dry-run`     | `false`                  | Print planned actions, change nothing|
| `--log-format`  | `text`                   | Log format: `text` or `json`         |

Every log line carries a `run_id` (UUID v4) generated once per invocation.

## Repository layout

```
cmd/infraforge/    entrypoint
internal/cli/      cobra commands, flags, logging
internal/config/   YAML config parsing and validation
internal/ansible/  Ansible driver (stub)
internal/cgroup/   cgroup v2 limits (stub)
internal/virt/     VM management (stub)
internal/report/   reporting (stub)
deploy/            Ansible inventory and playbooks
testdata/          test fixtures
```

## Development

```sh
make build   # build bin/infraforge
make test    # run unit tests
make lint    # golangci-lint run
make fmt     # gofmt cmd/ and internal/
```
