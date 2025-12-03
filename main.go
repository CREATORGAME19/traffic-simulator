package main

import (
	"fmt"
	. "traffic-simulator/simulation"
)

type VehicleLocation struct {
	x    float64
	y    float64
	time int64
}

func main() {
	fmt.Printf("Start")

	// Initialize Map
	vertices := []Vertex{
		NewVertex(
			NewPosition(0, 0),
			[]int{0},
		),
		NewVertex(
			NewPosition(1, 1),
			[]int{},
		),
	}
	links := []Link{
		NewLink(
			0,
			1,
			1.61,
		),
	}
	mapsim := InitialiseMap(vertices, links)

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
	for i := 0; i < 100; i++ {
		time++

		for v := 0; v < len(vehicles); v++ {
			go vehicles[v].FetchVehicleSim(int64(time), vehicle_channel)
		}
		for v := 0; v < len(vehicles); v++ {
			fetchresult := <-vehicle_channel
			vehicle_locations[fetchresult.Vehicle_ID] = VehicleLocation{fetchresult.X, fetchresult.Y, fetchresult.Time}
		}
		fmt.Println("Vehicles at time", i, ":", *(vehicles[0]))
	}

}
