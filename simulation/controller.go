package simulation

import (
	"fmt"
	"time"
	"math/rand/v2"
)

type VehicleLocation struct {
	id     VehicleID
	x      float64
	y      float64
	time   SimTime
	status VehicleStatus
}

type VehicleInfoDatagram VehicleLocation

type SimTime float64

const SIM_TIME_STEP = 0.1 //seconds

func RunController(map_config *MapConfig, config_parameters *ConfigParameters) {
	// Initialize Map

	nodes := make([]RoadNode,len(map_config.Road_nodes))
	lanes := make([]Lane,len(map_config.Lanes))

	for n:=0;n<len(nodes);n++{
		agent,err := NewStaticAgent(map_config.Road_nodes[n].ID, map_config.Road_nodes[n].Agent, map_config.Road_nodes[n].AgentProp)
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

	for l:=0;l<len(lanes);l++{
		lanes[int(map_config.Lanes[l].ID)] = NewLane(
			map_config.Lanes[l].ID, //Should be equal to l
			NewPosition(map_config.Lanes[l].Start_Position.X, map_config.Lanes[l].Start_Position.Y),
			NewPosition(map_config.Lanes[l].End_Position.X,map_config.Lanes[l].End_Position.Y),
			map_config.Lanes[l].From_Node,
			map_config.Lanes[l].To_Node,
			config_parameters,
		)
	}

	mapsim := InitialiseMap(nodes, lanes, config_parameters)

	frontend_chan := make(chan VehicleInfoDatagram, config_parameters.MAX_VEHICLES)
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
	vehicle_locations := make([]VehicleLocation, config_parameters.MAX_VEHICLES)
	vehicle_channel := make(chan VehicleFetchResult, config_parameters.MAX_VEHICLES) //Set number of vehicles as the size of buffer

	var sim_time SimTime
	sim_time = 0
	for i := 0; i < config_parameters.NUM_RUNS; i++ {
		for mapsim.config_parameters.SIM_RATE == 0 { //Wait until SIM_RATE > 0
			<-simrate_chan
		}
		minDuration := time.Duration((SIM_TIME_STEP/mapsim.config_parameters.SIM_RATE)*1000000000) //Defines time duration for each iteration
		start := time.Now()
		for _,a := range rand.Perm(len(mapsim.nodes)) { //Update spawners/intersections each time
			mapsim.nodes[a].agent.Poke(mapsim, sim_time)
		}
		for v := 0; v < config_parameters.MAX_VEHICLES; v++ {
			go mapsim.vehicles.vehicle_array[v].FetchVehicleSim(mapsim, sim_time, vehicle_channel, VehicleID(v))
		}
		for v := 0; v < config_parameters.MAX_VEHICLES; v++ {
			fetchresult := <-vehicle_channel
			vehicle_locations[fetchresult.Vehicle_ID] = VehicleLocation{fetchresult.Vehicle_ID, fetchresult.X, fetchresult.Y, fetchresult.Time, fetchresult.Status}
		}
		for v := 0; v < config_parameters.MAX_VEHICLES; v++ {
			frontend_chan <- VehicleInfoDatagram(vehicle_locations[v])
		}
		sim_time += SimTime(SIM_TIME_STEP)
		elapsed := time.Since(start)
		time.Sleep(minDuration - elapsed)
	}
}
