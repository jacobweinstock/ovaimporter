package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/jacobweinstock/ovaimporter/cmd"
)

func main() {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)

	defer func() {
		signal.Stop(signals)
	}()

	go func() {
		// TODO propagate ctx through Execute
		if err := cmd.Execute(); err != nil {
			exitCode = 1
		}
		close(signals)
	}()

	<-signals
}
