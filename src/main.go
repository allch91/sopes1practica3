package main

import (
	"net/http"

	"flag"
	"log"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/jandre/procfs"
	"github.com/shirou/gopsutil/cpu"
	"golang.org/x/net/websocket"
)

var cookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

func getUserName(request *http.Request) (userName string) {
	if cookie, err := request.Cookie("session"); err == nil {
		cookieValue := make(map[string]string)
		if err = cookieHandler.Decode("session", cookie.Value, &cookieValue); err == nil {
			userName = cookieValue["name"]
		}
	}
	return userName
}

func setSession(userName string, response http.ResponseWriter) {
	value := map[string]string{
		"name": userName,
	}
	if encoded, err := cookieHandler.Encode("session", value); err == nil {
		cookie := &http.Cookie{
			Name:  "session",
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(response, cookie)
	}
}

func clearSession(response http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(response, cookie)
}

func loginHandler(response http.ResponseWriter, request *http.Request) {
	name := request.FormValue("name")
	pass := request.FormValue("password")
	redirectTarget := "/"
	if name != "" && pass != "" {
		if name == "admin" && pass == "admin" {
			setSession(name, response)
			redirectTarget = "/panel"
		}
	}
	http.Redirect(response, request, redirectTarget, 302)
}

func logoutHandler(response http.ResponseWriter, request *http.Request) {
	clearSession(response)
	http.Redirect(response, request, "/", 302)
}

func indexPageHandler(response http.ResponseWriter, request *http.Request) {
	log.Println("request for \"login.html\" received")
	http.ServeFile(response, request, "./login.html")
}

func internalPageHandler(response http.ResponseWriter, request *http.Request) {
	userName := getUserName(request)
	if userName != "" {
		http.ServeFile(response, request, "./panel.html")
	} else {
		http.Redirect(response, request, "/", 302)
	}
}

var router = mux.NewRouter()

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	flag.Parse()
	router.HandleFunc("/", indexPageHandler)
	http.Handle("/static/", http.FileServer(http.Dir(".")))
	router.HandleFunc("/panel", internalPageHandler)

	router.HandleFunc("/login", loginHandler).Methods("POST")
	router.HandleFunc("/logout", logoutHandler).Methods("POST")

	http.Handle("/", router)

	http.Handle("/websocket", websocket.Handler(websockethandler))
	http.Handle("/websocket1", websocket.Handler(websockethandler1))

	log.Println("starting web server at", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatalln("http.ListenAndServe:", err)
	}
}

func websockethandler(c *websocket.Conn) {

	log.Println("request for \"websocket\" received")

	go func() {

		defer c.Close()

		var message string
		for {
			err := websocket.Message.Receive(c, &message)

			if err != nil {
				log.Println("websocket.Message.Receive:", err)
				return
			}
			log.Println("websocket no error received", message)

		}

	}()

	var timestamp int64 = 0
	var val int64 = 0
	var memtotal int64 = 0
	var memfree int64 = 0
	var memused float64 = 0

	for {

		meminfo, err := procfs.ParseMeminfo("/proc/meminfo")
		if err != nil {
			log.Println("Error al leer meminfo", err)
			return
		}

		if meminfo == nil {
			log.Println("Meminfo is missing")
			return
		}
		memtotal = meminfo.MemTotal / 1024
		memfree = meminfo.MemFree / 1024
		memused = float64(meminfo.MemTotal-meminfo.MemFree-meminfo.Cached-meminfo.Slab-meminfo.Buffers) / float64(meminfo.MemTotal)
		val = int64(100 * memused)
		err = websocket.JSON.Send(c, [4]int64{timestamp, val, memtotal, memfree})
		if err != nil {
			log.Println("encoder.Encode", err)
			return
		}

		timestamp++
		time.Sleep(1000 * time.Millisecond)
	}

}

func websockethandler1(c *websocket.Conn) {

	log.Println("request for \"websocket1\" received")

	go func() {

		defer c.Close()

		var message string
		for {
			err := websocket.Message.Receive(c, &message)

			if err != nil {
				log.Println("websocket1.Message.Receive:", err)
				return
			}
			log.Println("websocket1 no error received", message)

		}

	}()

	var timestamp int64 = 0
	var val int64 = 0
	var perc []float64
	for {
		perc, _ = cpu.Percent(0, false)
		val = int64(perc[0])
		err := websocket.JSON.Send(c, [2]int64{timestamp, val})
		if err != nil {
			log.Println("encoder1.Encode", err)
			return
		}

		timestamp++
		time.Sleep(500 * time.Millisecond)
	}

}
