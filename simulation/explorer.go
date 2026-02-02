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
	ID        VehicleID
	X, Y      float64
	Status    VehicleStatus
	Time      SimTime
	Speed     float64
	Acc       float64
	Origin    RoadNodeID
	Dest      RoadNodeID
	SpawnTime SimTime
}

type MapSetupMessage struct {
	Nodes            []MapNodeSetupMessage
	Lanes            []MapLaneSetupMessage
	ConfigParameters ConfigParameters
}

type MapNodeSetupMessage struct {
	ID        RoadNodeID
	X, Y      float64
	AgentType StaticAgentType
}

type MapLaneSetupMessage struct {
	ID                             LaneID
	Start_X, Start_Y, End_X, End_Y float64
}

var reader sync.WaitGroup //Only 1 reader and 1 writer web socket at a time
var writer sync.WaitGroup

func runFrontend(channel chan SimFetchResult, simrate_chan chan float64, mapsim *Map) {
	logger := InitLogger(mapsim)

	vehicles_seen := 0 //Vehicles seen in the run so far
	current_vehicle_fetch := make([]VehicleFetchResult, mapsim.config_parameters.MAX_VEHICLES)
	current_agent_fetch := make([]StaticAgentFetchResult, len(mapsim.nodes))
	for i := 0; i< len(mapsim.nodes); i++{
		current_agent_fetch[i] = StaticAgentFetchResult{VehiclesProcessed: 0}
	}
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
		go RunWebReaderLoop(conn, mapsim, webReaderChan, simrate_chan, logger)

		if err := SendMapSetupMessage(conn, &writer, mapsim); err != nil {
			fmt.Println("Write Error:", err)
			return
		}
		if err := <-webReaderChan; err != nil {
			fmt.Println(err)
			return
		}

		run := 0
		for logger.current_run-run > 0 { //Resend all lost packets in order if page connection is lost.
			if err := SendWebVehicleMessage(conn, &writer, logger.vehicle_fetch_history[run]); err != nil {
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
		go func() { //Reader loop for controller channel
			for data := range channel { 
				if data.vehicle_fetch_result != nil {
					vehicle_data := data.vehicle_fetch_result
					current_vehicle_fetch[mapsim.GetMapArrayVehicleIDIndex(vehicle_data.ID)] = *vehicle_data
					vehicles_seen++

					if vehicles_seen >= mapsim.config_parameters.MAX_VEHICLES {
						vehicles_seen -= mapsim.config_parameters.MAX_VEHICLES
						if current_vehicle_fetch[0].Time-lastRecord > SimTime(mapsim.config_parameters.RECORD_INTERVAL) || AlmostEqual(mapsim.config_parameters.RECORD_INTERVAL, float64(current_vehicle_fetch[0].Time-lastRecord)) {
							logger.SaveCurrentVehicleFetchHistory(mapsim,&current_vehicle_fetch)
							logger.SaveCurrentAgentFetchHistory(mapsim,&current_agent_fetch)
							logger.IncrementCurrentRun()
							lastRecord = current_vehicle_fetch[0].Time
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
				} else if data.static_agent_fetch_result != nil {
					agent_data := data.static_agent_fetch_result
					current_agent_fetch[agent_data.ID] = StaticAgentFetchResult{ID: agent_data.ID, Time: agent_data.Time, VehiclesProcessed: agent_data.VehiclesProcessed + current_agent_fetch[agent_data.ID].VehiclesProcessed}
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

func SendWebVehicleMessage(conn *websocket.Conn, writer *sync.WaitGroup, data []VehicleFetchResult) error {
	defer writer.Done()
	writer.Wait()
	writer.Add(1)

	msg := make([]VehicleMessage, len(data))
	for i := 0; i < len(data); i++ {
		msg[i] = VehicleMessage{ID: data[i].ID, X: data[i].X, Y: data[i].Y, Time: data[i].Time, Status: data[i].Status, Speed: data[i].Speed, Acc: data[i].Acc, Origin: data[i].Origin, Dest: data[i].Dest, SpawnTime: data[i].SpawnTime}
	}
	return conn.WriteJSON(Message{Type: "vehicle_message", Data: msg})
}

func SendVehicleLogDataMessage(conn *websocket.Conn, vehicle_fetch_history *[][]VehicleFetchResult, spawn_index int, current_run *int, vehicle_id int) error {
	defer writer.Done()
	writer.Wait()
	writer.Add(1)

	msg := make([]VehicleMessage, *current_run-spawn_index)
	for i := 0; i <= *current_run-1-spawn_index; i++ {
		data := (*vehicle_fetch_history)[i+spawn_index][vehicle_id]
		msg[i] = VehicleMessage{ID: data.ID, X: data.X, Y: data.Y, Time: data.Time, Status: data.Status, Speed: data.Speed, Acc: data.Acc, Origin: data.Origin, Dest: data.Dest, SpawnTime: data.SpawnTime}
	}
	return conn.WriteJSON(Message{Type: "vehicle_log_data_message", Data: msg})
}

func SendStaticAgentLogDataMessage(conn *websocket.Conn, agent_fetch_history *[][]LoggerStaticAgentFetchResult, current_run *int, agent_id int) error {
	defer writer.Done()
	writer.Wait()
	writer.Add(1)

	msg := make([]LoggerStaticAgentFetchResult, *current_run)
	for i := 0; i < *current_run; i++ {
		msg[i] = (*agent_fetch_history)[i][agent_id]
	}
	return conn.WriteJSON(Message{Type: "node_log_data_message", Data: msg})
}

func SendMapSetupMessage(conn *websocket.Conn, writer *sync.WaitGroup, mapsim *Map) error {
	defer writer.Done()
	writer.Wait()
	writer.Add(1)

	msg := MapSetupMessage{Nodes: make([]MapNodeSetupMessage, len(mapsim.nodes)), Lanes: make([]MapLaneSetupMessage, len(mapsim.lanes)), ConfigParameters: *mapsim.config_parameters}
	for n := 0; n < len(mapsim.nodes); n++ {
		msg.Nodes[n] = MapNodeSetupMessage{ID: mapsim.nodes[n].id, X: mapsim.nodes[n].pos.x, Y: mapsim.nodes[n].pos.y, AgentType: mapsim.nodes[n].agent.Descriptor()}
	}
	for l := 0; l < len(mapsim.lanes); l++ {
		msg.Lanes[l] = MapLaneSetupMessage{ID: mapsim.lanes[l].id, Start_X: mapsim.lanes[l].start_pos.x, Start_Y: mapsim.lanes[l].start_pos.y, End_X: mapsim.lanes[l].end_pos.x, End_Y: mapsim.lanes[l].end_pos.y}
	}
	return conn.WriteJSON(Message{Type: "map_setup_message", Data: msg})
}

func RunWebReaderLoop(conn *websocket.Conn, mapsim *Map, webReaderChan chan error, simrate_chan chan float64, logger *Logger) {
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
		case "fetch_vehicle_data":
			vehicle_id, ok := msg.Data.(float64)
			if !ok {
				fmt.Println("Error: vehicle_id is not an int!")
				webReaderChan <- fmt.Errorf("Error: vehicle_id is not an int!")
				return
			}
			lastSpawn := (logger.vehicle_fetch_history)[logger.current_run-1][int(vehicle_id)].SpawnTime
			curr_time := (logger.vehicle_fetch_history)[logger.current_run-1][int(vehicle_id)].Time
			spawn_index := (logger.current_run - 1) - int((curr_time-lastSpawn)/SimTime(mapsim.config_parameters.RECORD_INTERVAL))
			if err := SendVehicleLogDataMessage(conn, &logger.vehicle_fetch_history, spawn_index, &logger.current_run, int(vehicle_id)); err != nil {
				fmt.Println(err)
				webReaderChan <- err
				return
			}
		case "fetch_node_data":
			node_id, ok := msg.Data.(float64)
			if !ok {
				fmt.Println("Error: node_id is not an int!")
				webReaderChan <- fmt.Errorf("Error: node_id is not an int!")
				return
			}
			if err := SendStaticAgentLogDataMessage(conn, &logger.static_agent_fetch_history, &logger.current_run, int(node_id)); err != nil {
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
