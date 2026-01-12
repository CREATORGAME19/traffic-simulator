package simulation

import (
	"fmt"
	"math"
	"math/rand/v2"
)

type Position struct {
	x float64
	y float64
}

func NewPosition(x float64, y float64) Position {
	return Position{x: x, y: y}
}

type RoadNode struct {
	id        int
	pos       Position
	lanes_out []int
	lanes_in  []int
	agent     StaticAgent
}

func NewRoadNode(id int, pos Position, lanes_out []int, lanes_in []int, agent StaticAgent) RoadNode {
	return RoadNode{id: id, pos: pos, lanes_out: lanes_out, lanes_in: lanes_in, agent: agent}
}

type Lane struct {
	id            int
	start_pos     Position
	end_pos       Position
	from          int //Node id
	to            int //Node id
	distance      float64
	vehicle_queue *VehicleLaneQueue

	// more Lane Properties here (like speed limit)
}

func NewLane(id int, start_pos Position, end_pos Position, from int, to int) Lane {
	return Lane{id: id, start_pos: start_pos, end_pos: end_pos, from: from, to: to, distance: CalculateDistance(start_pos, end_pos), vehicle_queue: EmptyVehicleLaneQueue()}
}

func CalculateDistance(p1 Position, p2 Position) float64 {
	change_x := p1.x - p2.x
	change_y := p1.y - p2.y
	return math.Sqrt(math.Pow(change_x, 2) + math.Pow(change_y, 2))
}

type VehicleDB struct {
	vehicle_array []*Vehicle
	next_empty    int64
}

func FindNextEmptyVehicles(mapsim *Map) int {
	for i := 0; i < MAX_VEHICLES; i++ {
		if mapsim.vehicles.vehicle_array[i] == nil {
			return i
		}
	}
	return MAX_VEHICLES
}

type Map struct {
	nodes    []RoadNode
	lanes    []Lane
	vehicles VehicleDB
}

func InitialiseMap(v []RoadNode, l []Lane) *Map {
	return &Map{
		nodes:    v,
		lanes:    l,
		vehicles: VehicleDB{vehicle_array: make([]*Vehicle, MAX_VEHICLES), next_empty: 0},
	}
}

func (m *Map) AddNode(v RoadNode) {
	m.nodes = append(m.nodes, v)
}

func (m *Map) AddLane(l Lane) {
	m.lanes = append(m.lanes, l)
}

func (m *Map) FindADestinationNode() int {
	dest_nodes := []int{}
	for n := 0; n < len(m.nodes); n++ {
		if m.nodes[n].agent.Descriptor() == SinkAgentType {
			dest_nodes = append(dest_nodes, n)
		}
	}
	return dest_nodes[rand.IntN(len(dest_nodes))]
}

type VehicleLaneQueue struct {
	vehicles   []*Vehicle
}

func EmptyVehicleLaneQueue() *VehicleLaneQueue {
	return &VehicleLaneQueue{vehicles: make([]*Vehicle, MAX_VEHICLES)}
}

func (l *Lane) FindNextVehicleAhead(v *Vehicle, t SimTime) *Vehicle {
	progress := v.pos.progress
	closest_progress := 1.1
	var closest_vehicle *Vehicle
	closest_vehicle = nil
	for i := 0; i < MAX_VEHICLES; i++ {
		if (l.vehicle_queue.vehicles[i] != nil) && (progress == l.vehicle_queue.vehicles[i].pos.progress) {
			//println("Collision:",v.id,i,t, progress) //Warning
		}
		if (l.vehicle_queue.vehicles[i] != nil) && (l.vehicle_queue.vehicles[i] != v) && (progress < l.vehicle_queue.vehicles[i].pos.progress) && (l.vehicle_queue.vehicles[i].pos.progress < closest_progress) {
			closest_progress = l.vehicle_queue.vehicles[i].pos.progress
			closest_vehicle = l.vehicle_queue.vehicles[i]
		}
	}
	return closest_vehicle
}

func (l *Lane) FindNextVehicleAheadFromPos(desired_pos VehiclePosition) *Vehicle {
	progress := desired_pos.progress
	closest_progress := 1.1
	var closest_vehicle *Vehicle
	closest_vehicle = nil
	for i := 0; i < MAX_VEHICLES; i++ {
		vehicle_candidate := l.vehicle_queue.vehicles[i]
		if (vehicle_candidate != nil) && (progress <= vehicle_candidate.pos.progress) && (vehicle_candidate.pos.progress < closest_progress) {
			closest_progress = vehicle_candidate.pos.progress
			closest_vehicle = vehicle_candidate
		}
	}
	return closest_vehicle
}

func (l *Lane) RemoveVehicleFromQueue(v *Vehicle) {
	l.vehicle_queue.vehicles[v.id] = nil
}

func (l *Lane) AddVehicleToQueue(v *Vehicle) {
	if l.vehicle_queue.vehicles[v.id] != nil {
		fmt.Println("Error: Vehicle queue", v.id, "is not empty!")
		return
	}
	l.vehicle_queue.vehicles[v.id] = v
}
