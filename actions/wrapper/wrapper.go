package main

import (
	//    "github.com/rs/zerolog"

	"os/exec"

	"github.com/rs/zerolog/log"
)

func main() {
	log.Print("Starting action wrapper")

	_, err := exec.Command("/app/test/sub_/sub").Output()
	log.Err(err).Msg("failed to run action")
}
