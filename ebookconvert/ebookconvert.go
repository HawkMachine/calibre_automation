package ebookconvert

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/HawkMachine/calibre_automation/calibre"
)

func PathConvert(book string, output string) error {
	return exec.Command("ebook-convert", book, output).Run()
}

func CalibreBookConvert(book *calibre.CalibreBook, dir string, format string) (string, error) {
	if len(book.Formats) == 0 {
		return "", fmt.Errorf("book has no formats")
	}

	var newPath string
	var err error
	for _, bformat := range book.Formats {
		newPath = filepath.Join(dir, filepath.Base(bformat)+format)
		err = PathConvert(bformat, newPath)
		if err == nil {
			break
		} else {
			fmt.Printf("Failed id=%s, title=%s, err=%v\n", book.Identifiers, book.Title, err)
		}
	}
	return newPath, err
}

func CalibreBooksConvert(books []*calibre.CalibreBook, dir string, format string) ([]string, error) {
	var res []string
	for _, b := range books {
		r, err := CalibreBookConvert(b, dir, format)
		if err != nil {
			return res, err
		}
		res = append(res, r)
	}
	return res, nil
}

type workParams struct {
	DryRun bool
	TmpDir string
	Format string
}

type workResult struct {
	Err  error
	Path string
}

type calibreWorkItem struct {
	Book   *calibre.CalibreBook
	Params workParams
	Result workResult
}

type pathWorkItem struct {
	Path   string
	Params workParams
	Result workResult
}

func calibreWorker(in chan calibreWorkItem, out chan calibreWorkItem) {
	for req := range in {
		var path string
		var err error
		if req.Params.DryRun {
			fmt.Printf("ebook-convert %s canclled (--dry_run flag)\n", req.Book.Title)
			err = fmt.Errorf("conversion cancelled because of dry run flag")
		} else {
			path, err = CalibreBookConvert(req.Book, req.Params.TmpDir, req.Params.Format)
		}
		req.Result.Path = path
		req.Result.Err = err
		out <- req
	}
}

func pathWorker(in chan pathWorkItem, out chan pathWorkItem) {
	for req := range in {
		var path string
		var err error
		if req.Params.DryRun {
			fmt.Printf("ebook-convert %s canclled (--dry_run flag)\n", req.Path)
			err = fmt.Errorf("conversion cancelled because of dry run flag")
		} else {
			path = filepath.Join(req.Params.TmpDir, filepath.Base(req.Path)+req.Params.Format)
			err = PathConvert(req.Path, path)
		}
		req.Result.Path = path
		req.Result.Err = err
		out <- req
	}
}

type EbookConverter struct {
	threads int
	dryRun  bool
}

func New(threads int) *EbookConverter {
	return &EbookConverter{threads: threads}
}

func (ec *EbookConverter) PathsConvert(books []string, outputDir string, format string) ([]string, error) {
	in := make(chan pathWorkItem, 100)
	workerOut := make(chan pathWorkItem, 100)
	aggrOut := make(chan []string)

	// Start workers
	for i := 0; i < ec.threads; i++ {
		go pathWorker(in, workerOut)
	}

	// Start thread gathering results from workers.
	go func(books []string, in chan pathWorkItem, out chan []string) {
		newPaths := []string{}
		for range books {
			req := <-in
			if req.Result.Err != nil {
				newPaths = append(newPaths, req.Path)
			}
		}
		out <- newPaths
	}(books, workerOut, aggrOut)

	// Add all books to the input channel.
	for _, b := range books {
		in <- pathWorkItem{
			Path: b,
			Params: workParams{
				Format: format,
				TmpDir: outputDir,
				DryRun: ec.dryRun,
			},
		}
	}

	newPaths := <-aggrOut
	return newPaths, nil
}

func (ec *EbookConverter) CalibreBooksConvert(books []*calibre.CalibreBook, outputDir string, format string) ([]string, error) {
	in := make(chan calibreWorkItem, 100)
	workerOut := make(chan calibreWorkItem, 100)
	aggrOut := make(chan []string)

	// Start workers
	for i := 0; i < ec.threads; i++ {
		go calibreWorker(in, workerOut)
	}

	// Start thread gathering results from workers.
	go func(books []*calibre.CalibreBook, in chan calibreWorkItem, out chan []string) {
		newPaths := []string{}
		for range books {
			req := <-in
			if req.Result.Err != nil {
				newPaths = append(newPaths, req.Result.Path)
			}
		}
		out <- newPaths
	}(books, workerOut, aggrOut)

	// Add all books to the input channel.
	for _, b := range books {
		in <- calibreWorkItem{
			Book: b,
			Params: workParams{
				Format: format,
				TmpDir: outputDir,
				DryRun: ec.dryRun,
			},
		}
	}

	newPaths := <-aggrOut
	return newPaths, nil
}
