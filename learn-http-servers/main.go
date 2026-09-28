package main

import (
	"net/http"
)

func main() {
	mux := http.ServeMux{}
	server := http.Server{Handler: &mux, Addr: ":8080"}

	if err := server.ListenAndServe(); err != nil {
		println(err)
	}
}
