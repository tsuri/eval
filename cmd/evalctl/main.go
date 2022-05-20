package main

import (
	"eval/cmd/evalctl/cmd"

	"github.com/kyokomi/emoji"
)

func main() {
	emoji.Println("Hold my :beer:!!!")
	cmd.Execute()
}
