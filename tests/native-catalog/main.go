// native-catalog materializes public, version-matched metadata for explicit
// native integration checks. It never reads operator profiles or credentials.
package main

import (
	"context"
	"github.com/clankercode/oneline-ccc/internal/catalog"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		os.Exit(2)
	}
	data, err := catalog.ConfirmedCodex(context.Background(), "0.166.0", "gpt-6.1-sol", []string{"priority", "ultrafast"})
	if err != nil {
		os.Stderr.WriteString("Public native catalog fixture unavailable\n")
		os.Exit(1)
	}
	if os.WriteFile(os.Args[1], data, 0600) != nil {
		os.Exit(1)
	}
}
