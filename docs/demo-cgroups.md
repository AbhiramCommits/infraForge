# Demo: cgroups v2 enforcement

This demo runs the CPU/memory burner from `hack/burner.go` inside cgroup v2
limits applied by `infraforge limit`. It was captured on Linux (cgroups v2,
unified hierarchy, run as root) with Go 1.25.

The two scenarios:

1. **256M memory cap**: the burner allocates 512 MiB and must be OOM-killed.
2. **25% CPU cap**: the burner spins two CPU workers and must be throttled.

## Demo 1: 256M memory cap, burner allocates 512M

```
$ infraforge limit --name demo-oom --memory-max 256M -- ./bin/burner -mem-mb 512 -cpu 0
burner: allocated 64 MiB
burner: allocated 128 MiB
burner: allocated 192 MiB
burner: allocated 256 MiB
limit summary
  cgroup:   /sys/fs/cgroup/infraforge/demo-oom
  command:  /opt/demo/burner -mem-mb 512 -cpu 0
  exit:     signal: killed
  cpu:      throttled_usec=0
  memory:   current=204800 peak=268435456 oom_kill=1 high_events=0
  verdict:  oom-killed
time=2026-09-15T10:29:44.404Z level=INFO msg="limit complete" run_id=66d5a10e-2734-4f88-8ec1-bdf54d85fac0 command=limit cgroup=demo-oom verdict=oom-killed throttled_usec=0 oom_kill=1 memory_current=204800 memory_peak=268435456
```

Observations:

- Allocation proceeds until `memory.current` reaches the 256M cap.
- The kernel kills the burner with SIGKILL (`exit: signal: killed`).
- `memory.events` shows `oom_kill=1`.
- `memory.peak` records exactly 268435456 bytes (256 MiB) — the limit held.
- The CLI exits non-zero so CI and scripts can detect the kill.

## Demo 2: 25% CPU cap, burner spins 2 workers

```
$ infraforge limit --name demo-cpu --cpu-max 25% -- ./bin/burner -mem-mb 32 -cpu 2 -duration 8s
burner: holding 32 MiB
burner: spinning 2 cpu worker(s)
... cgroup sampled ... throttled_usec=700922
... cgroup sampled ... throttled_usec=1583744
... cgroup sampled ... throttled_usec=2509430
... cgroup sampled ... throttled_usec=3372901
... cgroup sampled ... throttled_usec=4259004
... cgroup sampled ... throttled_usec=5193995
... cgroup sampled ... throttled_usec=6068058
... cgroup sampled ... throttled_usec=7023081
... cgroup sampled ... throttled_usec=7896936
... cgroup sampled ... throttled_usec=8788929
... cgroup sampled ... throttled_usec=9671146
... cgroup sampled ... throttled_usec=10540493
... cgroup sampled ... throttled_usec=11567163
... cgroup sampled ... throttled_usec=12443608
... cgroup sampled ... throttled_usec=13388694
... cgroup sampled ... throttled_usec=14260537
limit summary
  cgroup:   /sys/fs/cgroup/infraforge/demo-cpu
  command:  /opt/demo/burner -mem-mb 32 -cpu 2 -duration 8s
  exit:     0
  cpu:      throttled_usec=14434296
  memory:   current=8192 peak=544768 oom_kill=0 high_events=0
  verdict:  throttled
time=2026-09-15T10:29:52.452Z level=INFO msg="limit complete" command=limit cgroup=demo-cpu verdict=throttled throttled_usec=14434296 oom_kill=0 memory_peak=544768
```

Observations:

- `cpu.max` is set to `25000 100000` (25% of one core).
- Two spinning workers request two cores; the kernel throttles the cgroup.
- `cpu.stat`'s `throttled_usec` rises monotonically every sample: ~700 ms of
  throttle per 500 ms interval, ending at ~14.4 s of throttled CPU over 8 s
  of wall time.
- The process itself stays healthy and exits 0.

## Reproduce

Requirements: Linux with cgroups v2, root, Go.

```sh
make build
go build -o bin/burner ./hack

bin/infraforge limit --name demo-oom --memory-max 256M -- ./bin/burner -mem-mb 512 -cpu 0
bin/infraforge limit --name demo-cpu --cpu-max 25% -- ./bin/burner -mem-mb 32 -cpu 2 -duration 15s
```

`scripts/demo-runaway.sh` runs both scenarios and is suitable for CI.
