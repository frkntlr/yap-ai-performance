package dryrun

import (
	"fmt"
)

// PrintSimulation prints a yellow [SIMÜLASYON] (or [SİMÜLASYON]) message to the terminal.
func PrintSimulation(action string) {
	fmt.Printf("\x1b[33m[SİMÜLASYON]\x1b[0m %s\n", action)
}
