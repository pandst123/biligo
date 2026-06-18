//go:build windows

package main

import "github.com/fdcs99/biligo/internal/wintray"

func run(configPath string) error {
	return wintray.Run(configPath)
}
