# infraforge

CLI for infrastructure provisioning: it drives Ansible playbooks and applies
Linux cgroup v2 resource limits to hosts and VMs.

| Command               | Status      | Purpose                                        |
| --------------------- | ----------- | ---------------------------------------------- |
| `infraforge provision`| implemented | Provision hosts via Ansible + Docker           |
| `infraforge limit`    | stub        | Apply cgroup v2 resource limits to hosts       |
| `infraforge vm`       | stub        | Manage virtual machines                        |
| `infraforge report`   | stub        | Report host state and applied limits           |

## Global flags

| Flag            | Default                  | Description                          |
| --------------- | ------------------------ | ------------------------------------ |
| `--inventory`   | `deploy/inventory.yml`   | Path to the Ansible inventory        |
| `--config`      | `infraforge.yaml`        | Path to the infraforge config file   |
| `--dry-run`     | `false`                  | Print planned actions, change nothing|
| `--log-format`  | `text`                   | Log format: `text` or `json`         |

Every log line carries a `run_id` (UUID v4) generated once per invocation.

## Provisioning

`infraforge provision` performs the following steps:

1. Loads the config file (default `infraforge.yaml`).
2. Generates `deploy/inventory.yml` from the config — never edit it by hand.
3. Runs `ansible-playbook deploy/playbooks/site.yml` with the JSON stdout
   callback, streaming each task transition into the log.
4. Exits non-zero if any host fails or is unreachable.

With `--dry-run`, ansible runs in `--check --diff` mode.

The playbook builds a node image from `deploy/Dockerfile.node` (Ubuntu 22.04
plus sshd, keyed to the host's `image` from the config), starts one Docker
container per host publishing SSH on `127.0.0.1:<ssh_port>`, then applies:

- `roles/base`: service user, base packages, `/etc/infraforge/node.conf`
  rendered from a template.
- `roles/workload`: a supervisor-managed CPU/memory burner script.

### Prerequisites

- Docker CLI and daemon.
- Python 3 with the Docker SDK on the control node
  (`pip install docker`), for the `community.docker` modules.
- Ansible core with the pinned collections:
  `ansible-galaxy install -r deploy/requirements.yml`.
- `sshpass`, because the generated inventory authenticates over SSH with a
  password.

> Dev-only credentials: the node image bakes in user/password
> `infraforge`/`infraforge` with passwordless sudo, and containers bind SSH
> to loopback only. Replace this with key-based auth before any real use.

## Repository layout

```
cmd/infraforge/    entrypoint
internal/cli/      cobra commands, flags, logging
internal/config/   YAML config parsing and validation
internal/ansible/  inventory generation and ansible-playbook runner
internal/cgroup/   cgroup v2 limits (stub)
internal/virt/     VM management (stub)
internal/report/   reporting (stub)
deploy/            Dockerfile, playbooks, roles, collection requirements
testdata/          test fixtures (config and recorded ansible output)
```

## Development

```sh
make build   # build bin/infraforge
make test    # run unit tests
make lint    # golangci-lint run
make fmt     # gofmt cmd/ and internal/
```
