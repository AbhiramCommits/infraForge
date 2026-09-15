# infraforge

CLI for infrastructure provisioning: it drives Ansible playbooks and applies
Linux cgroup v2 resource limits to hosts and VMs.

| Command               | Status      | Purpose                                          |
| --------------------- | ----------- | ------------------------------------------------ |
| `infraforge provision`| implemented | Provision hosts via Ansible + Docker             |
| `infraforge limit`    | implemented | Run a command inside a cgroup v2 with limits     |
| `infraforge vm`       | implemented | Manage VMs (docker backend; libvirt via tag)     |
| `infraforge report`   | implemented | Summarize the most recent runs                   |

## Global flags

| Flag            | Default                  | Description                          |
| --------------- | ------------------------ | ------------------------------------ |
| `--inventory`   | `deploy/inventory.yml`   | Path to the Ansible inventory        |
| `--config`      | `infraforge.yaml`        | Path to the infraforge config file   |
| `--reports-dir` | `reports`                | Directory for run reports            |
| `--dry-run`     | `false`                  | Print planned actions, change nothing|
| `--log-format`  | `text`                   | Log format: `text` or `json`         |

Every log line carries a `run_id` (UUID v4) generated once per invocation.

## Provisioning

`infraforge provision` performs the following steps:

1. Loads the config file (default `infraforge.yaml`).
2. Generates `deploy/inventory.yml` from the config — never edit it by hand.
3. Runs `ansible-playbook deploy/playbooks/site.yml` with the JSON stdout
   callback, streaming each task transition into the log.
4. Writes `reports/<run_id>.json` and exits non-zero if any host fails or is
   unreachable.

With `--dry-run`, ansible runs in `--check --diff` mode.

The playbook builds a node image from `deploy/Dockerfile.node` (the host's
`image` from the config, defaulting to Ubuntu 22.04, plus sshd), starts one
Docker container per host publishing SSH on `127.0.0.1:<ssh_port>`, then
applies:

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

## Limiting

`infraforge limit --name <name> [--cpu-max 25%] [--memory-max 256M]
[--pids-max 64] -- <command...>` creates a cgroup under
`/sys/fs/cgroup/infraforge/<name>`, writes `cpu.max`, `memory.max`,
`memory.high`, `memory.swap.max`, and `pids.max`, forks the command into it,
samples `memory.events` and `cpu.stat` while it runs, and reports whether it
was throttled or OOM-killed. Requires Linux with cgroups v2; other platforms
return a clear error. See `docs/demo-cgroups.md` for a captured run.

## Virtual machines

`infraforge vm start|stop|inspect|list [--backend docker|libvirt]` manages
VMs through a pluggable backend. The docker backend creates containers with
their own PID/UTS/network namespaces and reports namespace IDs from
`/proc/<pid>/ns/*` (Linux hosts only). The libvirt backend is compiled in
with `-tags libvirt` (cgo, `libvirt-dev` required). Without `--backend`, the
backend is auto-detected: docker when its daemon answers, otherwise libvirt
when compiled in. `vm start` picks up name/image from the config file.

## Reporting

Every provision and limit run writes `reports/<run_id>.json`.
`infraforge report [--format table|json]` summarizes the most recent
provision and limit records: hosts provisioned, per-host task counts, tasks
that changed state, limits applied, throttle/OOM counters, and wall time.

## Repository layout

```
cmd/infraforge/    entrypoint
internal/cli/      cobra commands, flags, logging
internal/config/   YAML config parsing and validation
internal/ansible/  inventory generation and ansible-playbook runner
internal/cgroup/   cgroup v2 limits (linux-tagged; stub elsewhere)
internal/virt/     VM backends (docker; libvirt behind build tag)
internal/report/   run report persistence and rendering
deploy/            Dockerfile, playbooks, roles, collection requirements
hack/              burner used by limit demos and integration tests
scripts/           demo scripts
docs/              demo captures
testdata/          test fixtures (config and recorded ansible output)
test/integration/  integration tests (behind -tags integration)
```

## Development

```sh
make build             # build bin/infraforge
make test              # run unit tests
make test-integration  # integration tests: -tags=integration -timeout=10m
make lint              # golangci-lint run
make fmt               # gofmt cmd/ and internal/
```

Integration tests (idempotent double-provision against real Docker
containers, plus a cgroup OOM-kill test) need Linux with cgroups v2, root,
Docker, Ansible, and sshpass. CI runs them on `ubuntu-latest`; see
`.github/workflows/ci.yml`.
