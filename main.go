package main

import (
	"flag"

	"github.com/clementd64/ebook-server/pkg/server"
)

func main() {
	addr := flag.String("addr", ":8080", "Bind address")
	path := flag.String("path", ".", "Path to the epub files")
	flag.Parse()

	app, err := server.New(*path)
	if err != nil {
		panic(err)
	}

	app.Listen(*addr)
}
