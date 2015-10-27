package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/HawkMachine/calibre_automation/calibre"
	"github.com/HawkMachine/calibre_automation/ebookconvert"
)

func exit(err error) {
	fmt.Printf("Error: %v\n", err)
	os.Exit(1)
}

// GetBooksWithoutFormat filters out books that already have the given format
// and returns only those that don't.
func GetBooksWithoutFormat(books []*calibre.CalibreBook, ext string) []*calibre.CalibreBook {
	booksWithoutFormat := []*calibre.CalibreBook{}
	for _, b := range books {
		hasFormat := false
		for _, p := range b.Formats {
			if strings.ToLower(path.Ext(p)) == ext {
				hasFormat = true
				break
			}
		}
		if !hasFormat {
			booksWithoutFormat = append(booksWithoutFormat, b)
		}
	}
	return booksWithoutFormat
}

// AddFormatIfMssing scans the database and use ebook-convert tool to convert to
// given to given format books that don't have it. A temporary directory is used
// for the created files. These are then added to calibre.
func AddFormatIfMssing(cdb *calibre.CalibreDB, format string, threads int) error {
	if !strings.HasPrefix(format, ".") {
		format = "." + format
	}

	books, err := cdb.List()
	if err != nil {
		return err
	}

	// Get all books that are missing that format.
	books = GetBooksWithoutFormat(books, format)
	calibre.By(func(b1, b2 *calibre.CalibreBook) bool {
		return b1.Title < b2.Title
	}).Sort(books)

	if len(books) == 0 {
		return nil
	}

	// Create a temp directory for the books.
	tmpDir, err := ioutil.TempDir(os.TempDir(), "ebook_convert")
	if err != nil {
		return err
	}

	// Convert all books to the missing format.
	fmt.Println(tmpDir)
	ec := ebookconvert.New(threads)
	paths, err := ec.CalibreBooksConvert(books, tmpDir, format)

	return cdb.Add(paths)
}

func main() {
	libraryFlag := flag.String("library", "", "path to Calibre library (--with-library)")
	formatFlag := flag.String("format", "", "format to add to db")
	threadsFlag := flag.Int("threads", 8, "number of threads")
	flag.Parse()

	if *libraryFlag == "" || *formatFlag == "" {
		exit(fmt.Errorf("--library and --format flags are required!\n"))
	}

	if *threadsFlag < 1 {
		exit(fmt.Errorf("--threads cannot be lower than 1"))
	}

	// Get all books from the DB.
	cdb := calibre.New(*libraryFlag)
	err := AddFormatIfMssing(cdb, *formatFlag, *threadsFlag)
	if err != nil {
		exit(err)
	}
}
