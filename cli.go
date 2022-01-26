package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		r, failure := Interpret(os.Args[1])
		if failure != nil {
			fmt.Printf("%s: %v", os.Args[1], failure)
		} else {
			fmt.Println(r)
		}
	}
}