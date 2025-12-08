package simulation

import (
	"fmt"
)

type VehicleLocation struct {
	x    float64
	y    float64
	time int64
}

func RunController() {
	//go runFrontend()

	// Initialize Map
	nodes := []Node{
		NewNode(
			0,
			NewPosition(0, 0),
			[]int{0},
			[]int{},
		),
		NewNode(
			1,
			NewPosition(1, 1),
			[]int{1},
			[]int{0},
		),
		NewNode(
			2,
			NewPosition(2, 2),
			[]int{},
			[]int{1},
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
	vehicle_locations := make([]VehicleLocation, len(vehicles))
	vehicle_channel := make(chan VehicleFetchResult, len(vehicles)) //Set number of vehicles as the size of buffer

	time := 0

	for v := 0; v < len(vehicles); v++ {
		go vehicles[v].StartVehicleSim(int64(time), vehicle_channel)
	}
	for v := 0; v < len(vehicles); v++ {
		fetchresult := <-vehicle_channel
		vehicle_locations[fetchresult.Vehicle_ID] = VehicleLocation{fetchresult.X, fetchresult.Y, fetchresult.Time}
	}
	fmt.Println("Vehicles Start: ", vehicles)
	for i := 0; i < 400; i++ {
		time++

		for v := 0; v < len(vehicles); v++ {
			go vehicles[v].FetchVehicleSim(int64(time), vehicle_channel)
		}
		for v := 0; v < len(vehicles); v++ {
			fetchresult := <-vehicle_channel
			vehicle_locations[fetchresult.Vehicle_ID] = VehicleLocation{fetchresult.X, fetchresult.Y, fetchresult.Time}
		}
		fmt.Println("Vehicles at time", i, ":", vehicle_locations[0])
	}

	//select {} //Temporary
}
