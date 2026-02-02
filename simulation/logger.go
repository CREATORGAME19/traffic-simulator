package simulation

type Logger struct {
	current_run                int
	vehicle_fetch_history      [][]VehicleFetchResult
	static_agent_fetch_history [][]LoggerStaticAgentFetchResult
}

type LoggerStaticAgentFetchResult struct {
	ID                RoadNodeID
	Time              SimTime
	VehiclesProcessed int
	Throughput        float64
}

func InitLogger(mapsim *Map) *Logger {
	num_record_runs := max(1, int(float64(mapsim.config_parameters.NUM_RUNS)/(mapsim.config_parameters.RECORD_INTERVAL/SIM_TIME_STEP)))
	vehicle_fetch_history := make([][]VehicleFetchResult, num_record_runs)
	for i := 0; i < num_record_runs; i++ {
		vehicle_fetch_history[i] = make([]VehicleFetchResult, mapsim.config_parameters.MAX_VEHICLES)
	}

	static_agent_fetch_history := make([][]LoggerStaticAgentFetchResult, num_record_runs)
	for i := 0; i < num_record_runs; i++ {
		static_agent_fetch_history[i] = make([]LoggerStaticAgentFetchResult, len(mapsim.nodes))
	}

	return &Logger{current_run: 0, vehicle_fetch_history: vehicle_fetch_history, static_agent_fetch_history: static_agent_fetch_history}
}

func (l *Logger) IncrementCurrentRun() {
	l.current_run++
}

func (l *Logger) SaveCurrentVehicleFetchHistory(mapsim *Map, current_vehicle_fetch *[]VehicleFetchResult) {
	for i := 0; i < mapsim.config_parameters.MAX_VEHICLES; i++ {
		l.vehicle_fetch_history[l.current_run][i] = (*current_vehicle_fetch)[i]
	}
}

func (l *Logger) SaveCurrentAgentFetchHistory(mapsim *Map, current_agent_fetch *[]StaticAgentFetchResult) {
	for i := 0; i < len(mapsim.nodes); i++ {
		l.static_agent_fetch_history[l.current_run][i] = LoggerStaticAgentFetchResult{ID: (*current_agent_fetch)[i].ID, Time: (*current_agent_fetch)[i].Time, VehiclesProcessed: (*current_agent_fetch)[i].VehiclesProcessed, Throughput: float64((*current_agent_fetch)[i].VehiclesProcessed) / mapsim.config_parameters.RECORD_INTERVAL}
		(*current_agent_fetch)[i].VehiclesProcessed = 0
	}
}
