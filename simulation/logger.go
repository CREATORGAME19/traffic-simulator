package simulation

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Logger struct {
	current_run                int
	vehicle_fetch_history      [][]VehicleFetchResult
	static_agent_fetch_history [][]LoggerStaticAgentFetchResult
	vehicles_seen              int
	lastRecord                 SimTime

	controller_channel chan SimFetchResult
	mu                 sync.Mutex
	log_file           *os.File
	encoder            *json.Encoder
}

type LoggerStaticAgentFetchResult struct {
	ID                RoadNodeID
	Time              SimTime
	VehiclesProcessed int
	Throughput        float64
}

type LoggerVehicleEvent struct {
	Time      SimTime         `json:"time"`
	VehicleID VehicleID       `json:"vehicle_id"`
	LaneID    LaneID          `json:"lane_id"`
	NodeID    RoadNodeID      `json:"node_id"`
	EventType LoggerEventType `json:"event_type"`
}

type LoggerEventType string

const (
	LoggerVehicleEventLaneIn  LoggerEventType = "VEHICLE ENTER LANE"
	LoggerVehicleEventLaneOut LoggerEventType = "VEHICLE EXIT LANE"
	LoggerVehicleEventSpawn   LoggerEventType = "VEHICLE SPAWN"
	LoggerVehicleEventDestroy LoggerEventType = "VEHICLE DESTROY"
)

func InitLogger(mapsim *Map) (*Logger, chan SimFetchResult) {
	num_record_runs := max(1, int(float64(mapsim.config_parameters.NUM_RUNS)/(mapsim.config_parameters.RECORD_INTERVAL/SIM_TIME_STEP)))
	vehicle_fetch_history := make([][]VehicleFetchResult, num_record_runs)
	for i := 0; i < num_record_runs; i++ {
		vehicle_fetch_history[i] = make([]VehicleFetchResult, mapsim.config_parameters.MAX_VEHICLES)
	}

	static_agent_fetch_history := make([][]LoggerStaticAgentFetchResult, num_record_runs)
	for i := 0; i < num_record_runs; i++ {
		static_agent_fetch_history[i] = make([]LoggerStaticAgentFetchResult, len(mapsim.nodes))
	}

	out_channel := make(chan SimFetchResult, (mapsim.config_parameters.MAX_VEHICLES + len(mapsim.nodes)))
	f, err := os.OpenFile(mapsim.config_parameters.LOG_FILE, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}

	l := &Logger{
		current_run:                0,
		vehicle_fetch_history:      vehicle_fetch_history,
		static_agent_fetch_history: static_agent_fetch_history,
		vehicles_seen:              0,
		lastRecord:                 SimTime(-1 * mapsim.config_parameters.RECORD_INTERVAL),
		controller_channel:         out_channel,
		log_file:                   f,
		encoder:                    json.NewEncoder(f),
	}

	return l, out_channel
}

func (l *Logger) RunLogger(in_channel chan SimFetchResult, log_channel chan LoggerVehicleEvent, mapsim *Map) {
	defer l.Close()
	out_channel := l.controller_channel
	go func() {
		current_vehicle_fetch := make([]VehicleFetchResult, mapsim.config_parameters.MAX_VEHICLES)
		current_agent_fetch := make([]StaticAgentFetchResult, len(mapsim.nodes))
		for i := 0; i < len(mapsim.nodes); i++ {
			current_agent_fetch[i] = StaticAgentFetchResult{VehiclesProcessed: 0}
		}
		for data := range in_channel {
			if data.vehicle_fetch_result != nil {
				vehicle_data := data.vehicle_fetch_result
				current_vehicle_fetch[mapsim.GetMapArrayVehicleIDIndex(vehicle_data.ID)] = *vehicle_data
				l.vehicles_seen++
				if l.vehicles_seen >= mapsim.config_parameters.MAX_VEHICLES {
					l.vehicles_seen -= mapsim.config_parameters.MAX_VEHICLES
					if current_vehicle_fetch[0].Time-l.lastRecord > SimTime(mapsim.config_parameters.RECORD_INTERVAL) || AlmostEqual(mapsim.config_parameters.RECORD_INTERVAL, float64(current_vehicle_fetch[0].Time-l.lastRecord)) {
						l.SaveCurrentVehicleFetchHistory(mapsim, &current_vehicle_fetch)
						l.SaveCurrentAgentFetchHistory(mapsim, &current_agent_fetch)
						l.IncrementCurrentRun()
						l.lastRecord = current_vehicle_fetch[0].Time
					}
				}
			} else if data.static_agent_fetch_result != nil {
				agent_data := data.static_agent_fetch_result
				current_agent_fetch[agent_data.ID] = StaticAgentFetchResult{ID: agent_data.ID, Time: agent_data.Time, VehiclesProcessed: agent_data.VehiclesProcessed + current_agent_fetch[agent_data.ID].VehiclesProcessed}
			}
			out_channel <- data
		}
	}()

	for data := range log_channel {
		if err := l.WriteToLog(data); err != nil {
			fmt.Println(err)
		}
	}
}

func (l *Logger) IncrementCurrentRun() {
	l.current_run++
}

func (l *Logger) SaveCurrentVehicleFetchHistory(mapsim *Map, current_vehicle_fetch *[]VehicleFetchResult) {
	for i := 0; i < mapsim.config_parameters.MAX_VEHICLES; i++ {
		l.vehicle_fetch_history[l.current_run][i] = (*current_vehicle_fetch)[i]
		if (*current_vehicle_fetch)[i].Status != NilVehicle { //Only log if vehicle is spawned in!
			if err := l.WriteToLog((*current_vehicle_fetch)[i]); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (l *Logger) SaveCurrentAgentFetchHistory(mapsim *Map, current_agent_fetch *[]StaticAgentFetchResult) {
	for i := 0; i < len(mapsim.nodes); i++ {
		l.static_agent_fetch_history[l.current_run][i] = LoggerStaticAgentFetchResult{ID: (*current_agent_fetch)[i].ID, Time: (*current_agent_fetch)[i].Time, VehiclesProcessed: (*current_agent_fetch)[i].VehiclesProcessed, Throughput: float64((*current_agent_fetch)[i].VehiclesProcessed) / mapsim.config_parameters.RECORD_INTERVAL}
		if err := l.WriteToLog((*current_agent_fetch)[i]); err != nil {
			fmt.Println(err)
		}
		(*current_agent_fetch)[i].VehiclesProcessed = 0
	}
}

func (l *Logger) WriteToLog(data interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.encoder.Encode(data)
}

func (l *Logger) Close() {
	if l.log_file != nil {
		l.log_file.Close()
	}
}
