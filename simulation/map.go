package simulation

type Position struct {
	x int
	y int
}

func NewPosition(x int, y int) Position {
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
	next int
	// Link Properties here
}

func NewLink(next int) Link {
	return Link{next: next}
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
