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
	SpawnerRoadNode RoadNodeType = "spawner"
	SinkRoadNode RoadNodeType = "sink"
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
	id        int
	start_pos Position
	end_pos   Position
	from      int //Node id
	to        int //Node id
	distance  float64
	lane_queue  *LaneQueue 
	// more Lane Properties here (like speed limit)
}

func NewLane(id int, start_pos Position, end_pos Position, from int, to int) Lane {
	return Lane{id: id, start_pos: start_pos, end_pos: end_pos, from: from, to: to, distance: CalculateDistance(start_pos, end_pos), lane_queue: nil}
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

type LaneQueue struct {
	current_vehicle *Vehicle
	current_lane *Lane
	next_vehicle *LaneQueue
	previous_vehicle *LaneQueue
}

func NewLaneQueue(current_vehicle *Vehicle, current_lane *Lane) *LaneQueue {
	return &LaneQueue{
		current_vehicle: current_vehicle,
		current_lane: current_lane,
		next_vehicle: nil,
		previous_vehicle: nil,
	}
}