package simulation

import (
	"fmt"
	"math"
	"math/rand/v2"
)

type VehiclePosition struct {
	lane_id  LaneID
	progress float64
}

func NewVehicleNodePos(node RoadNodeID) VehiclePosition {
	//Sets lane_id to be -1 when initially spawned in. We can add position later on after we check for free space on the lane.
	return VehiclePosition{
		lane_id:  -1,
		progress: 0,
	}
}

type VehicleID int

type Vehicle struct {
	id          VehicleID
	prop        VehicleProp
	pos         VehiclePosition
	destination RoadNodeID
	origin      RoadNodeID
	speed       float64
	acc         float64
	path        *VehiclePath
	lastFetch   SimTime
	spawnTime   SimTime
}

type VehicleProp struct {
	max_speed        float64
	max_acc          float64
	max_decc         float64
	minimum_gap_size float64
	v2v_lookahead    int64
}

func NewVehicleProp(max_speed float64, max_acc float64, max_decc float64, minimum_gap_size float64, v2v_lookahead int64) VehicleProp {
	return VehicleProp{
		max_speed:        max_speed,
		max_acc:          max_acc,
		max_decc:         max_decc,
		minimum_gap_size: minimum_gap_size,
		v2v_lookahead: v2v_lookahead,
	}
}

func NewVehicle(id VehicleID, prop VehicleProp, destination RoadNodeID, origin RoadNodeID, path VehiclePath, time SimTime) *Vehicle {
	res := Vehicle{
		id:          id,
		prop:        prop,
		pos:         NewVehicleNodePos(origin),
		destination: destination,
		origin:      origin,
		speed:       0,
		acc:         0,
		path:        &path,
		lastFetch:   -1,
		spawnTime:   time,
	}
	return &res
}

func (a *Vehicle) CheckAndSetPosition(m *Map, lane_id LaneID, progress float64, time SimTime) bool {
	//Returns false if lane is not free
	desired_pos := VehiclePosition{
		lane_id:  lane_id,
		progress: progress,
	}
	if !a.isLaneFreeAtPos(m, desired_pos) {
		return false
	}
	a.pos = desired_pos
	new_lane := a.GetCurrentLane(m)
	new_lane.AddVehicleToQueue(a, m)
	m.nodes[new_lane.from_node].agent.SendLogMessage(LoggerVehicleEvent{VehicleID: a.id, Time: time, NodeID: new_lane.from_node, LaneID: lane_id, EventType: LoggerVehicleEventLaneIn})
	return true
}

func (a *Vehicle) GetPosXY(m *Map) Position { //Get position for Vehicle in XY coordinates
	return ConvertVehiclePosToXY(m, a.pos)
}

func ConvertVehiclePosToXY(m *Map, vehicle_pos VehiclePosition) Position {
	lane_id := vehicle_pos.lane_id
	progress := vehicle_pos.progress

	to_coords := Position{x: m.lanes[lane_id].end_pos.x, y: m.lanes[lane_id].end_pos.y}
	from_coords := Position{x: m.lanes[lane_id].start_pos.x, y: m.lanes[lane_id].start_pos.y}

	diff_x := to_coords.x - from_coords.x
	diff_y := to_coords.y - from_coords.y
	return Position{x: from_coords.x + (diff_x * progress), y: from_coords.y + (diff_y * progress)}
}

type VehicleStatus int

const (
	NilVehicle       VehicleStatus = 0
	VehicleInTransit VehicleStatus = 1
)

type VehicleFetchResult struct {
	Time      SimTime       `json:"time"`
	ID        VehicleID     `json:"vehicle_id"`
	X         float64       `json:"x"`
	Y         float64       `json:"y"`
	Status    VehicleStatus `json:"status"`
	Speed     float64       `json:"speed"`
	Acc       float64       `json:"acc"`
	Origin    RoadNodeID    `json:"source_node_id"`
	Dest      RoadNodeID    `json:"destination_node_id"`
	SpawnTime SimTime       `json:"spawn_time"`
}

func (a *Vehicle) FindNextLanePosition(mapsim *Map, curr_node_id RoadNodeID) (VehiclePosition, error) {
	lanes_out := mapsim.nodes[curr_node_id].lanes_out
	if len(lanes_out) <= 0 {
		return VehiclePosition{lane_id: a.pos.lane_id, progress: 1}, fmt.Errorf("Error: Vehicle has hit a deadend!")
		//This is reached if vehicle hits a deadend!
	}
	if a.NextNodeVehiclePathExists() && a.GetNextNodeVehiclePath() == curr_node_id {
		a.IncrementVehiclePathIndex()
	}

	if !a.NextNodeVehiclePathExists() {
		if err := a.CalculateVehiclePath(mapsim, curr_node_id); err != nil {
			return VehiclePosition{}, err
		}
	}

	next_node := a.GetNextNodeVehiclePath()
	new_lane_id, err := a.FindLaneWithNextNode(mapsim, next_node, curr_node_id)
	if err != nil {
		return VehiclePosition{}, err
	}

	return VehiclePosition{lane_id: new_lane_id, progress: 0}, nil
}

func (a *Vehicle) FindLaneWithNextNode(mapsim *Map, next_node RoadNodeID, curr_node_id RoadNodeID) (LaneID, error) { //Find the next lane along the vehicle's path
	lanes_out := mapsim.nodes[curr_node_id].lanes_out
	lane_candidates := []LaneID{}
	for i := 0; i < len(lanes_out); i++ {
		if mapsim.lanes[lanes_out[i]].to_node == next_node {
			lane_candidates = append(lane_candidates, lanes_out[i])
		}
	}
	if len(lane_candidates) <= 0 {
		return 0, fmt.Errorf("Error: Could not find next road node for Vehicle on path!")
	}
	return lane_candidates[rand.IntN(len(lane_candidates))], nil
}

func (a *Vehicle) GetCurrentLane(m *Map) *Lane {
	lane_id := a.pos.lane_id
	return &m.lanes[lane_id]
}

func (a *Vehicle) CalculateNewPos(m *Map, old_time SimTime, new_time SimTime, pos VehiclePosition) (VehiclePosition, error) {
	curr_node_id := m.lanes[a.pos.lane_id].to_node
	if a.HasReachedEndLane() {
		vehicle_in_inter_lane := m.nodes[curr_node_id].agent.isVehicleInTransitInterLane(a, m)
		if vehicle_in_inter_lane {
			desired_pos := VehiclePosition{lane_id: m.nodes[curr_node_id].agent.GetInterLaneNextLane(a, m), progress: 0}
			if m.nodes[curr_node_id].agent.HasReachedEndInterLane(a, m) && a.isLaneFreeAtPos(m, desired_pos) {
				m.nodes[curr_node_id].agent.RemoveTransitingVehicle(a, m)
				new_lane := m.lanes[desired_pos.lane_id]
				new_lane.AddVehicleToQueue(a, m)
				pos = desired_pos
			}
		} else {
			desired_pos, err := a.FindNextLanePosition(m, curr_node_id)
			if err != nil {
				return VehiclePosition{}, err
			}
			if a.isLaneFreeAtPos(m, desired_pos) && m.nodes[curr_node_id].agent.CanVehicleProceed(m, new_time, a, desired_pos.lane_id) {
				//If lane is free to enter, then proceed to the requested position
				if CalculateDistance(m.lanes[a.pos.lane_id].end_pos, m.lanes[desired_pos.lane_id].start_pos) > 0 {
					m.nodes[curr_node_id].agent.AddTransitingVehicle(a, m, desired_pos)
					m.lanes[a.pos.lane_id].RemoveVehicleFromQueue(a, m)
				} else {
					a.SwitchToNewLane(m, desired_pos)
					pos = desired_pos
				}
			} else { //Otherwise wait
				a.speed = 0
				a.acc = 0
				return pos, nil
			}
		}
	}
	vehicle_in_inter_lane := m.nodes[curr_node_id].agent.isVehicleInTransitInterLane(a, m)

	new_speed := max(0, a.CalculateSpeed(new_time-old_time))
	distance_travelled := max(0, a.CalculateDistanceTravelled(new_time-old_time))

	curr_lane := a.GetCurrentLane(m)
	if vehicle_in_inter_lane {
		desired_lane_id := m.nodes[curr_node_id].agent.GetInterLaneNextLane(a, m)
		curr_lane = &m.lanes[desired_lane_id]
	}

	var next_vehicle_lane *Vehicle
	var next_vehicle_inter_lane *Vehicle

	if vehicle_in_inter_lane { //Checks if vehicle in inter lane
		next_vehicle_lane = curr_lane.FindNextVehicleAheadFromPos(m, VehiclePosition{progress: 0, lane_id: curr_lane.id})
		next_vehicle_inter_lane = m.nodes[curr_node_id].agent.FindInterLaneNextVehicleAhead(a, m)
	} else {
		next_vehicle_lane = curr_lane.FindNextVehicleAhead(m, a)
	}

	total_lane_distance_inter_lane := 0.0
	total_lane_distance_lane := 0.0
	current_distance := 0.0

	if vehicle_in_inter_lane { //Checks if vehicle in inter lane
		distance, progress := m.nodes[curr_node_id].agent.GetInterLaneDistanceProgress(a, m)
		total_lane_distance_inter_lane = distance
		current_distance = progress * total_lane_distance_inter_lane
	}
	if !vehicle_in_inter_lane || next_vehicle_inter_lane != nil {
		total_lane_distance_lane = curr_lane.distance
		if !vehicle_in_inter_lane {
			current_distance = pos.progress * total_lane_distance_lane
		}
	}

	var next_vehicle *Vehicle
	gap_ahead := max(0, total_lane_distance_lane+total_lane_distance_inter_lane-distance_travelled-current_distance)

	if next_vehicle_lane != nil && (!vehicle_in_inter_lane || next_vehicle_inter_lane == nil) {
		gap_ahead = CalculateDistance(a.GetPosXY(m), next_vehicle_lane.GetPosXY(m))
		next_vehicle = next_vehicle_lane
		if vehicle_in_inter_lane {
			gap_ahead += total_lane_distance_inter_lane
		}
	} else if vehicle_in_inter_lane && next_vehicle_inter_lane != nil {
		next_vehicle_pos := CalculateXYInterLaneVehicle(next_vehicle_inter_lane, m)
		if next_vehicle_pos != nil {
			gap_ahead = CalculateDistance(*CalculateXYInterLaneVehicle(a, m), *next_vehicle_pos)
			next_vehicle = next_vehicle_inter_lane
		}
	}

	a.speed = new_speed

	max_speed := max(a.prop.max_speed, m.lanes[a.pos.lane_id].speed_limit)

	new_acc := a.CalculateAcceleration(m, gap_ahead, next_vehicle, new_time-old_time, max_speed)

	a.acc = new_acc

	new_progress := min((distance_travelled+current_distance)/total_lane_distance_lane, 1)
	if vehicle_in_inter_lane { //Checks if vehicle in inter lane
		new_progress := min((distance_travelled+current_distance)/total_lane_distance_inter_lane, 1)
		m.nodes[curr_node_id].agent.UpdateInterLaneVehicleProgress(a, m, new_progress)
		return VehiclePosition{lane_id: pos.lane_id, progress: 1}, nil //Currently doesn't show movement
	}
	return VehiclePosition{lane_id: pos.lane_id, progress: float64(new_progress)}, nil
}

func (a *Vehicle) CalculateSpeed(time_delta SimTime) float64 { //Linear model for acceleration
	return a.speed + (float64(time_delta) * a.acc)
}

func (a *Vehicle) CalculateDistanceTravelled(time_delta SimTime) float64 { //Linear model for acceleration
	return (float64(time_delta) * a.speed) + (0.5 * a.acc * math.Pow(float64(time_delta), 2))
}

func (a *Vehicle) CalculateAcceleration(m *Map, gap_ahead float64, next_vehicle *Vehicle, time_delta SimTime, max_speed float64) float64 {
	effective_v2v_lookahead := a.prop.v2v_lookahead
	if effective_v2v_lookahead == 0 || (next_vehicle != nil && next_vehicle.prop.v2v_lookahead == 0) {
		effective_v2v_lookahead = 0
	} else if next_vehicle != nil {
		//Fetch next v2v_lookahead vehicles
		effective_v2v_lookahead = a.FindMaxV2VLookAhead(m)
	}
	stopping_gap := max(a.prop.minimum_gap_size, (2/float64(1+effective_v2v_lookahead))*a.speed) //2s stop gap
	new_acc := ((max_speed - a.speed) / max_speed) * a.prop.max_acc
	if next_vehicle != nil {
		vehicle_ahead_next_advance := next_vehicle.CalculateDistanceTravelled(time_delta)
		if gap_ahead < stopping_gap {
			dist_travelled := gap_ahead - stopping_gap - vehicle_ahead_next_advance
			if dist_travelled > 0 {
				new_acc = (math.Pow(next_vehicle.speed, 2) - math.Pow(a.speed, 2)) / (2 * dist_travelled)
			} else {
				new_acc = min(0, ((next_vehicle.speed*0.1)-a.speed)/float64(time_delta))
				//Can be bad if the vehicle behind goes way too fast!
			}
		} else if gap_ahead <= 2.5*stopping_gap {
			dist_travelled := vehicle_ahead_next_advance + gap_ahead - stopping_gap
			new_acc = (math.Pow(next_vehicle.speed, 2) - math.Pow(a.speed, 2)) / (2 * dist_travelled)
		}
	} else if AlmostEqual(gap_ahead, 0) {
		new_acc = 0
	} else if gap_ahead < 2.5*stopping_gap {
		new_acc = (-math.Pow(a.speed, 2)) / (2 * gap_ahead)
		if a.speed < 2 {
			new_acc = max(new_acc, (2-a.speed)/float64(time_delta))
		}
	}
	return max(min(new_acc, a.prop.max_acc), a.prop.max_decc)
}

func (a *Vehicle) FetchVehicleSim(mapsim *Map, time SimTime, vehicle_channel chan VehicleFetchResult, id VehicleID) error {
	if a == nil {
		vehicle_channel <- VehicleFetchResult{X: 0, Y: 0, Time: time, ID: id, Status: NilVehicle}
		return nil
	}
	curr_pos := a.pos
	if curr_pos.lane_id < 0 { //Case when Vehicle not initialised
		origin_node_id := a.origin
		curr_node := mapsim.nodes[origin_node_id]
		new_pos, err := a.FindNextLanePosition(mapsim, curr_node.id)
		if err != nil {
			fmt.Println(err)
			return err
		}
		a.CheckAndSetPosition(mapsim, new_pos.lane_id, 0, time)

		coords := curr_node.pos
		a.lastFetch = time
		vehicle_channel <- VehicleFetchResult{X: coords.x, Y: coords.y, Time: a.lastFetch, ID: a.id, Status: VehicleInTransit}
		return nil
	}

	new_pos, err := a.CalculateNewPos(mapsim, a.lastFetch, time, curr_pos)
	if err != nil {
		fmt.Println(err)
		return err
	}

	a.pos = new_pos

	//Finish for this turn

	coords := a.GetPosXY(mapsim)

	if a.CheckVehicleInInterLane(mapsim) { //Calculate Inter-lane XY and update coords
		coords = *CalculateXYInterLaneVehicle(a, mapsim)
	}

	a.lastFetch = time

	if a.hasReachedDestination(mapsim) {
		lane := a.GetCurrentLane(mapsim)
		sink := mapsim.nodes[lane.to_node].agent
		sink.DestroyVehicle(mapsim, time, a)
	}

	vehicle_channel <- VehicleFetchResult{X: coords.x, Y: coords.y, Time: a.lastFetch, ID: a.id, Status: VehicleInTransit, Speed: a.speed, Acc: a.acc, Origin: a.origin, Dest: a.destination, SpawnTime: a.spawnTime}
	return nil
}

func (a *Vehicle) SwitchToNewLane(m *Map, new_pos VehiclePosition) {
	new_lane := m.lanes[new_pos.lane_id]
	new_lane.AddVehicleToQueue(a, m)
	old_lane := a.GetCurrentLane(m)
	old_lane.RemoveVehicleFromQueue(a, m)
}

func (a *Vehicle) hasReachedDestination(m *Map) bool {
	pos := a.pos
	lane_id := pos.lane_id
	dest := a.destination
	return a.HasReachedEndLane() && (m.lanes[lane_id].to_node == dest)
}

func (a *Vehicle) isLaneFreeAtPos(m *Map, desired_pos VehiclePosition) bool {
	min_gap_size := a.prop.minimum_gap_size
	lane := m.lanes[desired_pos.lane_id]
	next_vehicle := lane.FindNextVehicleAheadFromPos(m, desired_pos)
	return (next_vehicle == nil) || (CalculateDistance(ConvertVehiclePosToXY(m, desired_pos), next_vehicle.GetPosXY(m)) >= min_gap_size)
}

func AlmostEqual(a float64, b float64) bool {
	return math.Abs(a-b) <= 0.00000001
}

func (a *Vehicle) HasReachedEndLane() bool {
	return AlmostEqual(a.pos.progress, 1)
}

type VehiclePath struct {
	path_array *[]RoadNodeID
	curr_index int
}

func CreateVehiclePath(path_array *[]RoadNodeID) VehiclePath {
	return VehiclePath{path_array: path_array, curr_index: 0}
}

func (a *Vehicle) AddVehiclePath(path_array []RoadNodeID) {
	*a.path.path_array = append(*a.path.path_array, path_array...)
}

func (a *Vehicle) IncrementVehiclePathIndex() {
	a.path.curr_index++
}

func (a *Vehicle) NextNodeVehiclePathExists() bool {
	return len(*a.path.path_array) > a.path.curr_index
}

func (a *Vehicle) GetNextNodeVehiclePath() RoadNodeID {
	return (*a.path.path_array)[a.path.curr_index]
}

func (a *Vehicle) CheckVehicleInInterLane(m *Map) bool {
	curr_node_id := m.lanes[a.pos.lane_id].to_node
	return m.nodes[curr_node_id].agent.isVehicleInTransitInterLane(a, m)
}

func (a *Vehicle) FindMaxV2VLookAhead(m *Map) int64 {
	curr_lane := a.GetCurrentLane(m)
	vehicle_ahead := curr_lane.FindNextVehicleAhead(m, a)
	if vehicle_ahead == nil {
		return 0
	}
	curr_vehicle := vehicle_ahead
	max_i := 1
	for i := 2; int64(i) <= a.prop.v2v_lookahead; i++ {
		vehicle_ahead = curr_lane.FindNextVehicleAhead(m, curr_vehicle)
		if vehicle_ahead == nil {
			return int64(max_i)
		}
		if vehicle_ahead.prop.v2v_lookahead > 0 {
			max_i = i
		}
	}
	return int64(max_i)
}