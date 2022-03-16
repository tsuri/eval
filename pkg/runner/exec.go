package main

import (
	"eval/pkg/runner"
	"fmt"
)

func main() {
	a := runner.Running
	fmt.Printf(">> %v\n", a)
	a = 50
	fmt.Printf(">> %v\n", a)
}
