package main

import (
	"fmt"
	"os"
)

func main() {
	version := "v1.0.0"
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		println("Demo Binary Version:", version)
		return
	}

	fmt.Println("This is a running demo binary.")

}
