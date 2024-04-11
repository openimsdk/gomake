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
	fmt.Println("this is hello world for tools"))
	fmt.Printf("Program name: %s, args: -i %d -c %s\n", os.Args[0], *index, *config)


}
