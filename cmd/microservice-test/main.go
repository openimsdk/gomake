package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	index := flag.Int("i", 0, "Index number")
	config := flag.String("c", "", "Configuration directory")

	// Parse the flags
	flag.Parse()
	fmt.Printf("This is a microservice.\n")
	fmt.Printf("Program name: %s, args: -i %d -c %s\n", os.Args[0], *index, *config)

	// Generate a random port
	rand.Seed(time.Now().UnixNano())
	port := rand.Intn(65535-1024) + 1024

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	defer listener.Close()

	fmt.Printf("Listening on port %d\n", port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you've hit %s\n", r.URL.Path)
	})

	// Start serving, using the listener we created
	log.Fatal(http.Serve(listener, nil))
}
