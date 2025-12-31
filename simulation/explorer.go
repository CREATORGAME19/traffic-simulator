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
		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity

		for {
			for data := range channel {
				msg := ""
				for i := 0; i < MAX_VEHICLES; i++ {
					msg += fmt.Sprintln("Vehicle ID:", i, ", Time:", data[i].time, ", X:", data[i].x, ", Y:", data[i].y)
				}

				// Print the message to the console
				fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))

				// Write message back to browser
				if err := conn.WriteMessage(1, []byte(msg)); err != nil {
					return
				}
				wg.Done()
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})

	http.ListenAndServe(":"+PORT, nil)
}
