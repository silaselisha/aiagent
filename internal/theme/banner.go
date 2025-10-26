package theme

import (
	"fmt"
)

// Banner returns a retro anime intergalactic themed banner.
func Banner() string {
	// ANSI colors for neon retro feel
	const cyan = "\033[36m"
	const magenta = "\033[35m"
	const yellow = "\033[33m"
	const reset = "\033[0m"

	art := "" +
		"  ✦✵✷   " + magenta + "STARSEED" + reset + "   ✷✵✦\n" +
		cyan + "   ▄██████▄   ▄▄   ▄▄   ▄██████▄\n" + reset +
		cyan + "  ▐██▀  ▀██▌ ███ ▐███ ▐██▀  ▀██▌\n" + reset +
		cyan + "   ▀██▄▄██▀  ▐███▌███▌ ▀██▄▄██▀\n" + reset +
		yellow + "     ────────────────────────────\n" + reset +
		"   a retro-anime intergalactic navigator for X ✦\n"

	stars := magenta + "       ✦    ✧     ✦     ✧    ✦\n" + reset
	return art + stars
}

// PrintBanner prints the banner to stdout.
func PrintBanner() {
	fmt.Print(Banner())
}
