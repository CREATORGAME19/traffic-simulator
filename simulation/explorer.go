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

var reader sync.WaitGroup

func runFrontend(wg *sync.WaitGroup, channel chan VehicleInfoDatagram, mapsim *Map, runs int) {
	//TODO: Use map data to construct initial frontend map
	vehicle_fetch_history := make([][]VehicleInfoDatagram,runs)
	for i:=0;i<runs;i++ {
		vehicle_fetch_history[i] = make([]VehicleInfoDatagram, MAX_VEHICLES)
	}

	current_run := 0
	vehicles_seen := 0 //Vehicles seen in the run so far

	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
			return
		}

		defer func() {
			reader.Done()
			conn.Close()
		}()

		reader.Wait()
		reader.Add(1) //Only one reader allowed at once

		run := 0
		for current_run-run > 0 { //Resend all lost packets in order if page connection is lost.
			var msg string
			for i := 0; i < MAX_VEHICLES; i++ {
				msg += fmt.Sprintln("Vehicle ID:", i, ", Time:", vehicle_fetch_history[run][i].time, ", X:",  vehicle_fetch_history[run][i].x, ", Y:",  vehicle_fetch_history[run][i].y)
			}

			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				fmt.Println("write error:", err)
				return
			}
			run++
		}

		//go func() { //Reader loop(Websocket)
		//	for {
		//		if _,_,err := conn.ReadMessage(); err != nil {
		//			fmt.Println("read error:", err)
		//			return
		//		}
		//	}
		//}()
		

		for data := range channel { //Reader loop for controller channel
			vehicle_fetch_history[current_run][data.id] = data
			vehicles_seen++
			wg.Done()

			if vehicles_seen >= MAX_VEHICLES {
				var msg string
				for i := 0; i < MAX_VEHICLES; i++ {
					msg += fmt.Sprintln("Vehicle ID:", i, ", Time:", vehicle_fetch_history[current_run][i].time, ", X:",  vehicle_fetch_history[current_run][i].x, ", Y:",  vehicle_fetch_history[current_run][i].y)
				}
				
				vehicles_seen -= MAX_VEHICLES
				current_run++

				// Print the message to the console
				fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))

				// Write message back to browser
				if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
					fmt.Println("write error:", err)
					return
				}
			}
		}


	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})

	http.ListenAndServe(":"+PORT, nil)
}

