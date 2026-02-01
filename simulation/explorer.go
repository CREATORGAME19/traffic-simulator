package simulation

import (
	"fmt"
	"net/http"
	"sync"
	"time"

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
	X, Y       float64
	Status     VehicleStatus
	Time       SimTime
	Speed      float64
	Acc        float64
	Origin     RoadNodeID
	Dest       RoadNodeID
	SpawnTime  SimTime
}

type MapSetupMessage struct {
	Nodes            []MapNodeSetupMessage
	Lanes            []MapLaneSetupMessage
	ConfigParameters ConfigParameters
}

type MapNodeSetupMessage struct {
	Node_ID   RoadNodeID
	X, Y      float64
	AgentType StaticAgentType
}

type MapLaneSetupMessage struct {
	Lane_ID                        LaneID
	Start_X, Start_Y, End_X, End_Y float64
}

var reader sync.WaitGroup //Only 1 reader and 1 writer web socket at a time
var writer sync.WaitGroup

func runFrontend(channel chan VehicleInfoResult, simrate_chan chan float64, mapsim *Map) {
	num_record_runs := max(1, int(float64(mapsim.config_parameters.NUM_RUNS)/(mapsim.config_parameters.RECORD_INTERVAL/SIM_TIME_STEP)))
	vehicle_fetch_history := make([][]VehicleInfoResult, num_record_runs)
	for i := 0; i < num_record_runs; i++ {
		vehicle_fetch_history[i] = make([]VehicleInfoResult, mapsim.config_parameters.MAX_VEHICLES)
	}

	current_run := new(int)
	*current_run = 0
	vehicles_seen := 0 //Vehicles seen in the run so far
	current_vehicle_fetch := make([]VehicleInfoResult, mapsim.config_parameters.MAX_VEHICLES)
	lastRecord := SimTime(-1 * mapsim.config_parameters.RECORD_INTERVAL)

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

		var webReaderChan chan error = make(chan error)
		go RunWebReaderLoop(conn, mapsim, webReaderChan, simrate_chan, &vehicle_fetch_history, current_run)

		if err := SendMapSetupMessage(conn, &writer, mapsim); err != nil {
			fmt.Println("Write Error:", err)
			return
		}
		if err := <-webReaderChan; err != nil {
			fmt.Println(err)
			return
		}

		run := 0
		for *current_run-run > 0 { //Resend all lost packets in order if page connection is lost.
			if err := SendWebVehicleMessage(conn, &writer, vehicle_fetch_history[run]); err != nil {
				fmt.Println("Write Error:", err)
				return
			}
			if err := <-webReaderChan; err != nil {
				fmt.Println(err)
				return
			}
			run++
		}
		if err := SendLoadingDoneMessage(conn, &writer); err != nil {
			fmt.Println("Write Error:", err)
			return
		}
		go func() {
			for data := range channel { //Reader loop for controller channel
				current_vehicle_fetch[data.id] = data
				vehicles_seen++

				if vehicles_seen >= mapsim.config_parameters.MAX_VEHICLES {
					vehicles_seen -= mapsim.config_parameters.MAX_VEHICLES
					if current_vehicle_fetch[0].time-lastRecord > SimTime(mapsim.config_parameters.RECORD_INTERVAL) || AlmostEqual(mapsim.config_parameters.RECORD_INTERVAL, float64(current_vehicle_fetch[0].time-lastRecord)) {
						for i := 0; i < mapsim.config_parameters.MAX_VEHICLES; i++ {
							vehicle_fetch_history[*current_run][i] = current_vehicle_fetch[i]
						}
						*current_run++
						lastRecord = current_vehicle_fetch[0].time
					}
					if err := SendWebVehicleMessage(conn, &writer, current_vehicle_fetch); err != nil {
						fmt.Println("Write Error:", err)
						return
					}
					if err := <-webReaderChan; err != nil {
						fmt.Println(err)
						return
					}
				}
			}
		}()
		for { //Checks if frontend websocket is still active periodically
			if err := SendPingMessage(conn, &writer); err != nil {
				fmt.Println("Write Error:", err)
				return
			}
			time.Sleep(5 * time.Second)
		}
	})

	fs := http.FileServer(http.Dir("templates/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})

	http.ListenAndServe(":"+PORT, nil)
}

func SendLoadingDoneMessage(conn *websocket.Conn, writer *sync.WaitGroup) error {
	defer writer.Done()
	writer.Wait()
	writer.Add(1)
	return conn.WriteJSON(Message{Type: "loading_done_message", Data: ""})
}

func SendWebVehicleMessage(conn *websocket.Conn, writer *sync.WaitGroup, data []VehicleInfoResult) error {
	defer writer.Done()
	writer.Wait()
	writer.Add(1)

	msg := make([]VehicleMessage, len(data))
	for i := 0; i < len(data); i++ {
		msg[i] = VehicleMessage{Vehicle_ID: data[i].id, X: data[i].x, Y: data[i].y, Time: data[i].time, Status: data[i].status, Speed: data[i].speed, Acc: data[i].acc, Origin: data[i].origin, Dest: data[i].dest, SpawnTime: data[i].spawn_time}
	}
	return conn.WriteJSON(Message{Type: "vehicle_message", Data: msg})
}

func SendVehicleSpeedMessage(conn *websocket.Conn, vehicle_fetch_history *[][]VehicleInfoResult, spawn_index int, current_run *int, vehicle_id int) error {
	defer writer.Done()
	writer.Wait()
	writer.Add(1)

	msg := make([]VehicleMessage, *current_run-spawn_index)
	for i := 0; i <= *current_run -1 - spawn_index; i++ {
		data := (*vehicle_fetch_history)[i+spawn_index][vehicle_id]
		msg[i] = VehicleMessage{Vehicle_ID: data.id, X: data.x, Y: data.y, Time: data.time, Status: data.status, Speed: data.speed, Acc: data.acc, Origin: data.origin, Dest: data.dest, SpawnTime: data.spawn_time}
	}
	return conn.WriteJSON(Message{Type: "vehicle_speed_message", Data: msg})
} 

func SendMapSetupMessage(conn *websocket.Conn, writer *sync.WaitGroup, mapsim *Map) error {
	defer writer.Done()
	writer.Wait()
	writer.Add(1)

	msg := MapSetupMessage{Nodes: make([]MapNodeSetupMessage, len(mapsim.nodes)), Lanes: make([]MapLaneSetupMessage, len(mapsim.lanes)), ConfigParameters: *mapsim.config_parameters}
	for n := 0; n < len(mapsim.nodes); n++ {
		msg.Nodes[n] = MapNodeSetupMessage{Node_ID: mapsim.nodes[n].id, X: mapsim.nodes[n].pos.x, Y: mapsim.nodes[n].pos.y, AgentType: mapsim.nodes[n].agent.Descriptor()}
	}
	for l := 0; l < len(mapsim.lanes); l++ {
		msg.Lanes[l] = MapLaneSetupMessage{Lane_ID: mapsim.lanes[l].id, Start_X: mapsim.lanes[l].start_pos.x, Start_Y: mapsim.lanes[l].start_pos.y, End_X: mapsim.lanes[l].end_pos.x, End_Y: mapsim.lanes[l].end_pos.y}
	}
	return conn.WriteJSON(Message{Type: "map_setup_message", Data: msg})
}

func RunWebReaderLoop(conn *websocket.Conn, mapsim *Map, webReaderChan chan error, simrate_chan chan float64, vehicle_fetch_history *[][]VehicleInfoResult, current_run *int) {
	defer func() {
		if mapsim.config_parameters.SIM_RATE == 0 {
			mapsim.config_parameters.SIM_RATE = 1
			simrate_chan <- 1
		}
		close(webReaderChan)
	}()
	for {
		msg, err := ReadWebMessage(conn)
		if err != nil {
			fmt.Println("Read Error:", err)
			webReaderChan <- err
			return
		}
		switch msg.Type {
		case "ACK":
			webReaderChan <- nil
		case "sim_rate_update":
			rate, ok := msg.Data.(float64)
			if !ok {
				fmt.Println("Error: SIM_RATE is not a float!")
				webReaderChan <- fmt.Errorf("Error: SIM_RATE is not a float!")
				return
			}
			mapsim.config_parameters.SIM_RATE = rate
			select {
			case simrate_chan <- rate:
				//Nothing
			default:
				//Nothing
			}
		case "fetch_vehicle_speed":
			vehicle_id, ok := msg.Data.(float64)
			if !ok {
				fmt.Println("Error: vehicle_id is not an int!")
				webReaderChan <- fmt.Errorf("Error: vehicle_id is not an int!")
				return
			}
			lastSpawn := (*vehicle_fetch_history)[*current_run -1][int(vehicle_id)].spawn_time
			curr_time := (*vehicle_fetch_history)[*current_run -1][int(vehicle_id)].time
			spawn_index := (*current_run-1) - int((curr_time-lastSpawn)/SimTime(mapsim.config_parameters.RECORD_INTERVAL))
			if err := SendVehicleSpeedMessage(conn, vehicle_fetch_history, spawn_index, current_run, int(vehicle_id)); err != nil {
				fmt.Println(err)
				webReaderChan <- err
				return
			}
		default:
			fmt.Println("Read Error: Unknown Web Message!")
			webReaderChan <- fmt.Errorf("Read Error: Unknown Web Message!")
			return
		}
	}
}

func ReadWebMessage(conn *websocket.Conn) (*Message, error) {
	var msg Message
	err := conn.ReadJSON(&msg)
	if err != nil {
		return &Message{Type: "", Data: nil}, err
	}
	return &msg, nil
}

func SendPingMessage(conn *websocket.Conn, writer *sync.WaitGroup) error {
	defer writer.Done()
	writer.Wait()
	writer.Add(1)

	return conn.WriteJSON(Message{Type: "ping_message", Data: ""})
}
