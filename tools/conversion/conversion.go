package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	//"math/rand"
	"os"
	//"time"
	//"os/exec"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/HawkMachine/calibre_tools/calibre"
)

var libraryFlag = flag.String("library", "", "path to Calibre library (--with-library)")
var formatFlag = flag.String("format", "", "format to add to db")
var threadsFlag = flag.Int("threds", 8, "number of threads")

type workItem struct {
	Book   *calibre.CalibreBook
	TmpDir string
	Format string
	Path   string
	Err    error
}

func convertBook(book *calibre.CalibreBook, dir string, format string) (string, error) {
	var newPath string
	var err error
	for _, bformat := range book.Formats {
		newPath = filepath.Join(dir, filepath.Base(bformat)+format)
		fmt.Printf("ebook-convert %s %s\n", bformat, newPath)
		//time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
		err = exec.Command("ebook-convert", bformat, newPath).Run()
		if err == nil {
			break
		} else {
			fmt.Printf("Failed id=%s, title=%s, err=%v\n", book.Identifiers, book.Title, err)
		}
	}
	return newPath, err
}

func worker(in chan workItem, out chan workItem) {
	for req := range in {
		path, err := convertBook(req.Book, req.TmpDir, req.Format)
		req.Path = path
		req.Err = err
		out <- req
	}
}

func ebookConvert(books []*calibre.CalibreBook, dir string, format string) ([]string, error) {
	in := make(chan workItem, 100)
	out := make(chan workItem, 100)
	pathsOut := make(chan []string)
	for i := 0; i < *threadsFlag; i++ {
		go worker(in, out)
	}

	go func(books []*calibre.CalibreBook, in chan workItem, out chan []string) {
		newPaths := []string{}
		for range books {
			req := <-in
			if req.Err != nil {
				newPaths = append(newPaths, req.Path)
			}
		}
		out <- newPaths
	}(books, out, pathsOut)

	for _, b := range books {
		in <- workItem{Book: b, Format: format, TmpDir: dir}
	}

	newPaths := <-pathsOut

	return newPaths, nil
}

func booksWithoutFormat(books []*calibre.CalibreBook, ext string) []*calibre.CalibreBook {
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

func exit(err error) {
	fmt.Printf("%v\n", err)
	os.Exit(1)
}

func main() {
	flag.Parse()
	if *libraryFlag == "" || *formatFlag == "" {
		exit(fmt.Errorf("--library and --format flags are required!\n"))
	}

	if *threadsFlag < 1 {
		exit(fmt.Errorf("--threads cannot be lower than 1"))
	}

	if !strings.HasPrefix(*formatFlag, ".") {
		*formatFlag = "." + *formatFlag
	}

	// Get all books from the DB.
	cdb := calibre.New(*libraryFlag)
	books, err := cdb.List()
	if err != nil {
		exit(err)
	}

	// Get all books that are missing that format.
	books = booksWithoutFormat(books, *formatFlag)
	calibre.By(func(b1, b2 *calibre.CalibreBook) bool {
		return b1.Title < b2.Title
	}).Sort(books)

	if len(books) == 0 {
		return
	}

	// Create a temp directory for the books.
	tmpDir, err := ioutil.TempDir(os.TempDir(), "ebook_convert")
	if err != nil {
		exit(err)
	}

	// Convert all books to the missing format.
	fmt.Println(tmpDir)
	paths, err := ebookConvert(books, tmpDir, *formatFlag)
	if err != nil {
		exit(err)
	}
	return

	err = cdb.Add(paths)
	if err != nil {
		exit(err)
	}
}
