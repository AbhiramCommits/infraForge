#!/usr/bin/env python3
"""infraforge-burn: hold memory and spin CPU for cgroup limit testing."""

import os
import sys
import threading
import time


def spin() -> None:
    while True:
        pass


def main() -> None:
    mem_mb = int(os.environ.get("BURN_MEM_MB", "64"))
    cpus = int(os.environ.get("BURN_CPUS", "1"))

    buf = bytearray()
    if mem_mb > 0:
        buf = bytearray(mem_mb * 1024 * 1024)
        for i in range(0, len(buf), 4096):
            buf[i] = 1

    for _ in range(cpus):
        threading.Thread(target=spin, daemon=True).start()

    print(f"burner active: holding {mem_mb} MiB, spinning {cpus} cpu(s)", flush=True)
    while True:
        time.sleep(3600)


if __name__ == "__main__":
    sys.exit(main())
