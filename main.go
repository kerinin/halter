package main

import (
	"flag"
	"log"
	"net/http"
)

var (
	dataPath   = flag.String("reward-file", "./rewards.tsv", "File path for saving reward values")
	imagesPath = flag.String("images", "./images", "Directory containing images to evaluate")
)

func main() {
	server, err := NewServer(*dataPath, *imagesPath)
	if err != nil {
		log.Fatalf("Unable to create server: %s", err)
	}

	http.HandleFunc("/toscore", server.Get)
	http.HandleFunc("/scored", server.Post)

	log.Print("Starting server on localhost:8080")
	log.Print("EXAMPLE: Get a randomized batch of images: curl localhost:8080/toscore")
	log.Print("EXAMPLE: Persist a batch of scored images: curl -X POST localhost:8080/scored -d {\"filename\": \"1\"}")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
