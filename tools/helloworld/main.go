package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openimsdk/gomake/mageutil"
)

func main() {

	index := flag.Int("i", 0, "Index number")
	config := flag.String("c", "", "Configuration directory")
	// Parse the flags
	flag.Parse()
	mageutil.PrintBlue(fmt.Sprintf("This is a helloworld tool. Program: %s, args: -i %d -c %s", os.Args[0], *index, *config))
}
