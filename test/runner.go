package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	fmt.Printf("Runner. Listing /app:")

	err := filepath.Walk("/app",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fmt.Println(path, info.Size())
			return nil
		})

	if err != nil {
		log.Println(err)
	}

	fmt.Println("------------")
	out, err := exec.Command("/app/test").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("test says %s\n", out)
}
