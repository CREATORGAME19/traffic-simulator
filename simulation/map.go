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

type Node struct { //TODO: Change this to distinguish between intersection node and spawn/sink node
	id        int
	pos       Position
	lanes_out []int
	lanes_in  []int
}

func NewNode(id int, pos Position, lanes_out []int, lanes_in []int) Node {
	return Node{id: id, pos: pos, lanes_out: lanes_out, lanes_in: lanes_in}
}

type Lane struct {
	id        int
	start_pos Position
	end_pos   Position
	from      int //Node id
	to        int //Node id
	distance  float64
	vehicles  []*Vehicle //TODO: Implement vehicle lane queuing (use linked list approach)
	// more Lane Properties here (like speed limit)
}

func NewLane(id int, start_pos Position, end_pos Position, from int, to int) Lane {
	return Lane{id: id, start_pos: start_pos, end_pos: end_pos, from: from, to: to, distance: CalculateDistance(start_pos, end_pos), vehicles: []*Vehicle{}}
}

func CalculateDistance(p1 Position, p2 Position) float64 {
	change_x := p1.x - p2.x
	change_y := p1.y - p2.y
	return math.Sqrt(math.Pow(change_x, 2) + math.Pow(change_y, 2))
}

type Map struct {
	nodes []Node
	lanes []Lane
}

func InitialiseMap(v []Node, l []Lane) *Map {
	return &Map{
		nodes: v,
		lanes: l,
	}
}

func (m *Map) AddNode(v Node) {
	m.nodes = append(m.nodes, v)
}

func (m *Map) AddLane(l Lane) {
	m.lanes = append(m.lanes, l)
}
