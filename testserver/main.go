package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() { 
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello! Time: %s - Shivam's Assignment!\n", time.Now().Format(time.RFC3339))
	}) 

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok") 
	})

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	log.Printf("test server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)   
	}
}
