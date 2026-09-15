# cgroups: what infraforge writes and reads

infraforge enforces resource limits through the Linux **cgroup v2 unified
hierarchy**, mounted at `/sys/fs/cgroup`. Every `infraforge limit` run creates
one cgroup at:

```
/sys/fs/cgroup/infraforge/<name>
```

`infraforge` is the parent group; the `<name>` cgroup is where the limited
command lives. All interaction is plain file I/O on the cgroup filesystem —
no daemon, no library.

## Files written at creation

| File                | Written value                        | Meaning                                                            |
| ------------------- | ------------------------------------ | ------------------------------------------------------------------ |
| `cgroup.subtree_control` | `+cpu +memory +pids`            | Enables controllers on the *infraforge* parent so its children expose controller files. A fresh cgroup enables no controllers by default. |
| `cpu.max`           | `"50000 100000"` (or unset)         | CPU bandwidth limit as `<quota> <period>` in microseconds: 50000 µs of CPU per 100000 µs window = 50% of one core. `--cpu-max 25%` writes `"25000 100000"`. |
| `memory.max`        | bytes, e.g. `268435456`             | Hard memory limit. Allocation beyond it fails, and the cgroup's OOM killer eventually kills the offending process. |
| `memory.high`       | bytes (defaults to `memory.max`)    | Throttle threshold: crossing it stalls the workload and ramps reclaim pressure *before* the hard limit kills it. Counted in `memory.events` as `high`. |
| `memory.swap.max`   | `0`                                 | Forbids swap. Without this, the kernel swaps pages out instead of enforcing `memory.max`, so the OOM test would never fire. |
| `pids.max`          | integer, e.g. `64` (or unset)       | Caps the number of processes/threads in the cgroup. |
| `cgroup.procs`      | the child command's PID             | Moves the forked command into the cgroup. The PID is resolved in the writer's PID namespace, moves the whole thread group, and children forked afterwards inherit the cgroup. |

## Files read while the command runs

| File              | What infraforge reads                                     |
| ----------------- | --------------------------------------------------------- |
| `memory.current`  | Current memory usage in bytes (RSS-style, swap excluded).  |
| `memory.peak`     | Highest `memory.current` the cgroup ever observed. This survives process death, so the OOM demo still reports the true peak even when sampling misses the kill. |
| `memory.events`   | Event counters, one per line: `low`, `high`, `max`, `oom`, `oom_kill`, `oom_group_kill`. infraforge watches `oom_kill` (≥1 means the kernel killed a process in this cgroup) and `high` (throttling events). |
| `cpu.stat`        | Accounting fields. infraforge reads `throttled_usec`: total microseconds the cgroup was throttled by `cpu.max`. `nr_periods` / `nr_throttled` count enforcement windows. |

## What the counters prove

- **OOM enforcement**: `memory.events`'s `oom_kill` increments and the child
  dies with `signal: killed` when the burner exceeds `memory.max` (see
  `docs/demo-cgroups.md`).
- **CPU enforcement**: `cpu.stat`'s `throttled_usec` rises monotonically
  while a CPU burner runs under a `cpu.max` quota.
- **Headroom**: `memory.high` lets infraforge see throttling (`high` events)
  before the hard kill happens — the difference between "slow" and "dead".

## Cleanup

`infraforge limit` removes the `<name>` cgroup after the command exits
(`os.Remove` on the directory). A cgroup with live processes refuses
`rmdir`, which is why the child is reaped first. The empty `infraforge`
parent group is left behind; it costs nothing and makes repeated runs
cheaper.
