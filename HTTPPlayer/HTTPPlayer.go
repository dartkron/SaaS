// First frontend of board package. Handle functionality of streaming, caching and of webm
package HTTPPlayer

import (
	"SaaS/board"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// ID for session cookie
const session_cookie = "sosach_session_id"

/* Type represents player. Handle downloading, caching, queue navigation, laos own Board instance
 */
type HTTPPlayer struct {
	Config struct {
		SaveDirectory    string
		Cookie           string
		BrowserUserAgent string
		DownloadURL      string
		Port             string
		Tempdir          string
	}
	sessionsControl struct {
		sync.RWMutex
		sessions map[string]sessionType
	}

	cachedFiles struct {
		sync.RWMutex
		files map[string]string
	}
	sosach          *board.Board
	defaultPosition int
}

type sessionType struct {
	position int
	created  time.Time
}

// Function to create Player and fill all necessary fields
func NewHTTPPlayer(SaveDirectory, Cookie, BrowserUserAgent, DownloadURL, JSONUrl, Port string) (*HTTPPlayer, error) {
	player := new(HTTPPlayer)

	// init sessions map
	player.sessionsControl.RLock()
	player.sessionsControl.sessions = make(map[string]sessionType)
	player.sessionsControl.RUnlock()

	// Pre-creation of tmp dir, due a bug in Golang https://github.com/golang/go/issues/6842
	tempdir, err := ioutil.TempDir("", "sosach_webm")
	if err != nil {
		return player, err
	}

	// Initial configutation
	player.Config.SaveDirectory = SaveDirectory
	player.Config.Cookie = Cookie
	player.Config.BrowserUserAgent = BrowserUserAgent
	player.Config.DownloadURL = DownloadURL
	player.Config.Port = Port
	player.Config.Tempdir = tempdir

	// temporary queue for debug
	//player.sosach.Queue = []board.FileInfo{{"14450448066140.webm", "src/104033532/14450448066140.webm"}, {"14450448067471.webm", "src/104033532/14450448067471.webm"}, {"14450448629300.webm", "src/104033532/14450448629300.webm"}}

	// Init ImageBoard watcher
	sosach, err := board.NewBoard(JSONUrl, DownloadURL, BrowserUserAgent, Cookie)
	if err != nil {
		log.Fatalln("Error inititating new board instance: ", err)
	}
	player.sosach = sosach

	player.sosach.AutoWatcher()

	// Check if cache directory exist and create it not. If indicated path is file istead of directory, show alert and stop
	dir, err := os.Stat(player.Config.SaveDirectory)
	if err != nil {
		log.Println("Could not stat directory for saving webm. Trying create a new one.")
		err := os.Mkdir(player.Config.SaveDirectory, 0755)
		if err != nil {
			return player, err
		} else {
			log.Println("Creating new directory ", player.Config.SaveDirectory)
		}
	} else {
		if !dir.IsDir() {
			return player, errors.New("Indicated SaveDirectory is a file! Unfortunatenly, this is critical error.")
		} else {
			player.refrestFileCache()
		}
	}

	return player, nil
}

// Function handlig new seession creation: lock, random, etc
func (p *HTTPPlayer) newSession(resp *http.ResponseWriter) (string, error) {
	var sessionID string
	for {
		sessionID = strconv.Itoa(int(time.Now().Unix())) + strconv.Itoa(int(rand.Int63()))

		p.sessionsControl.RLock()
		defer p.sessionsControl.RUnlock()
		if _, ok := p.sessionsControl.sessions[sessionID]; !ok {
			p.sessionsControl.sessions[sessionID] = sessionType{p.defaultPosition, time.Now()}
			break
		}
	}

	http.SetCookie(*resp, &http.Cookie{Name: session_cookie, Value: sessionID, Expires: time.Now().Add(24 * time.Hour)})

	return sessionID, nil
}

// Function to get session via sessionID
func (p *HTTPPlayer) getSession(sessionID string) (sessionType, error) {
	p.sessionsControl.RLock()
	defer p.sessionsControl.RUnlock()
	if session, ok := p.sessionsControl.sessions[sessionID]; !ok {
		return session, errors.New("No such session")
	} else {
		return session, nil
	}
}

// function to handle session work: check, create if neccessary, set proper headers, etc
func (p *HTTPPlayer) getRequestSession(resp *http.ResponseWriter, req *http.Request) (string, error) {
	var sessionID string
	cookie, err := req.Cookie(session_cookie)
	if err != nil {
		log.Println("Serving new user without cookie. Generating new one.")
		sessionID, err := p.newSession(resp)
		if err != nil {
			return sessionID, err
		}

	} else {
		sessionID = cookie.Value
		_, err := p.getSession(sessionID)
		if err != nil {
			sessionID, err := p.newSession(resp)
			if err != nil {
				return sessionID, err
			}
		}
	}
	return sessionID, nil
}

// Function to re-read cache directory and rebuild cachedFiles map
func (p *HTTPPlayer) refrestFileCache() {
	cacheDir, err := ioutil.ReadDir(p.Config.SaveDirectory)
	if err != nil {
		log.Println("Error on read directory for save files: ", err)
	}
	p.cachedFiles.RLock()https://github.com/ValdikSS/endless-sosuch
	p.cachedFiles.files = make(map[string]string)
	for _, file := range cacheDir {
		p.cachedFiles.files[file.Name()] = p.Config.SaveDirectory + string(os.PathSeparator) + file.Name()
	}

	p.cachedFiles.RUnlock()
}

// Function to check if target file already in cache and return path, otherwise return empty string
func (p *HTTPPlayer) getFileFromCache(file string) string {
	p.cachedFiles.RLock()
	defer p.cachedFiles.RUnlock()

	if path, ok := p.cachedFiles.files[file]; ok {
		return path
	} else {
		return ""
	}
}

// Function to add file to cache map
func (p *HTTPPlayer) addFileToCache(name, path string) {
	p.cachedFiles.RLock()
	defer p.cachedFiles.RUnlock()
	p.cachedFiles.files[name] = path
}

// Function to handle move position on queue
func (p *HTTPPlayer) sessionMovePos(sessionID string, move int) int {
	length := len(p.sosach.Queue)
	p.sessionsControl.RLock()
	defer p.sessionsControl.RUnlock()
	if p.sessionsControl.sessions[sessionID].position+move < 0 || p.sessionsControl.sessions[sessionID].position+move > length-1 {
		return rand.Intn(length)
	} else {
		session := p.sessionsControl.sessions[sessionID]
		session.position += move
		p.sessionsControl.sessions[sessionID] = session
		return p.sessionsControl.sessions[sessionID].position
	}
}

func (p *HTTPPlayer) getWebmInfo(resp http.ResponseWriter, req *http.Request) {
	sessionID, err := p.getRequestSession(&resp, req)
	log.Println("Request info with sessionID ", sessionID)
	if err != nil {
		log.Println("Error while handling session: ", err)
	}
	position := p.sessionMovePos(sessionID, 0)
	log.Println("sessionID ", sessionID, " have position ", position, ". Queue for this position is ", p.sosach.Queue[position])
	fileInfo, err := json.Marshal(p.sosach.Queue[position])
	if err != nil {
		log.Println("Error while marshaling file info: ", err)
		return
	}

	resp.Header().Add("Content-Type", "application/json")

	resp.Write(fileInfo)
}

func (p *HTTPPlayer) servePlay(resp *http.ResponseWriter, sessionID string, move int) error {

	position := p.sessionMovePos(sessionID, move)

	filePath := p.getFileFromCache(p.sosach.Queue[position].Name)

	if filePath == "" {

		client := &http.Client{}
		log.Println(p.Config.DownloadURL+p.sosach.Queue[position].Name, " not in cache, making following request: ", p.Config.DownloadURL+p.sosach.Queue[position].Path)

		outReq, err := http.NewRequest("GET", p.Config.DownloadURL+p.sosach.Queue[position].Path, nil)
		if err != nil {
			log.Println("Error on creating outgoing request ", err)
			log.Println("Removing ", p.sosach.Queue[position].Name, " from queue")
			p.sosach.Queue = append(p.sosach.Queue[:position], p.sosach.Queue[position+1:]...)
		}

		outReq.Header.Set("User-Agent", p.Config.BrowserUserAgent)
		outReq.AddCookie(&http.Cookie{Name: "__cfduid", Value: p.Config.Cookie})

		//file, err := os.OpenFile(p.Config.SaveDirectory+string(os.PathSeparator)+p.sosach.Queue[0].Name, os.O_WRONLY|os.O_CREATE, 0644)
		file, err := ioutil.TempFile(p.Config.Tempdir, "webmcache")
		if err != nil {
			log.Println("Error on creating temporary file: ", err)
		}
		defer file.Close()

		log.Println("Created temporary file ", file.Name())

		outerResp, err := client.Do(outReq)
		if err != nil {
			log.Println("Error on doing outgoing reqeuest: ", err)
		}

		defer outerResp.Body.Close()

		multiWrite := io.MultiWriter(file, *resp)

		bytesCount, err := io.Copy(multiWrite, outerResp.Body)
		if err != nil {
			log.Println("Error while downloading/uploading: ", err)
			os.Remove(file.Name())
		} else {
			err = os.Rename(file.Name(), p.Config.SaveDirectory+string(os.PathSeparator)+p.sosach.Queue[position].Name)
			if err != nil {
				log.Println("Error on saving file to cache: ", err)
			} else {
				p.addFileToCache(p.sosach.Queue[position].Name, p.Config.SaveDirectory+string(os.PathSeparator)+p.sosach.Queue[position].Name)
			}

		}

		log.Println(bytesCount, "bytes downloaded/uploaded.")
	} else {
		file, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
		if err != nil {
			log.Println("Error while opening cached file: ", err)

		}
		defer file.Close()
		bytesCount, err := io.Copy(*resp, file)
		if err != nil {
			log.Println("Error while uploading: ", err)
		}
		log.Println(bytesCount, "bytes uploaded.")
	}
	return nil
}

func (p *HTTPPlayer) Play(resp http.ResponseWriter, req *http.Request) {
	var move int

	sessionID, err := p.getRequestSession(&resp, req)
	if err != nil {
		log.Println("Error while handling session: ", err)
	}

	move = 0

	if req.URL.String() == "/play/next" {
		move = 1
	}

	if req.URL.String() == "/play/prev" {
		move = -1
	}

	if req.URL.String() == "/play/next10" {
		move = 10
	}

	if req.URL.String() == "/play/prev10" {
		move = -10
	}

	err = p.servePlay(&resp, sessionID, move)
	if err != nil {
		log.Println("error on playing webm ", err)
	}
}

func (p *HTTPPlayer) ListenAndServe() error {

	http.HandleFunc("/play/info", p.getWebmInfo)

	http.HandleFunc("/play/", p.Play)

	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Type", "text/html")
		resp.Header().Set("charset", "utf-8")
		pageContent := `
		<div id="header" style="width:100%;text-align: center;">
		<h1>SaaS - Sosach as a Service</h1>
		<h3>Endless webm flow from 2ch.hk in your browser</h3>
		<a class="github-ribbon" href="https://github.com/dartkron/SaaS"><img style="position: absolute; top: 0; left: 0; border: 0;" src="https://camo.githubusercontent.com/567c3a48d796e2fc06ea80409cc9dd82bf714434/68747470733a2f2f73332e616d617a6f6e6177732e636f6d2f6769746875622f726962626f6e732f666f726b6d655f6c6566745f6461726b626c75655f3132313632312e706e67" alt="Fork me on GitHub" data-canonical-src="https://s3.amazonaws.com/github/ribbons/forkme_left_darkblue_121621.png"></a>
		</div>
		<div id="player" style="width:100%; text-align: center;">
		    <video style="height:80%" id="video_player" controls autoplay src="play/"> Your user agent does not support the HTML5 Video element. </video><br/> 
			<input id="Prev10" type="button" value="Prev10(z)" onclick="playPrev10()" /> 
			<input id="Prev" type="button" value="Prev(x)" onclick="playPrev()" /> 
			<input id="Next" type="button" value="Next(b)" onclick="playNext()" /> 
			<input id="Skip10" type="button" value="Skip10(n)" onclick="playNext10()" />						
			<div id="hidden" style="display: none"></div>
			<div id="info"></div>
		</div>	 
			<script type='text/javascript'>
			
			function loadXMLDoc(target,url,callback) {
             var xmlhttp;
    
             callback = callback || function () {};
 
             if (window.XMLHttpRequest) {
               // code for IE7+, Firefox, Chrome, Opera, Safari
             xmlhttp = new XMLHttpRequest();
           } else {
             // code for IE6, IE5
            xmlhttp = new ActiveXObject("Microsoft.XMLHTTP");
            }

           xmlhttp.onreadystatechange = function() {
            if (xmlhttp.readyState == XMLHttpRequest.DONE ) {
              if(xmlhttp.status == 200){
                 document.getElementById(target).innerHTML = xmlhttp.responseText;
                  setTimeout(callback, 500);
                }
           else if(xmlhttp.status == 400) {
              alert('There was an error 400')
           }
           else {
               alert('Something went wrong.')
           }
           }
          }

           xmlhttp.open("GET", url, true);
           xmlhttp.send();
         }
		
		    function updateVideoInfo () {
				info = JSON.parse(document.getElementById("hidden").innerHTML);
				document.getElementById("info").innerHTML = "Link to original video: <a href=\"https://2ch.hk/b/"+info.path+"\">https://2ch.hk/b/"+info.path+"</a><br/>Link to original post: <a href=\"https://2ch.hk/b/res/"+info.thread+".html#"+info.post+"\">https://2ch.hk/b/res"+info.thread+".html#"+info.post+"</a>";
				}
			
			function playNext10() { document.getElementById('video_player').src='play/next10'; setTimeout('loadXMLDoc("hidden","/play/info",updateVideoInfo)',500);}
			function playNext() { document.getElementById('video_player').src='play/next'; setTimeout('loadXMLDoc("hidden","/play/info",updateVideoInfo)',500);}
			function playPrev() { document.getElementById('video_player').src='play/prev'; setTimeout('loadXMLDoc("hidden","/play/info",updateVideoInfo)',500);} 
			function playPrev10() { document.getElementById('video_player').src='play/prev10'; setTimeout('loadXMLDoc("hidden","/play/info",updateVideoInfo)',500);} 
			function playPause() { if (document.getElementById('video_player').paused) {document.getElementById('video_player').play();} else {document.getElementById('video_player').pause()}} 
			function myHandler(e) {  
			playNext() 
			} 
			function keyPressHandler(e) { 
			console.log(e.key); 
			if ((e.keyCode == 66) || (e.keyCode == 98) || (e.keyCode == 1048) || (e.keyCode == 1080) || (e.key == 'b') || (e.key == 'B') || (e.key == 'и') || (e.key == 'И')) { playNext(); } 
			if ((e.keyCode == 110) || (e.keyCode == 78) || (e.keyCode == 1058) || (e.keyCode == 1090)|| (e.key == 'n') || (e.key == 'N') || (e.key == 'т') || (e.key == 'Т')) { playNext10(); } 
			if ((e.keyCode == 88) || (e.keyCode == 120) || (e.keyCode == 1095) || (e.keyCode == 1063)|| (e.key == 'x') || (e.key == 'X') || (e.key == 'ч') || (e.key == 'Ч')) { playPrev(); } 
			if ((e.keyCode == 90) || (e.keyCode == 122) || (e.keyCode == 1071) || (e.keyCode == 1103)|| (e.key == 'z') || (e.key == 'Z') || (e.key == 'я') || (e.key == 'Я')) { playPrev10(); } 
			if ((e.keyCode == 99) || (e.keyCode == 67) || (e.keyCode == 1057) || (e.keyCode == 1089) || (e.keyCode == 32) || (e.key == 'c') || (e.key == 'C') || (e.key == 'с') || (e.key == 'С') || (e.key == ' ')) { playPause(); } 
			} 
			document.addEventListener('DOMContentLoaded', function () { 
			setTimeout(loadXMLDoc("hidden","/play/info",updateVideoInfo),1500)
			window.addEventListener('keypress', keyPressHandler, false); 
			document.getElementById('video_player').addEventListener('ended',myHandler,false); 
			}); 
			</script>`
		io.WriteString(resp, pageContent)
	})

	err := http.ListenAndServe(":"+p.Config.Port, nil)

	if err != nil {
		log.Fatal("Error on creating listener: ", err)
	}

	return nil
}
