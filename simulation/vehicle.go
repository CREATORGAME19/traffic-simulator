package simulation

import (
	"math"
)

type VehiclePosition struct {
	lane_id    int
	lane_queue *LaneQueue
	progress   float64

	m *Map
}

func NewVehicleNodePos(m *Map, node int) VehiclePosition {
	lane_id := m.nodes[node].lanes_out[0] //TODO: Change this to be more dynamic
	return VehiclePosition{
		lane_id:    lane_id,
		lane_queue: NewLaneQueue(nil, nil), //Vehicle is initialised later in NewVehicle
		progress:   0,
		m:          m,
	}
}

type Vehicle struct {
	id          int
	prop        VehicleProp
	pos         VehiclePosition
	destination int // Node id
	origin      int // Node id
	speed       float64
	acc         float64
	lastFetch   int64
}

type VehicleProp struct {
	max_speed        float64
	max_acc          float64
	minimum_gap_size float64
}

func NewVehicleProp(max_speed float64, max_acc float64, minimum_gap_size float64) VehicleProp {
	return VehicleProp{
		max_speed:        max_speed,
		max_acc:          max_acc,
		minimum_gap_size: minimum_gap_size,
	}
}

func NewVehicle(id int, prop VehicleProp, destination int, origin int, m *Map) *Vehicle {
	res := Vehicle{
		id:          id,
		prop:        prop,
		pos:         NewVehicleNodePos(m, origin),
		destination: destination,
		origin:      origin,
		speed:       0,
		acc:         0,
		lastFetch:   -1,
	}
	res.pos.lane_queue.current_vehicle = &res //Assign previously unassigned Vehicle in NewVehicleNodePos
	return &res
}

func (a *Vehicle) ChangePosition(lane_id int, progress float64) {
	a.pos = VehiclePosition{
		lane_id:  lane_id,
		progress: progress,

		m: a.pos.m,
	}
}

func (a *Vehicle) GetPosXY() Position { //Get position for Vehicle in XY coordinates
	return ConvertVehiclePosToXY(a.pos)
}

func ConvertVehiclePosToXY(vehicle_pos VehiclePosition) Position {
	m := vehicle_pos.m
	lane_id := vehicle_pos.lane_id
	progress := vehicle_pos.progress

	to_coords := Position{x: m.lanes[lane_id].end_pos.x, y: m.lanes[lane_id].end_pos.y}
	from_coords := Position{x: m.lanes[lane_id].start_pos.x, y: m.lanes[lane_id].start_pos.y}

	diff_x := to_coords.x - from_coords.x
	diff_y := to_coords.y - from_coords.y
	return Position{x: from_coords.x + (diff_x * progress), y: from_coords.y + (diff_y * progress)}
}

type VehicleFetchResult struct {
	X          float64
	Y          float64
	Time       int64
	Vehicle_ID int
}

func (a *Vehicle) StartVehicleSim(time int64, vehicle_channel chan VehicleFetchResult) {
	if a == nil {
		return
	}
	//Route vehicle to destination using pathfinding algorithm initially
	curr_pos := a.pos
	mapsim := a.pos.m

	curr_lane := mapsim.lanes[curr_pos.lane_id]
	curr_node := mapsim.nodes[curr_lane.from]
	for _, lane_id := range curr_node.lanes_out {
		lane := mapsim.lanes[lane_id]
		if lane.to == a.destination {
			a.ChangePosition(lane_id, 0)
		}
	}

	//Finish for this turn

	coords := a.GetPosXY()
	a.lastFetch = time
	vehicle_channel <- VehicleFetchResult{X: coords.x, Y: coords.y, Time: a.lastFetch}
}

func (a *Vehicle) FindNextLanePosition() VehiclePosition {
	mapsim := a.pos.m
	intersection_node_id := mapsim.lanes[a.pos.lane_id].to
	lanes_out := mapsim.nodes[intersection_node_id].lanes_out
	if len(lanes_out) <= 0 {
		return VehiclePosition{lane_id: a.pos.lane_id, progress: 1, m: a.pos.m, lane_queue: a.pos.lane_queue}
		//This is reached if vehicle hits a deadend!
	}

	//TODO: Run path finding algorithm here
	new_lane_id := mapsim.nodes[intersection_node_id].lanes_out[0] //Temporary
	return VehiclePosition{lane_id: new_lane_id, progress: 0, m: a.pos.m, lane_queue: a.pos.lane_queue}
}

func (a *Vehicle) CalculateNewPos(old_time int64, new_time int64, pos VehiclePosition) VehiclePosition {
	time_delta := new_time - old_time
	mapsim := pos.m
	if pos.progress == 1 {
		desired_pos := a.FindNextLanePosition()
		if a.pos.lane_queue.next_vehicle == nil && a.isStartLaneFree(desired_pos) {
			a.SwitchToStartLane(desired_pos) //If lane is free to enter, then proceed to the requested position
			a.PopPreviousFromQueue()
			pos = desired_pos
		} else { //Otherwise wait
			a.speed = 0
			a.acc = 0
			return pos
		}
	}
	total_lane_distance := mapsim.lanes[pos.lane_id].distance
	current_distance := pos.progress * total_lane_distance

	//Linear model for acceleration
	new_speed := a.speed + (float64(time_delta) * a.acc)
	distance_travelled := (float64(time_delta) * a.speed) + (0.5 * a.acc * math.Pow(float64(time_delta), 2))

	new_acc := min(a.prop.max_acc, 0.0005) //TODO: Change this to be dynamic

	a.speed = new_speed
	a.acc = new_acc

	new_progress := min((distance_travelled+current_distance)/total_lane_distance, 1)
	return VehiclePosition{lane_id: pos.lane_id, progress: float64(new_progress), m: pos.m, lane_queue: pos.lane_queue}
}

func (a *Vehicle) FetchVehicleSim(time int64, vehicle_channel chan VehicleFetchResult) {
	if a == nil {
		return
	}
	curr_pos := a.pos
	if a.hasReachedDestination() {
		//TODO: Add Deletion/Recycling functionality
	}

	lane_queue := curr_pos.lane_queue
	if lane_queue == nil {
		println("Error: lane_queue stuck!")
		return
	}
	if lane_queue.current_lane == nil {
		if a.isStartLaneFree(a.pos) {
			a.SwitchToStartLane(a.pos)
		} else {
			return
		}
	}
	a.pos = a.CalculateNewPos(a.lastFetch, time, curr_pos)

	//Finish for this turn

	coords := a.GetPosXY()
	a.lastFetch = time
	vehicle_channel <- VehicleFetchResult{X: coords.x, Y: coords.y, Time: a.lastFetch}
}

func (a *Vehicle) SwitchToStartLane(new_pos VehiclePosition) { //Switches LaneQueue to new lane once isStartLaneFree is checked
	lane_queue := a.pos.lane_queue
	lane := &new_pos.m.lanes[new_pos.lane_id]
	lane_queue.current_lane = lane
	lane_queue.next_vehicle = lane.lane_queue
	lane.lane_queue = lane_queue
	if lane_queue.next_vehicle == nil {
		return
	}
	lane_queue.next_vehicle.previous_vehicle = lane_queue
}

func (a *Vehicle) PopPreviousFromQueue() {
	if a.pos.lane_queue.previous_vehicle == nil {
		return
	}
	lane_queue := a.pos.lane_queue
	lane_queue.previous_vehicle.next_vehicle = nil
	lane_queue.previous_vehicle = nil
}

func (a *Vehicle) hasReachedDestination() bool {
	pos := a.pos
	m := pos.m
	lane_id := pos.lane_id
	progress := pos.progress
	dest := a.destination
	return (progress == 1) && (m.lanes[lane_id].to == dest)
}

func (a *Vehicle) isStartLaneFree(desired_pos VehiclePosition) bool {
	min_gap_size := a.prop.minimum_gap_size
	lane := desired_pos.m.lanes[desired_pos.lane_id]
	return (lane.lane_queue == nil) || (CalculateDistance(ConvertVehiclePosToXY(desired_pos), lane.lane_queue.current_vehicle.GetPosXY()) >= min_gap_size)
}
