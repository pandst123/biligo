package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fatalf("%v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s [ERROR] %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
	os.Exit(1)
}
