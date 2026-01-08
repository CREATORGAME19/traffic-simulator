package simulation

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

const PORT = "8080"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func runFrontend(wg *sync.WaitGroup, channel chan []VehicleLocation) {
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
			return
		}

		defer func() { //Cleanup when websocket connection is lost
			conn.Close()
			wg.Done()
		}()

		go func() { //Reader loop (Websocket)
			for {
				if _,_,err := conn.ReadMessage(); err != nil {
					fmt.Println("read error:", err)
					conn.Close()
					return
				}
			}
		}()

		for data := range channel { //Reader loop for controller channel
			var msg string
			for i := 0; i < MAX_VEHICLES; i++ {
				msg += fmt.Sprintln("Vehicle ID:", i, ", Time:", data[i].time, ", X:", data[i].x, ", Y:", data[i].y)
			}

			// Print the message to the console
			fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))

			// Write message back to browser
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				fmt.Println("write error:", err)
				return
			}
			wg.Done()
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})

	http.ListenAndServe(":"+PORT, nil)
}
