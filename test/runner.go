package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	fmt.Printf("Runner VERY NEW MORE THAN THAT. Listing .:")

	// err := filepath.Walk(".",
	// 	func(path string, info os.FileInfo, err error) error {
	// 		if err != nil {
	// 			return err
	// 		}
	// 		fmt.Println(path, info.Size())
	// 		return nil
	// 	})

	// if err != nil {
	// 	log.Println(err)
	// }

	fmt.Println("------------")
	mydir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(mydir)
	fmt.Println("------------")

	out, err := exec.Command("/app/test/sub_/sub").Output()
	if err != nil {
		fmt.Println("ERROR")
		log.Fatal(err)
	}
	fmt.Printf("sub says %s\n", out)

	out, err = exec.Command("/app/test/another_/another").Output()
	if err != nil {
		fmt.Println("ERROR")
		log.Fatal(err)
	}
	fmt.Printf("another says %s\n", out)

}
