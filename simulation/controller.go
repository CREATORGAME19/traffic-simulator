package simulation

import (
	"fmt"
	"math/rand/v2"
	"time"
)

type SimFetchResult struct {
	vehicle_fetch_result      *VehicleFetchResult
	static_agent_fetch_result *StaticAgentFetchResult
}

type SimTime float64

const SIM_TIME_STEP = 0.1 //seconds

func RunController(map_config *MapConfig, config_parameters *ConfigParameters) {
	// Initialize Map

	nodes := make([]RoadNode, len(map_config.Road_nodes))
	lanes := make([]Lane, len(map_config.Lanes))

	for n := 0; n < len(nodes); n++ {
		agent, err := NewStaticAgent(map_config.Road_nodes[n].ID, map_config.Road_nodes[n].Agent, map_config.Road_nodes[n].AgentProp)
		if err != nil {
			fmt.Println(err)
			return
		}
		nodes[int(map_config.Road_nodes[n].ID)] = NewRoadNode(
			map_config.Road_nodes[n].ID, // Should be equal to n
			NewPosition(map_config.Road_nodes[n].Position.X, map_config.Road_nodes[n].Position.Y),
			map_config.Road_nodes[n].Lanes_Out,
			map_config.Road_nodes[n].Lanes_In,
			map_config.Road_nodes[n].Radius,
			agent,
		)
	}

	for l := 0; l < len(lanes); l++ {
		lanes[int(map_config.Lanes[l].ID)] = NewLane(
			map_config.Lanes[l].ID, //Should be equal to l
			NewPosition(map_config.Lanes[l].Start_Position.X, map_config.Lanes[l].Start_Position.Y),
			NewPosition(map_config.Lanes[l].End_Position.X, map_config.Lanes[l].End_Position.Y),
			map_config.Lanes[l].From_Node,
			map_config.Lanes[l].To_Node,
			config_parameters,
		)
	}

	mapsim := InitialiseMap(nodes, lanes, config_parameters)

	frontend_chan := make(chan SimFetchResult, (config_parameters.MAX_VEHICLES + len(mapsim.nodes)))
	simrate_chan := make(chan float64)
	go runFrontend(frontend_chan, simrate_chan, mapsim)

	defer close(frontend_chan)

	err := open_url("http://localhost:" + PORT)
	if err != nil {
		fmt.Println("Error opening browser!")
		return
	}

	fmt.Println("Map Simulation: ", mapsim)

	// Initialize Vehicles
	vehicle_locations := make([]VehicleFetchResult, config_parameters.MAX_VEHICLES)
	vehicle_channel := make(chan VehicleFetchResult, config_parameters.MAX_VEHICLES) //Set number of vehicles as the size of buffer

	agent_fetch := make([]StaticAgentFetchResult, len(mapsim.nodes))
	agent_channel := make(chan StaticAgentFetchResult, len(mapsim.nodes))

	var sim_time SimTime
	sim_time = 0
	for i := 0; i < config_parameters.NUM_RUNS; i++ {
		for mapsim.config_parameters.SIM_RATE == 0 { //Wait until SIM_RATE > 0
			<-simrate_chan
		}
		minDuration := time.Duration((SIM_TIME_STEP / mapsim.config_parameters.SIM_RATE) * 1000000000) //Defines time duration for each iteration
		start := time.Now()
		for _, a := range rand.Perm(len(mapsim.nodes)) { //Update spawners/intersections each time
			mapsim.nodes[a].agent.Poke(mapsim, sim_time, agent_channel)
		}
		for m := 0; m < len(mapsim.nodes); m++ {
			fetchresult := <-agent_channel
			agent_fetch[fetchresult.ID] = fetchresult
		}
		for v := 0; v < config_parameters.MAX_VEHICLES; v++ {
			go mapsim.vehicles.vehicle_array[v].FetchVehicleSim(mapsim, sim_time, vehicle_channel, mapsim.GetRealVehicleID(v))
		}
		for v := 0; v < config_parameters.MAX_VEHICLES; v++ {
			fetchresult := <-vehicle_channel
			vehicle_locations[mapsim.GetMapArrayVehicleIDIndex(fetchresult.ID)] = fetchresult
		}
		for m:=0; m < len(mapsim.nodes); m++ {
			a_copy := agent_fetch[m]
			frontend_chan <- SimFetchResult{static_agent_fetch_result: &a_copy}
		}
		for v := 0; v < config_parameters.MAX_VEHICLES; v++ {
			v_copy := vehicle_locations[v]
			frontend_chan <- SimFetchResult{vehicle_fetch_result: &v_copy}
		}
		sim_time += SimTime(SIM_TIME_STEP)
		elapsed := time.Since(start)
		time.Sleep(minDuration - elapsed)
	}
}
