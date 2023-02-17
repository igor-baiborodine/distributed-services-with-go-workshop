package main

import (
	"log"

	"github.com/igor-baiborodine/distributed-services-with-go-workshop/LetsGo/internal/server"
)

func main() {
	log.Print("starting server at port 8080")
	srv := server.NewHTTPServer(":8080")
	log.Fatal(srv.ListenAndServe())
}
