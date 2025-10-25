package main

import (
	"fmt"
	. "traffic-simulator/simulation"
)

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
			1,
		),
	}
	mapsim := InitialiseMap(vertices, links)

	fmt.Printf("Map Simulation: %+v", mapsim)

	// Initialize Agent
	agents := []Agent{
		NewAgent(
			AgentProp{},
			NewPosition(0, 0),
			NewPosition(1, 1),
		),
	}

	fmt.Printf("Agent: %+v", agents)

}
