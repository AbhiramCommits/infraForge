// Command burner allocates memory in a loop and spins CPU workers, for
// exercising cgroup limits.
//
//	go run ./hack -mem-mb 512          # under a 256M cap this gets OOM-killed
//	go run ./hack -cpu 2               # under a 25% CPU cap this gets throttled
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	memMB := flag.Int("mem-mb", 256, "megabytes of memory to allocate and hold")
	cpus := flag.Int("cpu", 1, "number of CPU-spinning workers")
	duration := flag.Duration("duration", 15*time.Second, "how long to run before exiting cleanly")
	flag.Parse()

	allocated := 0
	var buf [][]byte
	for allocated < *memMB {
		chunk := make([]byte, 1<<20)
		for i := 0; i < len(chunk); i += 4096 {
			chunk[i] = 1
		}
		buf = append(buf, chunk)
		allocated++
		if allocated%64 == 0 {
			fmt.Printf("burner: allocated %d MiB\n", allocated)
		}
	}
	fmt.Printf("burner: holding %d MiB\n", allocated)

	for i := 0; i < *cpus; i++ {
		go spin()
	}
	fmt.Printf("burner: spinning %d cpu worker(s)\n", *cpus)

	time.Sleep(*duration)
	_ = buf
	os.Exit(0)
}

var sink byte

func spin() {
	for {
		if sink > 100 {
			sink = 0
		}
		sink++
	}
}
