package main

import (
	"flag"
	"fmt"

	"github.com/HawkMachine/calibre_tools/calibre"
)

var library = flag.String("library", "", "path to the Calibre library (--with-library)")

func main() {
	flag.Parse()

	if *library == "" {
		fmt.Println("library cannot be empty")
		return
	}
	cdb := calibre.New(*library)
	books, err := cdb.List()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	for _, b := range books {
		fmt.Printf("%s\t%s\n", b.UUID, b.Title)
		for _, f := range b.Formats {
			fmt.Printf("\t%s\n", f)
		}
	}
}
