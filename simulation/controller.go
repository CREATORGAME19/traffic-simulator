package simulation

import (
	"fmt"
	"sync"
	"time"
)

const MAX_VEHICLES = 100

const NUM_RUNS = 200

type VehicleLocation struct {
	id     VehicleID
	x      float64
	y      float64
	time   SimTime
	status VehicleStatus
}

type VehicleInfoDatagram VehicleLocation

type SimTime int64

func RunController() {
	var wg sync.WaitGroup

	// Initialize Map
	nodes := []RoadNode{
		NewRoadNode(
			0,
			NewPosition(-400, -400),
			[]LaneID{0},
			[]LaneID{},
			NewSpawnerAgent(0, 1),
		),
		NewRoadNode(
			1,
			NewPosition(0, 0),
			[]LaneID{1},
			[]LaneID{0},
			NewIntersectionAgent(1),
		),
		NewRoadNode(
			2,
			NewPosition(600, 600),
			[]LaneID{},
			[]LaneID{1},
			NewSinkAgent(2),
		),
	}
	lanes := []Lane{
		NewLane(
			0,
			NewPosition(-400, -400),
			NewPosition(0, 0),
			0,
			1,
		),
		NewLane(
			1,
			NewPosition(0, 0),
			NewPosition(600, 600),
			1,
			2,
		),
	}
	mapsim := InitialiseMap(nodes, lanes)

	frontend_chan := make(chan VehicleInfoDatagram, MAX_VEHICLES)
	go runFrontend(&wg, frontend_chan, mapsim, NUM_RUNS)

	err := open_url("http://localhost:" + PORT)
	if err != nil {
		fmt.Println("Error opening browser!")
		return
	}

	fmt.Println("Map Simulation: ", mapsim)

	// Initialize Vehicles
	vehicle_locations := make([]VehicleLocation, MAX_VEHICLES)
	vehicle_channel := make(chan VehicleFetchResult, MAX_VEHICLES) //Set number of vehicles as the size of buffer

	var sim_time SimTime
	sim_time = 0
	minDuration := 500*time.Millisecond

	for i := 0; i < NUM_RUNS; i++ {
		start := time.Now()
		for a := 0; a < len(mapsim.nodes); a++ { //Trigger spawners each time
			mapsim.nodes[a].agent.SpawnVehicles(mapsim, sim_time)
		}
		for v := 0; v < MAX_VEHICLES; v++ {
			go mapsim.vehicles.vehicle_array[v].FetchVehicleSim(mapsim, sim_time, vehicle_channel, VehicleID(v))
		}
		wg.Wait()
		for v := 0; v < MAX_VEHICLES; v++ {
			wg.Add(1)
			fetchresult := <-vehicle_channel
			vehicle_locations[fetchresult.Vehicle_ID] = VehicleLocation{fetchresult.Vehicle_ID, fetchresult.X, fetchresult.Y, fetchresult.Time, fetchresult.Status}
		}
		for v := 0; v < MAX_VEHICLES; v++ {
			frontend_chan <- VehicleInfoDatagram(vehicle_locations[v])
		}
		sim_time++
		elapsed := time.Since(start)
		time.Sleep(minDuration - elapsed)
	}

	select {} //Temporary
}
