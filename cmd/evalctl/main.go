package main

import (
	"eval/cmd/evalctl/cmd"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func cleanup() {
	fmt.Print("\r")
	if cmd.EvalOperation != nil {
		fmt.Printf("You can re-join this evaluation by running `evalctl attach %s`",
			cmd.EvalOperation.Name)
	}
}

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(1)
	}()

	cmd.Execute()
}
