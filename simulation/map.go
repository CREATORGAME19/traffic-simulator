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

type RoadNodeID int

type RoadNode struct {
	id        RoadNodeID
	pos       Position
	lanes_out []LaneID
	lanes_in  []LaneID
	radius    float64
	agent     StaticAgent
}

func NewRoadNode(id RoadNodeID, pos Position, lanes_out []LaneID, lanes_in []LaneID, radius float64, agent StaticAgent) RoadNode {
	return RoadNode{id: id, pos: pos, lanes_out: lanes_out, lanes_in: lanes_in, radius: radius, agent: agent}
}

type LaneID int

type Lane struct {
	id            LaneID
	start_pos     Position
	end_pos       Position
	from_node     RoadNodeID
	to_node       RoadNodeID
	distance      float64
	vehicle_queue *VehicleLaneQueue

	// more Lane Properties here (like speed limit)
}

func NewLane(id LaneID, start_pos Position, end_pos Position, from RoadNodeID, to RoadNodeID, config_parameters *ConfigParameters) Lane {
	return Lane{id: id, start_pos: start_pos, end_pos: end_pos, from_node: from, to_node: to, distance: CalculateDistance(start_pos, end_pos), vehicle_queue: EmptyVehicleLaneQueue(config_parameters)}
}

func CalculateDistance(p1 Position, p2 Position) float64 {
	change_x := p1.x - p2.x
	change_y := p1.y - p2.y
	return math.Sqrt(math.Pow(change_x, 2) + math.Pow(change_y, 2))
}

type VehicleDB struct {
	vehicle_array          []*Vehicle
	next_empty             int
	vehicle_id_index_array []int
}

func FindNextEmptyVehicles(mapsim *Map) int {
	for i := 0; i < mapsim.config_parameters.MAX_VEHICLES; i++ {
		if mapsim.vehicles.vehicle_array[i] == nil {
			return i
		}
	}
	return mapsim.config_parameters.MAX_VEHICLES
}

type Map struct {
	nodes             []RoadNode
	lanes             []Lane
	vehicles          VehicleDB
	config_parameters *ConfigParameters
}

func InitialiseMap(v []RoadNode, l []Lane, config_parameters *ConfigParameters) *Map {
	return &Map{
		nodes:             v,
		lanes:             l,
		vehicles:          VehicleDB{vehicle_array: make([]*Vehicle, config_parameters.MAX_VEHICLES), next_empty: 0, vehicle_id_index_array: make([]int, config_parameters.MAX_VEHICLES)},
		config_parameters: config_parameters,
	}
}

func (m *Map) AddNode(v RoadNode) {
	m.nodes = append(m.nodes, v)
}

func (m *Map) AddLane(l Lane) {
	m.lanes = append(m.lanes, l)
}

func (m *Map) FindADestinationNode() RoadNodeID {
	dest_nodes := []int{}
	for n := 0; n < len(m.nodes); n++ {
		if m.nodes[n].agent.Descriptor() == SinkAgentType {
			dest_nodes = append(dest_nodes, n)
		}
	}
	return RoadNodeID(dest_nodes[rand.IntN(len(dest_nodes))])
}

type VehicleLaneQueue struct {
	vehicles []*Vehicle
	usage    int64
}

func EmptyVehicleLaneQueue(config_parameters *ConfigParameters) *VehicleLaneQueue {
	return &VehicleLaneQueue{vehicles: make([]*Vehicle, config_parameters.MAX_VEHICLES), usage: 0}
}

func (l *Lane) FindNextVehicleAhead(m *Map, v *Vehicle) *Vehicle {
	progress := v.pos.progress
	closest_progress := 1.1
	var closest_vehicle *Vehicle
	closest_vehicle = nil
	for i := 0; i < m.config_parameters.MAX_VEHICLES; i++ {
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

func (l *Lane) FindNextVehicleAheadFromPos(m *Map, desired_pos VehiclePosition) *Vehicle {
	progress := desired_pos.progress
	closest_progress := 1.1
	var closest_vehicle *Vehicle
	closest_vehicle = nil
	for i := 0; i < m.config_parameters.MAX_VEHICLES; i++ {
		vehicle_candidate := l.vehicle_queue.vehicles[i]
		if (vehicle_candidate != nil) && (progress <= vehicle_candidate.pos.progress) && (vehicle_candidate.pos.progress < closest_progress) {
			closest_progress = vehicle_candidate.pos.progress
			closest_vehicle = vehicle_candidate
		}
	}
	return closest_vehicle
}

func (l *Lane) RemoveVehicleFromQueue(v *Vehicle, m *Map) {
	l.vehicle_queue.vehicles[m.GetMapArrayVehicleIDIndex(v.id)] = nil
	l.vehicle_queue.usage -= 1
}

func (l *Lane) AddVehicleToQueue(v *Vehicle, m *Map) {
	if l.vehicle_queue.vehicles[m.GetMapArrayVehicleIDIndex(v.id)] != nil {
		fmt.Println("Error: Vehicle queue", v.id, "is not empty!")
		return
	}
	l.vehicle_queue.vehicles[m.GetMapArrayVehicleIDIndex(v.id)] = v
	l.vehicle_queue.usage++
}

func (m *Map) GetMapArrayVehicleIDIndex(v VehicleID) int {
	return int(v) % m.config_parameters.MAX_VEHICLES
}

func (m *Map) GetRealVehicleID(id int) VehicleID {
	return VehicleID(id + (m.vehicles.vehicle_id_index_array[id] * m.config_parameters.MAX_VEHICLES))
}