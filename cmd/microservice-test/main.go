package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/openimsdk/gomake/mageutil"
)

func main() {
	index := flag.Int("i", 0, "Index number")
	config := flag.String("c", "", "Configuration directory")

	// Parse the flags
	flag.Parse()
	mageutil.PrintBlue(fmt.Sprintf("This is a microservice-test. Program: %s, args: -i %d -c %s", os.Args[0], *index, *config))

	// Generate a random port
	rand.Seed(time.Now().UnixNano())
	port := rand.Intn(65535-1024) + 1024

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		mageutil.PrintErr(fmt.Errorf("failed to listen on port %d: %w", port, err))
		os.Exit(1)
	}
	defer listener.Close()

	mageutil.PrintGreen(fmt.Sprintf("Listening on port %d", port))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you've hit %s\n", r.URL.Path)
	})

	// Start serving, using the listener we created
	if err := http.Serve(listener, nil); err != nil {
		mageutil.PrintErr(fmt.Errorf("HTTP server exited: %w", err))
		os.Exit(1)
	}
}
