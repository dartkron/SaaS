// Package to get queue of WEMB files from imageboard
package board

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// type to represent cache of board state with locks, to prevent issues with concurent access to map
type boardMap struct {
	sync.RWMutex
	threads map[string]map[string]string
}

//type represent information about single webm file
type FileInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Thread string `json:"thread"`
	Post   string `json:"post"`
}

// Type to parse JSON-view of imageboard page
type boardPage struct {
	Threads []struct {
		Thread_num string
		Posts      []struct {
			Comment string
			Files   []struct {
				Path string
				Name string
			}
			Num int
		}
	}
}

// Type to parse JSON-view of Main imageboard page. There is Number os posts in string
// instead of interer as on other pages. That's really not what i expected
type boardMainPage struct {
	Threads []struct {
		Thread_num string
		Posts      []struct {
			Comment string
			Files   []struct {
				Path string
				Name string
			}
			Num string
		}
	}
}

// Type to represent our view of imageboard state.
type Board struct {

	// Struct to save the configuration
	config struct {
		JSONUrl          string
		DownloadURL      string
		BrowserUserAgent string
		Cookie           string
	}

	// Regexp to check if thread is a WEBM-thread
	threadWebmRegexp *regexp.Regexp

	// Regexp to check if file is webm
	filenameWebmRegexp *regexp.Regexp

	// Map to cache threads and WEBM files status, to know when new ones added
	cache boardMap

	// Watch queue
	Queue []FileInfo
}

// Generates new board instance and fill default values.
func NewBoard(JSONUrl, DownloadURL, BrowserUserAgent, Cookie string) (*Board, error) {
	board := new(Board)
	board.config.JSONUrl = JSONUrl
	board.config.DownloadURL = DownloadURL
	board.config.BrowserUserAgent = BrowserUserAgent
	board.config.Cookie = Cookie

	// If we didn't recieve cookies, get it from site
	if board.config.Cookie == "" {
		err := board.getCFCookie(board.config.DownloadURL)
		if err != nil {
			log.Println("Error cookie request: ", err)
		}
	}

	//Precompile regexp's for parsing threads content
	board.threadWebmRegexp = regexp.MustCompile("([Ww][Ee][Bb].*[Mm])|([Цц][Уу][ИЙйи].*[Ьь])")
	board.filenameWebmRegexp = regexp.MustCompile(".webm$")

	//Struct to store target threads view
	board.cache.threads = make(map[string]map[string]string)

	return board, nil
}

// Function get fresh CloudFlare cookie and set it to config variable atribure.
func (b *Board) getCFCookie(URL string) error {
	client := &http.Client{}

	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", b.config.BrowserUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	b.config.Cookie = resp.Cookies()[0].Value
	return nil

}

// function make GET request with UserAgent and CloudFlare cookie from config variable.
func (b *Board) getUrl(URL string) (response []byte, err error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", b.config.BrowserUserAgent)
	req.AddCookie(&http.Cookie{Name: "__cfduid", Value: b.config.Cookie})

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return body, errors.New("Response status is " + resp.Status)
	}
	return body, nil
}

// Function to remove thread from threads map with RW lock.
func (b *Board) addThread(num string) error {
	b.cache.RLock()
	b.cache.threads[num] = make(map[string]string)
	b.cache.RUnlock()
	return nil
}

// Functuon delete dead thread from board state cache.
func (b *Board) deleteThread(num string) error {
	b.cache.RLock()
	delete(b.cache.threads, num)
	b.cache.RUnlock()
	return nil
}

// Function to check if thread already exist in cache or not.
func (b *Board) isThread(num string) bool {
	b.cache.RLock()
	if _, ok := b.cache.threads[num]; !ok {
		b.cache.RUnlock()
		return false
	}
	b.cache.RUnlock()
	return true
}

// Fuction return current board state cache to iterate over.
func (b *Board) getThreadsMap() map[string]map[string]string {
	b.cache.RLock()
	defer b.cache.RUnlock()
	return b.cache.threads
}

// Function to check if file already added to cache or not.
func (b *Board) isFile(thread, name string) bool {
	b.cache.RLock()
	if b.cache.threads[thread][name] == "" {
		return false
	}
	return true
}

// Function to add file to board cache and queue.
func (b *Board) addFile(thread string, post int, name, path string) error {
	b.cache.RLock()
	b.cache.threads[thread][name] = path
	b.Queue = append(b.Queue, FileInfo{name, path, thread, strconv.Itoa(post)})

	return nil
}

// Function to check board for a new WEBM threads and save them to cache.
func (b *Board) scan4Treads() error {
	var page boardMainPage
	log.Println("Inititated scan 0 page for a new WEBM threads")
	response, err := b.getUrl(b.config.JSONUrl)
	if err != nil {
		return err
	}
	err = json.Unmarshal(response, &page)
	if err != nil {
		return err
	}
	for _, thread := range page.Threads {
		for _, file := range thread.Posts[0].Files {
			if b.filenameWebmRegexp.MatchString(file.Path) && b.threadWebmRegexp.MatchString(thread.Posts[0].Comment) {
				if !b.isThread(thread.Thread_num) {
					log.Println("Found new WEBM thread ", thread.Thread_num)
					b.addThread(thread.Thread_num)
				}
			}
		}

	}

	return nil
}

// Function to check all threads from cache if they have new webm files.
func (b *Board) updateThreadsPosts() error {
	var page boardPage
	for thread_num, _ := range b.getThreadsMap() {
		response, err := b.getUrl(b.config.DownloadURL + "res/" + thread_num + ".json")
		if err != nil {
			b.deleteThread(thread_num)
		} else {
			err = json.Unmarshal(response, &page)
			if err != nil {
				return err
			}
			for _, files := range page.Threads[0].Posts {
				for _, file := range files.Files {
					if b.filenameWebmRegexp.MatchString(file.Name) && !b.isFile(thread_num, file.Name) {

						log.Println("Adding new file ", file.Name, " from thread ", thread_num, " to queue")
						err := b.addFile(thread_num, files.Num, file.Name, file.Path)
						if err != nil {
							log.Println("Error on adding file ", file.Name, " ", file.Path, " Text: ", err)
						}
					}
				}
			}

		}
	}

	return nil
}

// Function to refresh board state and update cache. Exported in case want do it manually.
func (b *Board) Refresh() error {
	err := b.scan4Treads()
	if err != nil {
		return err
	}

	err = b.updateThreadsPosts()
	if err != nil {
		return err
	}
	return nil
}

// Function to continiously watch updates.
func (b *Board) watcher() {
	for {
		err := b.Refresh()
		if err != nil {
			log.Fatalln("Error refreshing board: ", err)
		}
		time.Sleep(time.Duration(120+rand.Int31n(240)) * time.Second)
	}
}

// Starts automatic board cache updates every 2-6 minutes.
func (b *Board) AutoWatcher() {
	go b.watcher()
}
