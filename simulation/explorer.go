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

type Message struct {
	Type string
	Data interface{}
}

type VehicleMessage struct {
	Vehicle_ID VehicleID
	X, Y float64
	Status VehicleStatus
	Time SimTime
}

type MapSetupMessage struct {
	Nodes []MapNodeSetupMessage
	Lanes []MapLaneSetupMessage
	ConfigParameters ConfigParameters
}

type MapNodeSetupMessage struct {
	Node_ID RoadNodeID
	X, Y float64
	AgentType StaticAgentType
}

type MapLaneSetupMessage struct {
	Lane_ID LaneID
	Start_X, Start_Y, End_X, End_Y float64
}

var reader sync.WaitGroup

func runFrontend(wg *sync.WaitGroup, channel chan VehicleInfoDatagram, mapsim *Map) {
	vehicle_fetch_history := make([][]VehicleInfoDatagram, mapsim.config_parameters.NUM_RUNS)
	for i := 0; i < mapsim.config_parameters.NUM_RUNS; i++ {
		vehicle_fetch_history[i] = make([]VehicleInfoDatagram, mapsim.config_parameters.MAX_VEHICLES)
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
		
		var ackReceived sync.WaitGroup
		go RunWebReaderLoop(conn, mapsim, &ackReceived, wg)

		if err := SendMapSetupMessage(conn, mapsim, &ackReceived); err != nil {
			fmt.Println("Write Error:", err)
			return
		}
		ackReceived.Wait()

		run := 0
		for current_run-run > 0 { //Resend all lost packets in order if page connection is lost.
			if err := SendWebVehicleMessage(conn, vehicle_fetch_history[run], &ackReceived); err != nil {
				fmt.Println("Write Error:", err)
				return
			}
			ackReceived.Wait()
			run++
		}
		if err := SendLoadingDoneMessage(conn); err != nil {
			fmt.Println("Write Error:", err)
			return
		}
		for data := range channel { //Reader loop for controller channel
			vehicle_fetch_history[current_run][data.id] = data
			vehicles_seen++
			wg.Done()

			if vehicles_seen >= mapsim.config_parameters.MAX_VEHICLES {
				vehicles_seen -= mapsim.config_parameters.MAX_VEHICLES
				current_run++

				if err := SendWebVehicleMessage(conn, vehicle_fetch_history[current_run-1], &ackReceived); err != nil {
					fmt.Println("Write Error:", err)
					return
				}
				ackReceived.Wait()
			}
		}

	})

	fs := http.FileServer(http.Dir("templates/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})

	http.ListenAndServe(":"+PORT, nil)
}

func SendLoadingDoneMessage(conn *websocket.Conn) error {
	return conn.WriteJSON(Message{Type:"loading_done_message", Data: ""})
}

func SendWebVehicleMessage(conn *websocket.Conn, data []VehicleInfoDatagram, ack_received *sync.WaitGroup) error{
	msg := make([]VehicleMessage, len(data))
	for i:=0;i<len(data);i++ {
		msg[i] = VehicleMessage{Vehicle_ID: data[i].id, X: data[i].x, Y: data[i].y, Time: data[i].time, Status: data[i].status}
	}
	ack_received.Add(1)
	return conn.WriteJSON(Message{Type:"vehicle_message", Data: msg})
}

func SendMapSetupMessage(conn *websocket.Conn, mapsim *Map, ack_received *sync.WaitGroup) error {
	msg := MapSetupMessage{Nodes: make([]MapNodeSetupMessage,len(mapsim.nodes)), Lanes: make([]MapLaneSetupMessage, len(mapsim.lanes)), ConfigParameters: *mapsim.config_parameters}
	for n:=0;n<len(mapsim.nodes);n++ {
		msg.Nodes[n] = MapNodeSetupMessage{Node_ID: mapsim.nodes[n].id, X: mapsim.nodes[n].pos.x, Y: mapsim.nodes[n].pos.y, AgentType: mapsim.nodes[n].agent.Descriptor()}
	}
	for l:=0;l<len(mapsim.lanes);l++ {
		msg.Lanes[l] = MapLaneSetupMessage{Lane_ID: mapsim.lanes[l].id, Start_X: mapsim.lanes[l].start_pos.x, Start_Y: mapsim.lanes[l].start_pos.y, End_X: mapsim.lanes[l].end_pos.x, End_Y: mapsim.lanes[l].end_pos.y}
	}
	ack_received.Add(1)
	return conn.WriteJSON(Message{Type:"map_setup_message", Data: msg})
}

func RunWebReaderLoop(conn *websocket.Conn, mapsim *Map, ackReceived *sync.WaitGroup, wg *sync.WaitGroup) {
	blocking := false
	defer func() {
		if blocking {
			wg.Done()
			mapsim.config_parameters.SIM_RATE = 1
		}
	}()
	for {
		msg, err := ReadWebMessage(conn)
		if err != nil {
			fmt.Println("Read Error:", err)
			return
		}
		switch msg.Type {
		case "ACK":
			ackReceived.Done()
		case "sim_rate_update":
			rate, ok := msg.Data.(float64)
			if !ok {
				fmt.Println("Error: SIM_RATE is not a float!")
				return
			}
			if mapsim.config_parameters.SIM_RATE != rate && rate == 0 {
				wg.Add(1)
				blocking = true
			} else if mapsim.config_parameters.SIM_RATE != rate && mapsim.config_parameters.SIM_RATE == 0 {
				wg.Done()
				blocking = false
			}
			mapsim.config_parameters.SIM_RATE = rate
		default:
			fmt.Println("Read Error: Unknown Web Message!")
			return
		}
	}
}

func ReadWebMessage(conn *websocket.Conn) (*Message,error) {
	var msg Message
	err := conn.ReadJSON(&msg)
	if err != nil {
		return &Message{Type:"",Data:nil}, err
	}
	return &msg, nil
}