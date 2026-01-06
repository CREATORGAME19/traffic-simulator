package simulation

import (
	"math"
)

type Position struct {
	x float64
	y float64
}

func NewPosition(x float64, y float64) Position {
	return Position{x: x, y: y}
}

type RoadNodeType string

const (
	IntersectionRoadNode RoadNodeType = "intersection"
	SpawnerRoadNode      RoadNodeType = "spawner"
	SinkRoadNode         RoadNodeType = "sink"
)

type RoadNodeTypeProperties struct {
	//TODO
}

type RoadNode struct {
	id        int
	pos       Position
	lanes_out []int
	lanes_in  []int
	node_type RoadNodeType
	node_prop *RoadNodeTypeProperties //TODO: Add way of defining per-node properties
}

func NewRoadNode(id int, pos Position, lanes_out []int, lanes_in []int, node_type RoadNodeType) RoadNode {
	return RoadNode{id: id, pos: pos, lanes_out: lanes_out, lanes_in: lanes_in, node_type: node_type}
}

type Lane struct {
	id            int
	start_pos     Position
	end_pos       Position
	from          int //Node id
	to            int //Node id
	distance      float64
	vehicle_queue VehicleQueue

	// more Lane Properties here (like speed limit)
}

func NewLane(id int, start_pos Position, end_pos Position, from int, to int) Lane {
	return Lane{id: id, start_pos: start_pos, end_pos: end_pos, from: from, to: to, distance: CalculateDistance(start_pos, end_pos), vehicle_queue: EmptyVehicleQueue()}
}

func CalculateDistance(p1 Position, p2 Position) float64 {
	change_x := p1.x - p2.x
	change_y := p1.y - p2.y
	return math.Sqrt(math.Pow(change_x, 2) + math.Pow(change_y, 2))
}

type Map struct {
	nodes []RoadNode
	lanes []Lane
}

func InitialiseMap(v []RoadNode, l []Lane) *Map {
	return &Map{
		nodes: v,
		lanes: l,
	}
}

func (m *Map) AddNode(v RoadNode) {
	m.nodes = append(m.nodes, v)
}

func (m *Map) AddLane(l Lane) {
	m.lanes = append(m.lanes, l)
}

type VehicleQueue struct {
	vehicles []*Vehicle
	next_empty int
}

func EmptyVehicleQueue() VehicleQueue{
	return VehicleQueue{vehicles: make([]*Vehicle, MAX_VEHICLES), next_empty: 0}
}

func (l Lane) ReplaceNextEmpty() {
	for i:=0;i<MAX_VEHICLES;i++ {
		if l.vehicle_queue.vehicles[i] == nil {
			l.vehicle_queue.next_empty = i
			return
		}
	}
	println("Error: No more space in lane queue!")
}

func (l Lane) FindNextVehicleAhead(v *Vehicle) *Vehicle {
	vehicle_queue := l.vehicle_queue
	progress := v.pos.progress
	closest_progress := 1.1
	var closest_vehicle *Vehicle
	closest_vehicle = nil
	for i:=0;i<MAX_VEHICLES;i++ {
		vehicle_candidate := vehicle_queue.vehicles[i]
		if (vehicle_candidate != nil) && (vehicle_candidate != v) && (progress <= vehicle_candidate.pos.progress) && (vehicle_candidate.pos.progress < closest_progress) {
			closest_progress = vehicle_candidate.pos.progress
			closest_vehicle = vehicle_candidate
		}
	}
	return closest_vehicle
}

func (l Lane) FindNextVehicleAheadFromPos(desired_pos VehiclePosition) *Vehicle {
	vehicle_queue := l.vehicle_queue
	progress := desired_pos.progress
	closest_progress := 1.1
	var closest_vehicle *Vehicle
	closest_vehicle = nil
	for i:=0;i<MAX_VEHICLES;i++ {
		vehicle_candidate := vehicle_queue.vehicles[i]
		if (vehicle_candidate != nil) && (progress <= vehicle_candidate.pos.progress) && (vehicle_candidate.pos.progress < closest_progress) {
			closest_progress = vehicle_candidate.pos.progress
			closest_vehicle = vehicle_candidate
		}
	}
	return closest_vehicle
}

func (l Lane) RemoveVehicleFromQueue(v *Vehicle) {
	vehicle_queue := l.vehicle_queue
	for i:=0;i<MAX_VEHICLES;i++{
		vehicle_candidate := vehicle_queue.vehicles[i]
		if vehicle_candidate == v {
			vehicle_queue.vehicles[i] = nil
			vehicle_queue.next_empty = min(vehicle_queue.next_empty,i) //Change next empty if smaller
			return
		}
	}
	println("Error: Vehicle cannot be deleted! Not found in queue!")
}

func (l Lane) AddVehicleToQueue(v *Vehicle) {
	vehicle_queue := l.vehicle_queue
	if vehicle_queue.vehicles[vehicle_queue.next_empty] != nil {
		println("Error: Vehicle queue next empty is not empty!")
		return
	}
	vehicle_queue.vehicles[vehicle_queue.next_empty] = v
}
