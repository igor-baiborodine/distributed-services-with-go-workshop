package main

import (
	"log"

	"github.com/igor-baiborodine/distributed-services-with-go-workshop/LetsGo/internal/server"
)

func main() {
	srv := server.NewHTTPServer(":8080")
	log.Fatal(srv.ListenAndServe())
}
