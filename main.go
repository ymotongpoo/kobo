package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	xmlpath "gopkg.in/xmlpath.v2"
)

const (
	NewContentsURL = `http://www.netriver.jp/rbs/usr/himanayaro/rivbb.cgi`
	OldContentsURL = `http://www.netriver.jp/rbs/usr/umoo/rivbb.cgi`
	OldArchiveURL  = `http://baka.bakufu.org/kobokora/mee/index.html`
)

func main() {
	log.Println("*** start crawling")
	var wg sync.WaitGroup
	wg.Add(3)
	go NewContentsCrawler(&wg)
	go OldContentsCrawler(&wg)
	go OldArchiveCrawler(&wg)
	wg.Wait()
}

func NewContentsCrawler(wg *sync.WaitGroup) {
	dir := filepath.Join(os.TempDir(), "new")
	_ = dir
	queue := make(chan string, 100)
	go func() {
		for i := 0; i < 10; i++ {
			NewContentsPageCrawler(i, queue)
		}
		close(queue)
	}()

	for s := range queue {
		log.Println(s)
	}
	wg.Done()
}

func OldContentsCrawler(wg *sync.WaitGroup) {
	wg.Done()
}

func OldArchiveCrawler(wg *sync.WaitGroup) {
	wg.Done()
}

// NewContentsPageCrawler extracts direct image file path in a page.
func NewContentsPageCrawler(page int, queue chan string) error {
	value := url.Values{}
	value.Add("page", strconv.Itoa(page))
	urlStr := NewContentsURL + "?" + value.Encode()

	resp, err := http.Get(urlStr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	img := xmlpath.MustCompile(`//tbody//a/@href`)
	root, err := xmlpath.ParseHTML(resp.Body)
	if err != nil {
		return err
	}
	iter := img.Iter(root)
	for iter.Next() {
		n := iter.Node()
		src := n.String()
		if strings.HasSuffix(src, "png") || strings.HasSuffix(src, "jpg") {
			queue <- n.String()
		}
	}
	return nil
}
