package main

import (
	//    "github.com/rs/zerolog"

	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
)

func main() {
	log.Print("Starting action wrapper 3")

	out, err := exec.Command("/app/actions/generate/generate_/generate").Output()
	log.Err(err).Msg("generate")
	fmt.Print(out)
}
