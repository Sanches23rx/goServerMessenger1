/*package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Message struct {
	Text string `json:"text"`
}

func main() {
	messages := make([]Message, 0)

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://*"},
		AllowedMethods: []string{"GET", "POST"},
	}))

	r.Get("/api/echo", func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		w.Write(data)
	})
	r.Post("/api/messages", func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		fmt.Println("MSG SAVED:", string(data))
		msg := Message{}
		if err := json.Unmarshal(data, &msg); err != nil {
			fmt.Errorf("can't unmarshal: %s", err)
		}
		messages = append(messages, msg)
	})
	r.Get("/api/messages/{id}", func(w http.ResponseWriter, r *http.Request) {
		str_id := chi.URLParam(r, "id")
		id, err := strconv.Atoi(str_id)
		fmt.Println("MSG SENT:", str_id)
		if err != nil {
			log.Fatalf("can't parse id: %s", err)
		}
		if id >= 0 && id < len(messages) {
			msg := messages[id]
			data, _ := json.Marshal(msg)
			w.Write(data)
		}
	})

	fmt.Println("Ready")
	http.ListenAndServe(":5000", r)
} */
//-----------------------------------------------
package main

//sudo electron-packager ./ runit --platform=mas --arch=x64 --electron-version=1.4.3

import (
	"fmt"
	//"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/websocket"
)

var (
	dir               string
	clientRequests    = make(chan *NewCLientEvent, 100)
	clientDisconnects = make(chan string, 100)
	messages          = make(chan *Msg, 100)
)

type (
	Msg struct {
		clientKey string
		text      string
	}
	NewCLientEvent struct {
		clientKey string
		msgChan   chan *Msg
	}
)

const MAXBACKLOG = 100

func indexPage(w http.ResponseWriter, r *http.Request, filename string) {
	fp, err := os.Open(dir + "/" + filename)

	if err != nil {
		log.Fatal("No such dir", err)
	}
	_, err = io.Copy(w, fp)

	if err != nil {
		log.Println("Coulnot open filecontents")
	}
	defer fp.Close()
}

func EchoServer(ws *websocket.Conn) {

	var lenBuf = make([]byte, 5)
	msgChan := make(chan *Msg, 100)
	clientKey := ws.RemoteAddr().String()
	clientRequests <- &NewCLientEvent{clientKey, msgChan}
	defer func() { clientDisconnects <- clientKey }()

	go func() {
		for msg := range msgChan {
			ws.Write([]byte(msg.text))
		}
	}()

	for {
		_, err := ws.Read(lenBuf)
		if err != nil {
			log.Println("Error: ", err)
			return
		}
		length, _ := strconv.Atoi(strings.TrimSpace(string(lenBuf)))

		if length > 65536 {
			log.Println("Error: too big length: ", length)
			return
		}
		if length <= 0 {
			log.Println("Error: empty message")
			// return
		}
		var buf = make([]byte, length)
		_, err = ws.Read(buf)
		if err != nil {
			log.Println("Error: could nor read msg")
			return
		}

		messages <- &Msg{clientKey, string(buf)}

	}

}

// func EchoServer_bot(ws *websocket.Conn) {

// 	var lenBuf = make([]byte, 5)
// 	msgChan := make(chan *Msg, 100)
// 	clientKey := ws.RemoteAddr().String()
// 	clientRequests <- &NewCLientEvent{clientKey, msgChan}
// 	defer func() { clientDisconnects <- clientKey }()

// 	go func() {
// 		for msg := range msgChan {
// 			ws.Write([]byte(msg.text))
// 		}
// 	}()

// 	for {
// 		_, err := ws.Read(lenBuf)
// 		if err != nil {
// 			log.Println("Error: ", err)
// 			return
// 		}
// 		length, _ := strconv.Atoi(strings.TrimSpace(string(lenBuf)))

// 		if length > 65536 {
// 			log.Println("Error: too big length: ", length)
// 			return
// 		}
// 		if length <= 0 {
// 			log.Println("Error: empty message")
// 			// return
// 		}
// 		var buf = make([]byte, length)
// 		_, err = ws.Read(buf)
// 		if err != nil {
// 			log.Println("Error: could nor read msg")
// 			return
// 		}

// 		messages <- &Msg{clientKey, string(buf)}

// 	}

// }

func router() {
	clients := make(map[string]chan *Msg)

	for {
		select {
		case req := <-clientRequests:
			clients[req.clientKey] = req.msgChan
			log.Println("Connection client: ", req.clientKey)
		case clientKey := <-clientDisconnects:
			close(clients[clientKey])
			delete(clients, clientKey)
			log.Println("Disconnection client: ", clientKey)
		case msg := <-messages:
			log.Print(msg.clientKey + ": " + msg.text + "\n")
			for _, msgChan := range clients {
				if len(msgChan) < cap(msgChan) {
					msgChan <- msg
				}
			}

		}

	}
}

func main() {
	// dir = os.Args[1]
	dir = "src"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		indexPage(w, r, "index-1.html")
	})
	http.HandleFunc("/index.js", func(w http.ResponseWriter, r *http.Request) {
		indexPage(w, r, "index.js")
	})
	http.HandleFunc("/bot", func(w http.ResponseWriter, r *http.Request) {
		indexPage(w, r, "/bot/bot-page.html")
	})
	http.Handle("/ws", websocket.Handler(EchoServer))

	go router()

	fmt.Println("Server running...")
	http.ListenAndServe(":8080", nil)
}
