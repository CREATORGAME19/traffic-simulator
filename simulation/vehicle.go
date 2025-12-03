package simulation

type VehiclePosition struct { // Vehicle is either on a link or on a vertex
	link_id  int
	progress float64

	m *Map
}

type XYCoords struct {
	x float64
	y float64
}

func NewVehicleVertexPos(m *Map, vertex int) VehiclePosition {
	return VehiclePosition{
		link_id:  m.vertices[vertex].links_out[0],
		progress: 0,
		m:        m,
	}
}

type Vehicle struct {
	id          int
	prop        VehicleProp
	pos         VehiclePosition
	destination int // Vertex id
	origin      int // Vertex id
	speed       float64
	acc         float64
	lastFetch   int64
}

type VehicleProp struct {
	max_speed float64
	max_acc   float64
}

func NewVehicleProp(max_speed float64, max_acc float64) VehicleProp {
	return VehicleProp{
		max_speed: max_speed,
		max_acc:   max_acc,
	}
}

func NewVehicle(id int, prop VehicleProp, destination int, origin int, m *Map) *Vehicle {
	return &Vehicle{
		id:          id,
		prop:        prop,
		pos:         NewVehicleVertexPos(m, origin),
		destination: destination,
		origin:      origin,
		speed:       0,
		acc:         0,
		lastFetch:   -1,
	}
}

func (a *Vehicle) ChangePosition(link_id int, progress float64) {
	a.pos = VehiclePosition{
		link_id:  link_id,
		progress: progress,

		m: a.pos.m,
	}
}

func (a *Vehicle) GetPosXY() XYCoords { //Get position for Vehicle in XY coordinates
	m := a.pos.m
	link_id := a.pos.link_id
	progress := a.pos.progress

	vertex_id_to := m.links[link_id].to
	vertex_id_from := m.links[link_id].from

	vertex_to_coords := XYCoords{x: m.vertices[vertex_id_to].pos.x, y: m.vertices[vertex_id_to].pos.y}
	vertex_from_coords := XYCoords{x: m.vertices[vertex_id_from].pos.x, y: m.vertices[vertex_id_from].pos.y}

	diff_x := vertex_to_coords.x - vertex_from_coords.x
	diff_y := vertex_to_coords.y - vertex_from_coords.y
	return XYCoords{x: vertex_from_coords.x + (diff_x * progress), y: vertex_from_coords.y + (diff_y * progress)}
}

type VehicleFetchResult struct {
	X          float64
	Y          float64
	Time       int64
	Vehicle_ID int
}

func (a *Vehicle) StartVehicleSim(time int64, vehicle_channel chan VehicleFetchResult) {
	//Route vehicle to destination using pathfinding algorithm initially
	curr_pos := a.pos
	mapsim := a.pos.m

	curr_link := mapsim.links[curr_pos.link_id]
	curr_vertex := mapsim.vertices[curr_link.from]
	for _, link_id := range curr_vertex.links_out {
		link := mapsim.links[link_id]
		if link.to == a.destination {
			a.ChangePosition(link_id, 0)
			a.acc = 0.01
		}
	}

	//Finish for this turn

	coords := a.GetPosXY()
	a.lastFetch = time
	vehicle_channel <- VehicleFetchResult{X: coords.x, Y: coords.y, Time: a.lastFetch}
}

func (a *Vehicle) CalculateNewPos(old_time int64, new_time int64, pos VehiclePosition) VehiclePosition {
	time_delta := new_time - old_time
	mapsim := pos.m
	total_link_distance := mapsim.links[pos.link_id].distance
	current_distance := pos.progress * total_link_distance

	distance_travelled := float64(time_delta) / 100 //Constant velocity of 0.1m/s
	new_progress := min((distance_travelled+current_distance)/total_link_distance, 1)
	return VehiclePosition{link_id: pos.link_id, progress: float64(new_progress), m: pos.m}
}

func (a *Vehicle) FetchVehicleSim(time int64, vehicle_channel chan VehicleFetchResult) {
	curr_pos := a.pos

	a.pos = a.CalculateNewPos(a.lastFetch, time, curr_pos)

	//Finish for this turn

	coords := a.GetPosXY()
	a.lastFetch = time
	vehicle_channel <- VehicleFetchResult{X: coords.x, Y: coords.y, Time: a.lastFetch}
}
