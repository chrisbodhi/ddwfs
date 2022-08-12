package main

import (
	"fmt"
)

func main() {
	// l := FetchDatasets()
	// fmt.Println("got", len(l), "datasets downloaded as zips")

	Setup()
	Mount("", "./chrisbodhi")

	fmt.Println("tried it")
}
