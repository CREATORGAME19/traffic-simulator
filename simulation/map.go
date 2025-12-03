package simulation

import "math"

type Position struct {
	x float64
	y float64
}

func NewPosition(x float64, y float64) Position {
	return Position{x: x, y: y}
}

type Vertex struct {
	pos       Position
	links_out []int
}

func NewVertex(pos Position, links []int) Vertex {
	return Vertex{pos: pos, links_out: links}
}

type Link struct {
	from     int
	to       int
	distance float64
	// Link Properties here
}

func NewLink(from int, to int, distance float64) Link {
	return Link{from: from, to: to, distance: distance}
}

func CalculateDistance(p1 Position, p2 Position) float64 {
	change_x := p1.x - p2.x
	change_y := p1.y - p2.y
	return math.Sqrt(math.Pow(change_x, 2) + math.Pow(change_y, 2))
}

type Map struct {
	vertices []Vertex
	links    []Link
}

func InitialiseMap(v []Vertex, l []Link) *Map {
	return &Map{
		vertices: v,
		links:    l,
	}
}

func (m *Map) AddVertex(v Vertex) {
	m.vertices = append(m.vertices, v)
}

func (m *Map) AddLink(l Link) {
	m.links = append(m.links, l)
}
