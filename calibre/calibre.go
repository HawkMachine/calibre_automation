package calibre

import (
	"encoding/json"
	"os/exec"
	"sort"
	"time"
)

type CalibreDB struct {
	Library string
}

func New(library string) *CalibreDB {
	return &CalibreDB{Library: library}
}

type CalibreBook struct {
	AuthorSort       string     `json:"author_sort"`
	Authors          string     `json:"authors"`
	Comments         string     `json:"comments"`
	Cover            string     `json:"cover"`
	Formats          []string   `json:"formats"`
	Identifiers      int        `json:"id"`
	ISBN             string     `json:"isbn"`
	LastModified     string     `json:"last_modified"`
	PubDate          string     `json:"pubdate"`
	Publisher        string     `json:"publisher"`
	Rating           int        `json:"rating"`
	Series           string     `json:"series"`
	SeriesIndex      float32    `json:"series_index"`
	Size             int        `json:"size"`
	Tags             []string   `json:"tags"`
	Timestasmp       string     `json:"timestasmp"`
	Title            string     `json:"title"`
	UUID             string     `json:"uuid"`
	lastModifiedTime *time.Time `json:"lastmodifiedtime"`
	pubDateTime      *time.Time `json:"pubdatetime"`
}

func (cb *CalibreBook) LastModifedTime() (*time.Time, error) {
	// Example: "2015-04-06T17:14:50+00:00"
	if cb.lastModifiedTime == nil {
		tm, err := time.Parse("2006-Jan-02T15:04:05+00:00", cb.LastModified)
		if err != nil {
			return nil, err
		}
		cb.lastModifiedTime = &tm
	}
	return cb.lastModifiedTime, nil
}

func (c *CalibreDB) List() ([]*CalibreBook, error) {
	cmd := exec.Command("calibredb", "list",
		"--fields",
		"author_sort,authors,comments,cover,formats,identifiers,isbn,last_modified,pubdate,publisher,rating,series,series_index,size,tags,timestamp,title,uuid",
		"--for-machine", "--with-library", c.Library)
	bts, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	books := make([]*CalibreBook, 0)
	err = json.Unmarshal(bts, &books)
	if err != nil {
		return nil, err
	}
	return books, nil
}

// Add all books in the given direcotories.
func (c *CalibreDB) Add(paths []string) error {
	cmd := []string{"add"}
	cmd = append(cmd, paths...)
	return exec.Command("calibredb", cmd...).Run()
}

// CalibreBook sorter.

type By func(cb1, cb2 *CalibreBook) bool

func (by By) Sort(books []*CalibreBook) {
	bs := &bookSorter{
		books: books,
		by:    by,
	}
	sort.Sort(bs)
}

type bookSorter struct {
	books []*CalibreBook
	by    By
}

func (bs *bookSorter) Len() int {
	return len(bs.books)
}

func (bs *bookSorter) Swap(i, j int) {
	bs.books[i], bs.books[j] = bs.books[j], bs.books[i]
}

func (bs *bookSorter) Less(i, j int) bool {
	return bs.by(bs.books[i], bs.books[j])
}
