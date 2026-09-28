package main

import (
	"fmt"
	"log"
	"net/http"
)

const PORT = "8080"

func main() {
	mux := http.NewServeMux()
	server := http.Server{Handler: mux, Addr: ":" + PORT}

	mux.Handle("/", http.FileServer(http.Dir(".")))

	fmt.Printf("Server is listening on localhost:%s\n", PORT)
	log.Fatal(server.ListenAndServe())
}
