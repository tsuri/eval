package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	fmt.Println("hello world")

	fmt.Println("------------------")
	files, err := ioutil.ReadDir("./")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		fmt.Println(f.Name())
	}
	_, err = os.ReadFile("evaluations/graphs/comparison.star")
	if err != nil {
		fmt.Println("Failure ", err)
	} else {
		fmt.Println("Success")
	}

}
