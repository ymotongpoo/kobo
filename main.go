package main

import (
	"fmt"
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
	// NewContentsBaseURL is the base URL of image files.
	NewContentsBaseURL = `http://www.netriver.jp/rbs/usr/himanayaro/`

	// NewContentsURL is the URL of the BBS itself where new contents would be uplaoded.
	NewContentsURL = NewContentsBaseURL + `rivbb.cgi`

	// OldContentsBaseURL is the base URL of image files.
	OldContentsBaseURL = `http://www.netriver.jp/rbs/usr/umoo/`

	// OldContentsURL is the URL of the BBS where old popular contents would be uploaded.
	OldContentsURL = OldContentsBaseURL + `rivbb.cgi`

	// OldArchiveURL is the URL of the page where the archive is.
	OldArchiveURL = `http://baka.bakufu.org/kobokora/mee/index.html`

	// CustomUserAgent is based on Chrome 39 as of Jan 3, 2015.
	CustomUserAgent = `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.95 Safari/537.36`

	// QueueSize is commonly used in this program to limit channel capacity.
	QueueSize = 100

	// MaxNewContentsPageNum is due to the BBS' spec.
	MaxNewContentsPageNum = 8

	// MaxOldContentsPageNum is due to the BBS' spec.
	MaxOldContentsPageNum = 3
)

func main() {
	log.Println("*** start crawling")
	var wg sync.WaitGroup
	wg.Add(3)
	go NewContentsCrawler(&wg)
	go OldContentsCrawler(&wg)
	go OldArchiveCrawler(&wg)
	wg.Wait()
	log.Println("*** finished crawling")
}

// NewContentsCrawler starts crawling on contents pages.
func contentsCrawler(wg *sync.WaitGroup, maxPage int, baseURL string) {
	dir := filepath.Join(os.TempDir(), "new")
	_ = dir
	queue := make(chan string, QueueSize)
	errCh := make(chan error, QueueSize)
	go func() {
		var wg sync.WaitGroup
		for i := 0; i < MaxNewContentsPageNum; i++ {
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
			p := path.Join(baseURL, s)
			log.Println(p)
		case err := <-errCh:
			log.Println(err)
		}
	}
	wg.Done()
}

// NewContentsCrawler starts crawling on new contents pages.
func NewContentsCrawler(wg *sync.WaitGroup) {
	contentsCrawler(wg, MaxNewContentsPageNum, NewContentsBaseURL)
}

// OldContentsCrawler starts crawling on new contents pages.
func OldContentsCrawler(wg *sync.WaitGroup) {
	contentsCrawler(wg, MaxOldContentsPageNum, OldContentsBaseURL)
}

// OldArchiveCrawler starts crawling on archive page.
func OldArchiveCrawler(wg *sync.WaitGroup) {
	pageCh := make(chan string)
	contentCh := make(chan string)
	go archivePageProceeder(OldArchiveURL, pageCh)
	go func(pageCh chan string) {
		for p := range pageCh {
			go OldArchivePageCrawler(p, contentCh)
		}
	}(pageCh)
	wg.Done()
}

// OldArchivePageCrawler extracs actual page URL in page and send it to contentCh.
func OldArchivePageCrawler(page string, contentCh chan string) {
	return
}

// archivePageProceeder finds next page URL starting from start and send it to pageCh.
func archivePageProceeder(start string, pageCh chan string) {
	return
}

func contentsPageCrawler(wg *sync.WaitGroup, page int, bbsURL string, xpath string, queue chan string, errCh chan error) {
	defer wg.Done()
	value := url.Values{}
	value.Add("page", strconv.Itoa(page+1))
	urlStr := bbsURL + "?" + value.Encode()

	resp, err := CustomGet(urlStr)
	if err != nil {
		errCh <- err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errCh <- fmt.Errorf("HTTP status error on page %v: %v", page, resp.Status)
		return
	}

	img := xmlpath.MustCompile(xpath)
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

// NewContentsPageCrawler extracts direct image file path in a page.
func NewContentsPageCrawler(wg *sync.WaitGroup, page int, queue chan string, errCh chan error) {
	contentsPageCrawler(wg, page, NewContentsURL, `//tbody//a/@href`, queue, errCh)
}

// OldContentsPageCrawler extracts direct image file path in a page.
func OldContentsPageCrawler(wg *sync.WaitGroup, page int, queue chan string, errCh chan error) {
	contentsPageCrawler(wg, page, OldContentsURL, `//tbody//a/@href`, queue, errCh)
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
