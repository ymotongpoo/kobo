package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	xmlpath "gopkg.in/xmlpath.v2"
)

const (
	NewContentsBaseURL = `http://www.netriver.jp/rbs/usr/himanayaro/`
	NewContentsURL     = NewContentsBaseURL + `rivbb.cgi`
	OldContentsURL     = `http://www.netriver.jp/rbs/usr/umoo/rivbb.cgi`
	OldArchiveURL      = `http://baka.bakufu.org/kobokora/mee/index.html`
	CustomUserAgent    = `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.95 Safari/537.36`

	QueueSize = 100
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

// NewContentsCrawler starts crawling on new contents pages.
func NewContentsCrawler(wg *sync.WaitGroup) {
	dir := filepath.Join(os.TempDir(), "new")
	_ = dir
	queue := make(chan string, QueueSize)
	errCh := make(chan error, QueueSize)
	go func() {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go NewContentsPageCrawler(&wg, i, queue, errCh)
		}
		wg.Wait()
		close(queue)
	}()

LOOP:
	for {
		select {
		case s, ok := <-queue:
			if !ok {
				break LOOP
			}
			p := path.Join(NewContentsBaseURL, s)
			log.Println(p)
		case err := <-errCh:
			log.Println(err)
		}
	}
	wg.Done()
}

// OldContentsCrawler starts crawling on old contents pages.
func OldContentsCrawler(wg *sync.WaitGroup) {
	wg.Done()
}

// OldArchiveCrawler starts crawling on archive page.
func OldArchiveCrawler(wg *sync.WaitGroup) {
	wg.Done()
}

// NewContentsPageCrawler extracts direct image file path in a page.
func NewContentsPageCrawler(wg *sync.WaitGroup, page int, queue chan string, errCh chan error) {
	defer wg.Done()
	value := url.Values{}
	value.Add("page", strconv.Itoa(page))
	urlStr := NewContentsURL + "?" + value.Encode()

	resp, err := CustomGet(urlStr)
	if err != nil {
		errCh <- err
	}
	defer resp.Body.Close()

	img := xmlpath.MustCompile(`//tbody//a/@href`)
	root, err := xmlpath.ParseHTML(resp.Body)
	if err != nil {
		errCh <- err
	}
	iter := img.Iter(root)
	for iter.Next() {
		n := iter.Node()
		src := n.String()
		if strings.HasSuffix(src, "png") || strings.HasSuffix(src, "jpg") {
			queue <- n.String()
		}
	}
}

// CustomGet replace default User-Agent header with custom one and call GET method.
func CustomGet(urlStr string) (*http.Response, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", CustomUserAgent)
	client := &http.Client{}
	return client.Do(req)
}
