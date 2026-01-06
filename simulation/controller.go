package simulation

import (
	"fmt"
	"sync"
)

const MAX_VEHICLES = 1

type VehicleLocation struct {
	x    float64
	y    float64
	time int64
}

func RunController() {
	var wg sync.WaitGroup
	frontend_chan := make(chan []VehicleLocation)
	go runFrontend(&wg, frontend_chan)

	err := open_url("http://localhost:" + PORT)
	if err != nil {
		fmt.Println("Error opening browser!")
		return
	}

	// Initialize Map
	nodes := []RoadNode{
		NewRoadNode(
			0,
			NewPosition(0, 0),
			[]int{0},
			[]int{},
			SpawnerRoadNode,
		),
		NewRoadNode(
			1,
			NewPosition(1, 1),
			[]int{1},
			[]int{0},
			IntersectionRoadNode,
		),
		NewRoadNode(
			2,
			NewPosition(2, 2),
			[]int{},
			[]int{1},
			SinkRoadNode,
		),
	}
	lanes := []Lane{
		NewLane(
			0,
			NewPosition(0, 0),
			NewPosition(1, 1),
			0,
			1,
		),
		NewLane(
			1,
			NewPosition(1, 1),
			NewPosition(2, 2),
			1,
			2,
		),
	}
	mapsim := InitialiseMap(nodes, lanes)

	fmt.Println("Map Simulation: ", mapsim)

	// Initialize Vehicles
	vehicles := CreateVehicles(mapsim)
	vehicle_locations := make([]VehicleLocation, MAX_VEHICLES)
	vehicle_channel := make(chan VehicleFetchResult, MAX_VEHICLES) //Set number of vehicles as the size of buffer

	time := 0
	fmt.Println("Vehicles Start: ", vehicles)
	for i := 0; i < 400; i++ { //Temporary variable i
		for v := 0; v < len(vehicles); v++ {
			go vehicles[v].FetchVehicleSim(int64(time), vehicle_channel)
		}
		wg.Wait()
		for v := 0; v < len(vehicles); v++ {
			if vehicles[v] != nil {
				fetchresult := <-vehicle_channel 
				vehicle_locations[fetchresult.Vehicle_ID] = VehicleLocation{fetchresult.X, fetchresult.Y, fetchresult.Time} //TODO: Change this to send individual vehicles rather than grouping them.
			}
		}
		wg.Add(1)
		frontend_chan <- vehicle_locations
		time++
	}

	select {} //Temporary
}
