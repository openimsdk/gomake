package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {

	index := flag.Int("i", 0, "Index number")
	config := flag.String("c", "", "Configuration directory")
	// Parse the flags
	flag.Parse()
	fmt.Printf("This is a helloworld tool. Program: %s, args: -i %d -c %s\n", os.Args[0], *index, *config)
}
