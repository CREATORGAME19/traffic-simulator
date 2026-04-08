package simulation

import (
	"fmt"
	"math/rand"
)

type StaticAgent interface {
	Poke(mapsim *Map, time SimTime, agent_channel chan StaticAgentFetchResult)
	Descriptor() StaticAgentType
	DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle)
	CanVehicleProceed(mapsim *Map, time SimTime, vehicle *Vehicle, desired_lane LaneID) bool

	//Inter-Lane functions
	AddTransitingVehicle(v *Vehicle, m *Map, desired_pos VehiclePosition)
	RemoveTransitingVehicle(v *Vehicle, m *Map)
	isVehicleInTransitInterLane(v *Vehicle, m *Map) bool
	HasReachedEndInterLane(v *Vehicle, m *Map) bool
	GetInterLaneNextLane(v *Vehicle, m *Map) LaneID
	GetInterLaneDistanceProgress(v *Vehicle, m *Map) (float64, float64)
	FindInterLaneNextVehicleAhead(v *Vehicle, m *Map) *Vehicle
	UpdateInterLaneVehicleProgress(v *Vehicle, m *Map, new_progress float64)
}

type StaticAgentFetchResult struct {
	Time              SimTime    `json:"time"`
	ID                RoadNodeID `json:"road_node_id"`
	VehiclesProcessed int        `json:"vehicles_processed"`
}

type StaticAgentType int

const (
	IntersectionAgentType             StaticAgentType = 0
	SpawnerAgentType                  StaticAgentType = 1
	SinkAgentType                     StaticAgentType = 2
	TrafficLightIntersectionAgentType StaticAgentType = 3
)

const spawn_stdDev = 120 //Standard Deviation of spawn times

func NewStaticAgent(node_id RoadNodeID, agent_type StaticAgentType, agent_prop map[string]any, vehicle_log_channel chan LoggerVehicleEvent, config_parameters *ConfigParameters) (StaticAgent, error) {
	switch agent_type {
	case IntersectionAgentType:
		return NewIntersectionAgent(node_id, &IntersectionAgentParams{}, vehicle_log_channel, config_parameters), nil
	case SpawnerAgentType:
		param, ok := agent_prop["spawn_rate"]
		if !ok {
			return nil, fmt.Errorf("Error: Spawner Agent Param expects 'spawn_rate' as a parameter. 'spawn_rate' not found as valid param!")
		}
		spawn_rate, ok := param.(float64)
		if !ok {
			return nil, fmt.Errorf("Error: Spawner Agent Param expects 'spawn_rate' as a float64 parameter.")
		}
		return NewSpawnerAgent(node_id, &SpawnerAgentParams{spawn_rate: spawn_rate, nextSpawnTime: SimTime(0.5 + (spawn_stdDev * rand.NormFloat64()))}, vehicle_log_channel, config_parameters), nil
	case SinkAgentType:
		return NewSinkAgent(node_id, &SinkAgentParams{}, vehicle_log_channel, config_parameters), nil
	case TrafficLightIntersectionAgentType:
		param, ok := agent_prop["time_intervals"]
		if !ok {
			return nil, fmt.Errorf("Error: Traffic Light Intersection Agent Param expects 'time_intervals' as a parameter. 'time_intervals' not found as valid param!")
		}
		raw_time_intervals, ok := param.([]interface{})
		if !ok {
			return nil, fmt.Errorf("Error: Traffic Light Intersection Agent Param expects 'time_intervals' as an array parameter.")
		}
		time_intervals := make([]SimTime, len(raw_time_intervals))
		for i, raw_v := range raw_time_intervals {
			v, ok := raw_v.(float64)
			if !ok {
				return nil, fmt.Errorf("Error: Traffic Light Intersection Agent Param expects 'time_intervals' as an array parameter of type SimTime.")
			}
			time_intervals[i] = SimTime(v)
		}
		return NewTrafficLightIntersectionAgent(node_id, &TrafficLightIntersectionAgentParams{time_intervals: time_intervals, time_interval_index: 0, time_interval_lastChange: 0}, vehicle_log_channel, config_parameters), nil
	default:
		return nil, fmt.Errorf("Error: Unknown Static Agent Type!")
	}
}

type InterLaneVehicleTransit struct {
	vehicle_id VehicleID
	start_lane LaneID
	end_lane   LaneID
	distance   float64
	progress   float64
}

func GetInterLaneNextVehicle(vehicle_transit_array *[]*InterLaneVehicleTransit, start_lane LaneID, end_lane LaneID, progress float64, m *Map) VehicleID {
	nearest_progress := 1.1
	nearest_vehicle_id := VehicleID(-1)

	for _, transit := range *vehicle_transit_array {
		if transit != nil && transit.start_lane == start_lane && transit.end_lane == end_lane && transit.progress > progress && transit.progress < nearest_progress {
			nearest_vehicle_id = transit.vehicle_id
			nearest_progress = transit.progress
		}
	}
	return nearest_vehicle_id
}

func CalculateXYInterLaneVehicle(vehicle *Vehicle, mapsim *Map) *Position {
	curr_node_id := mapsim.lanes[vehicle.pos.lane_id].to_node
	if !mapsim.nodes[curr_node_id].agent.isVehicleInTransitInterLane(vehicle, mapsim) {
		return nil
	}
	_, progress := mapsim.nodes[curr_node_id].agent.GetInterLaneDistanceProgress(vehicle, mapsim)
	start_lane_id := vehicle.pos.lane_id
	end_lane_id := mapsim.nodes[curr_node_id].agent.GetInterLaneNextLane(vehicle, mapsim)

	//Interpolate using progress
	dx := mapsim.lanes[end_lane_id].start_pos.x - mapsim.lanes[start_lane_id].end_pos.x
	dy := mapsim.lanes[end_lane_id].start_pos.y - mapsim.lanes[start_lane_id].end_pos.y

	new_x := mapsim.lanes[start_lane_id].end_pos.x + progress*dx
	new_y := mapsim.lanes[start_lane_id].end_pos.y + progress*dy

	return &Position{x: new_x, y: new_y}
}

func isVehicleInterLaneConflicting(vehicle_transit_array *[]*InterLaneVehicleTransit, desired_lane LaneID, vehicle *Vehicle, mapsim *Map) bool {
	for _, vta := range *vehicle_transit_array {
		if vta != nil && VehicleInterLanesIntersect(vehicle.pos.lane_id, desired_lane, vta.start_lane, vta.end_lane, mapsim) {
			return true
		}
	}

	return false
}

func VehicleInterLanesIntersect(start_lane1 LaneID, end_lane1 LaneID, start_lane2 LaneID, end_lane2 LaneID, mapsim *Map) bool {
	if start_lane1 == start_lane2 && end_lane1 == end_lane2 { //We allow vehicles if they are in the same direction
		return false
	}
	start_pos1 := mapsim.lanes[start_lane1].end_pos
	end_pos1 := mapsim.lanes[end_lane1].start_pos
	start_pos2 := mapsim.lanes[start_lane2].end_pos
	end_pos2 := mapsim.lanes[end_lane2].start_pos

	grad1 := (end_pos1.y - start_pos1.y) / (end_pos1.x - start_pos1.x)
	grad2 := (end_pos2.y - start_pos2.y) / (end_pos2.x - start_pos2.x)

	c1 := start_pos1.y - grad1*start_pos1.x
	c2 := start_pos2.y - grad2*start_pos2.x

	if grad1 == grad2 && c1 != c2 { //Parallel lanes have no intersection
		return false
	}

	x_intersect := (c1 - c2) / (grad2 - grad1)
	y_intersect := grad1*x_intersect + c1

	if (x_intersect < min(start_pos1.x, end_pos1.x)) || (x_intersect > max(start_pos1.x, end_pos1.x)) || (y_intersect < min(start_pos1.y, end_pos1.y)) || (y_intersect > max(start_pos1.y, end_pos1.y)) {
		return true
	}
	if (x_intersect < min(start_pos2.x, end_pos2.x)) || (x_intersect > max(start_pos2.x, end_pos2.x)) || (y_intersect < min(start_pos2.y, end_pos2.y)) || (y_intersect > max(start_pos2.y, end_pos2.y)) {
		return true
	}

	if end_pos2 == end_pos1 {
		return true
	}

	return false
}

type IntersectionAgent struct {
	road_node_id           RoadNodeID
	num_vehicles_processed int
	vehicle_log_channel    chan LoggerVehicleEvent
	vehicle_transit_array  []*InterLaneVehicleTransit
	params                 *IntersectionAgentParams
}

type IntersectionAgentParams struct {
}

func NewIntersectionAgent(node_id RoadNodeID, params *IntersectionAgentParams, vehicle_log_channel chan LoggerVehicleEvent, config_parameters *ConfigParameters) *IntersectionAgent {
	return &IntersectionAgent{road_node_id: node_id, num_vehicles_processed: 0, vehicle_log_channel: vehicle_log_channel, vehicle_transit_array: make([]*InterLaneVehicleTransit, config_parameters.MAX_VEHICLES), params: params}
}

func (a *IntersectionAgent) Descriptor() StaticAgentType {
	return IntersectionAgentType
}

func (a *IntersectionAgent) Poke(mapsim *Map, time SimTime, agent_channel chan StaticAgentFetchResult) {
	agent_channel <- StaticAgentFetchResult{ID: a.road_node_id, Time: time, VehiclesProcessed: a.num_vehicles_processed}
	a.num_vehicles_processed = 0
	return
}

func (a *IntersectionAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) { //Only Sink nodes destroy vehicles
	return
}

func (a *IntersectionAgent) CanVehicleProceed(mapsim *Map, time SimTime, vehicle *Vehicle, desired_lane LaneID) bool {
	a.num_vehicles_processed++

	//Check whether InterLane is free to proceed
	next_vehicle_id := GetInterLaneNextVehicle(&a.vehicle_transit_array, vehicle.pos.lane_id, desired_lane, 0, mapsim)
	if next_vehicle_id >= 0 {
		next_vehicle_info := a.vehicle_transit_array[mapsim.GetMapArrayVehicleIDIndex(next_vehicle_id)]
		if next_vehicle_info.progress*next_vehicle_info.distance < vehicle.prop.minimum_gap_size {
			return false
		}
	}

	if isVehicleInterLaneConflicting(&a.vehicle_transit_array, desired_lane, vehicle, mapsim) {
		return false
	}

	a.vehicle_log_channel <- LoggerVehicleEvent{VehicleID: vehicle.id, Time: time, NodeID: a.road_node_id, LaneID: vehicle.pos.lane_id, EventType: LoggerVehicleEventLaneOut}
	a.vehicle_log_channel <- LoggerVehicleEvent{VehicleID: vehicle.id, Time: time, NodeID: a.road_node_id, LaneID: desired_lane, EventType: LoggerVehicleEventLaneIn}

	return true
}

func (a *IntersectionAgent) AddTransitingVehicle(v *Vehicle, m *Map, desired_pos VehiclePosition) {
	start_lane_id := v.pos.lane_id
	end_lane_id := desired_pos.lane_id
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] = &InterLaneVehicleTransit{vehicle_id: v.id, start_lane: start_lane_id, end_lane: end_lane_id, distance: CalculateDistance(m.lanes[start_lane_id].end_pos, m.lanes[end_lane_id].start_pos), progress: 0}
}

func (a *IntersectionAgent) RemoveTransitingVehicle(v *Vehicle, m *Map) {
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] = nil
}

func (a *IntersectionAgent) isVehicleInTransitInterLane(v *Vehicle, m *Map) bool {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] != nil
}

func (a *IntersectionAgent) HasReachedEndInterLane(v *Vehicle, m *Map) bool {
	return AlmostEqual(a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress, 1) || (a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress > 1)
}

func (a *IntersectionAgent) GetInterLaneNextLane(v *Vehicle, m *Map) LaneID {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].end_lane
}

func (a *IntersectionAgent) GetInterLaneDistanceProgress(v *Vehicle, m *Map) (float64, float64) {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].distance, a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress
}

func (a *IntersectionAgent) FindInterLaneNextVehicleAhead(v *Vehicle, m *Map) *Vehicle {
	start := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].start_lane
	end := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].end_lane
	progress := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress

	nearest_progress := 1.1
	nearest_map_vehicle_id := -1

	for map_vehicle_id, transit := range a.vehicle_transit_array {
		if transit != nil && transit.start_lane == start && transit.end_lane == end && transit.progress > progress && transit.progress < nearest_progress {
			nearest_map_vehicle_id = map_vehicle_id
			nearest_progress = transit.progress
		}
	}
	if nearest_map_vehicle_id == -1 {
		return nil
	}
	return m.vehicles.vehicle_array[nearest_map_vehicle_id]
}

func (a *IntersectionAgent) UpdateInterLaneVehicleProgress(v *Vehicle, m *Map, new_progress float64) {
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress = new_progress
}

type SpawnerAgent struct {
	road_node_id           RoadNodeID
	num_vehicles_processed int
	vehicle_log_channel    chan LoggerVehicleEvent
	vehicle_transit_array  []*InterLaneVehicleTransit
	params                 *SpawnerAgentParams
}

type SpawnerAgentParams struct {
	spawn_rate    float64
	nextSpawnTime SimTime
}

func NewSpawnerAgent(node_id RoadNodeID, params *SpawnerAgentParams, vehicle_log_channel chan LoggerVehicleEvent, config_parameters *ConfigParameters) *SpawnerAgent {
	return &SpawnerAgent{road_node_id: node_id, num_vehicles_processed: 0, vehicle_log_channel: vehicle_log_channel, vehicle_transit_array: make([]*InterLaneVehicleTransit, config_parameters.MAX_VEHICLES), params: params}
}

func (a *SpawnerAgent) Descriptor() StaticAgentType {
	return SpawnerAgentType
}

func (a *SpawnerAgent) Poke(mapsim *Map, time SimTime, agent_channel chan StaticAgentFetchResult) {
	a.SpawnVehicles(mapsim, time)
	agent_channel <- StaticAgentFetchResult{ID: a.road_node_id, Time: time, VehiclesProcessed: a.num_vehicles_processed}
	a.num_vehicles_processed = 0
}

func (a *SpawnerAgent) SpawnVehicles(mapsim *Map, time SimTime) {
	if mapsim.vehicles.next_empty >= mapsim.config_parameters.MAX_VEHICLES {
		//println("Vehicle limit reached!") //WARNING
		return
	}

	if time >= a.params.nextSpawnTime {
		if mapsim.vehicles.next_empty >= mapsim.config_parameters.MAX_VEHICLES {
			//println("Vehicle limit reached!") //WARNING
			return
		}
		next_vehicle_id := mapsim.GetRealVehicleID(mapsim.vehicles.next_empty)
		mapsim.vehicles.vehicle_array[mapsim.vehicles.next_empty] = CreateVehicle(VehicleID(next_vehicle_id), time, a.road_node_id, mapsim.FindADestinationNode())
		mapsim.vehicles.vehicle_id_index_array[mapsim.vehicles.next_empty]++
		mapsim.vehicles.next_empty = FindNextEmptyVehicles(mapsim)
		a.params.nextSpawnTime = time + SimTime(1/a.params.spawn_rate) + SimTime(spawn_stdDev*rand.NormFloat64())
		a.num_vehicles_processed++
		a.vehicle_log_channel <- LoggerVehicleEvent{VehicleID: next_vehicle_id, NodeID: a.road_node_id, Time: time, LaneID: -1, EventType: LoggerVehicleEventSpawn} //Technically vehicle is not yet on a lane upon spawn
	}
}

func (a *SpawnerAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) { //Only Sink nodes destroy vehicles
	return
}

func (a *SpawnerAgent) CanVehicleProceed(mapsim *Map, time SimTime, vehicle *Vehicle, desired_lane LaneID) bool {
	a.num_vehicles_processed++

	//Check whether InterLane is free to proceed
	next_vehicle_id := GetInterLaneNextVehicle(&a.vehicle_transit_array, vehicle.pos.lane_id, desired_lane, 0, mapsim)
	if next_vehicle_id >= 0 {
		next_vehicle_info := a.vehicle_transit_array[mapsim.GetMapArrayVehicleIDIndex(next_vehicle_id)]
		if next_vehicle_info.progress*next_vehicle_info.distance < vehicle.prop.minimum_gap_size {
			return false
		}
	}

	if isVehicleInterLaneConflicting(&a.vehicle_transit_array, desired_lane, vehicle, mapsim) {
		return false
	}

	a.vehicle_log_channel <- LoggerVehicleEvent{VehicleID: vehicle.id, NodeID: a.road_node_id, Time: time, LaneID: vehicle.pos.lane_id, EventType: LoggerVehicleEventLaneOut}
	a.vehicle_log_channel <- LoggerVehicleEvent{VehicleID: vehicle.id, NodeID: a.road_node_id, Time: time, LaneID: desired_lane, EventType: LoggerVehicleEventLaneIn}

	return true
}

func (a *SpawnerAgent) AddTransitingVehicle(v *Vehicle, m *Map, desired_pos VehiclePosition) {
	start_lane_id := v.pos.lane_id
	end_lane_id := desired_pos.lane_id
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] = &InterLaneVehicleTransit{vehicle_id: v.id, start_lane: start_lane_id, end_lane: end_lane_id, distance: CalculateDistance(m.lanes[start_lane_id].end_pos, m.lanes[end_lane_id].start_pos), progress: 0}
}

func (a *SpawnerAgent) RemoveTransitingVehicle(v *Vehicle, m *Map) {
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] = nil
}

func (a *SpawnerAgent) isVehicleInTransitInterLane(v *Vehicle, m *Map) bool {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] != nil
}

func (a *SpawnerAgent) HasReachedEndInterLane(v *Vehicle, m *Map) bool {
	return AlmostEqual(a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress, 1) || (a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress > 1)
}

func (a *SpawnerAgent) GetInterLaneNextLane(v *Vehicle, m *Map) LaneID {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].end_lane
}

func (a *SpawnerAgent) GetInterLaneDistanceProgress(v *Vehicle, m *Map) (float64, float64) {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].distance, a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress
}

func (a *SpawnerAgent) FindInterLaneNextVehicleAhead(v *Vehicle, m *Map) *Vehicle {
	start := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].start_lane
	end := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].end_lane
	progress := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress

	nearest_progress := 1.1
	nearest_map_vehicle_id := -1

	for map_vehicle_id, transit := range a.vehicle_transit_array {
		if transit != nil && transit.start_lane == start && transit.end_lane == end && transit.progress > progress && transit.progress < nearest_progress {
			nearest_map_vehicle_id = map_vehicle_id
			nearest_progress = transit.progress
		}
	}
	if nearest_map_vehicle_id == -1 {
		return nil
	}
	return m.vehicles.vehicle_array[nearest_map_vehicle_id]
}

func (a *SpawnerAgent) UpdateInterLaneVehicleProgress(v *Vehicle, m *Map, new_progress float64) {
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress = new_progress
}

type SinkAgent struct {
	road_node_id           RoadNodeID
	num_vehicles_processed int
	vehicle_log_channel    chan LoggerVehicleEvent
	vehicle_transit_array  []*InterLaneVehicleTransit
	params                 *SinkAgentParams
}

type SinkAgentParams struct {
}

func NewSinkAgent(node_id RoadNodeID, params *SinkAgentParams, vehicle_log_channel chan LoggerVehicleEvent, config_parameters *ConfigParameters) *SinkAgent {
	return &SinkAgent{road_node_id: node_id, num_vehicles_processed: 0, vehicle_log_channel: vehicle_log_channel, vehicle_transit_array: make([]*InterLaneVehicleTransit, config_parameters.MAX_VEHICLES), params: params}
}

func (a *SinkAgent) Descriptor() StaticAgentType {
	return SinkAgentType
}

func (a *SinkAgent) Poke(mapsim *Map, time SimTime, agent_channel chan StaticAgentFetchResult) {
	agent_channel <- StaticAgentFetchResult{ID: a.road_node_id, Time: time, VehiclesProcessed: a.num_vehicles_processed}
	a.num_vehicles_processed = 0
	return
}

func (a *SinkAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) {
	current_lane := vehicle.GetCurrentLane(mapsim)
	current_lane.RemoveVehicleFromQueue(vehicle, mapsim)
	mapsim.vehicles.vehicle_array[mapsim.GetMapArrayVehicleIDIndex(vehicle.id)] = nil
	mapsim.vehicles.next_empty = min(mapsim.GetMapArrayVehicleIDIndex(vehicle.id), mapsim.vehicles.next_empty)
	a.num_vehicles_processed++
	a.vehicle_log_channel <- LoggerVehicleEvent{VehicleID: vehicle.id, NodeID: a.road_node_id, Time: time, LaneID: current_lane.id, EventType: LoggerVehicleEventDestroy}
}

func (a *SinkAgent) CanVehicleProceed(mapsim *Map, time SimTime, vehicle *Vehicle, desired_lane LaneID) bool {
	a.num_vehicles_processed++

	//Check whether InterLane is free to proceed
	next_vehicle_id := GetInterLaneNextVehicle(&a.vehicle_transit_array, vehicle.pos.lane_id, desired_lane, 0, mapsim)
	if next_vehicle_id >= 0 {
		next_vehicle_info := a.vehicle_transit_array[mapsim.GetMapArrayVehicleIDIndex(next_vehicle_id)]
		if next_vehicle_info.progress*next_vehicle_info.distance < vehicle.prop.minimum_gap_size {
			return false
		}
	}

	if isVehicleInterLaneConflicting(&a.vehicle_transit_array, desired_lane, vehicle, mapsim) {
		return false
	}

	a.vehicle_log_channel <- LoggerVehicleEvent{VehicleID: vehicle.id, NodeID: a.road_node_id, Time: time, LaneID: vehicle.pos.lane_id, EventType: LoggerVehicleEventLaneOut}
	a.vehicle_log_channel <- LoggerVehicleEvent{VehicleID: vehicle.id, NodeID: a.road_node_id, Time: time, LaneID: desired_lane, EventType: LoggerVehicleEventLaneIn}

	return true
}

func (a *SinkAgent) AddTransitingVehicle(v *Vehicle, m *Map, desired_pos VehiclePosition) {
	start_lane_id := v.pos.lane_id
	end_lane_id := desired_pos.lane_id
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] = &InterLaneVehicleTransit{vehicle_id: v.id, start_lane: start_lane_id, end_lane: end_lane_id, distance: CalculateDistance(m.lanes[start_lane_id].end_pos, m.lanes[end_lane_id].start_pos), progress: 0}
}

func (a *SinkAgent) RemoveTransitingVehicle(v *Vehicle, m *Map) {
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] = nil
}

func (a *SinkAgent) isVehicleInTransitInterLane(v *Vehicle, m *Map) bool {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] != nil
}

func (a *SinkAgent) HasReachedEndInterLane(v *Vehicle, m *Map) bool {
	return AlmostEqual(a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress, 1) || (a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress > 1)
}

func (a *SinkAgent) GetInterLaneNextLane(v *Vehicle, m *Map) LaneID {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].end_lane
}

func (a *SinkAgent) GetInterLaneDistanceProgress(v *Vehicle, m *Map) (float64, float64) {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].distance, a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress
}

func (a *SinkAgent) FindInterLaneNextVehicleAhead(v *Vehicle, m *Map) *Vehicle {
	start := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].start_lane
	end := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].end_lane
	progress := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress

	nearest_progress := 1.1
	nearest_map_vehicle_id := -1

	for map_vehicle_id, transit := range a.vehicle_transit_array {
		if transit != nil && transit.start_lane == start && transit.end_lane == end && transit.progress > progress && transit.progress < nearest_progress {
			nearest_map_vehicle_id = map_vehicle_id
			nearest_progress = transit.progress
		}
	}
	if nearest_map_vehicle_id == -1 {
		return nil
	}
	return m.vehicles.vehicle_array[nearest_map_vehicle_id]
}

func (a *SinkAgent) UpdateInterLaneVehicleProgress(v *Vehicle, m *Map, new_progress float64) {
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress = new_progress
}

type TrafficLightIntersectionAgent struct {
	road_node_id           RoadNodeID
	num_vehicles_processed int
	vehicle_log_channel    chan LoggerVehicleEvent
	vehicle_transit_array  []*InterLaneVehicleTransit
	params                 *TrafficLightIntersectionAgentParams
}

type TrafficLightIntersectionAgentParams struct {
	time_intervals           []SimTime
	time_interval_index      int
	time_interval_lastChange SimTime
}

func NewTrafficLightIntersectionAgent(node_id RoadNodeID, params *TrafficLightIntersectionAgentParams, vehicle_log_channel chan LoggerVehicleEvent, config_parameters *ConfigParameters) *TrafficLightIntersectionAgent {
	return &TrafficLightIntersectionAgent{road_node_id: node_id, num_vehicles_processed: 0, vehicle_log_channel: vehicle_log_channel, vehicle_transit_array: make([]*InterLaneVehicleTransit, config_parameters.MAX_VEHICLES), params: params}
}

func (a *TrafficLightIntersectionAgent) Descriptor() StaticAgentType {
	return TrafficLightIntersectionAgentType
}

func (a *TrafficLightIntersectionAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) { //Only Sink nodes destroy vehicles
	return
}

func (a *TrafficLightIntersectionAgent) CanVehicleProceed(mapsim *Map, time SimTime, vehicle *Vehicle, desired_lane LaneID) bool {
	road_node := mapsim.nodes[a.road_node_id]
	if road_node.lanes_in[a.params.time_interval_index] == vehicle.pos.lane_id {
		a.num_vehicles_processed++

		//Check whether InterLane is free to proceed
		next_vehicle_id := GetInterLaneNextVehicle(&a.vehicle_transit_array, vehicle.pos.lane_id, desired_lane, 0, mapsim)
		if next_vehicle_id >= 0 {
			next_vehicle_info := a.vehicle_transit_array[mapsim.GetMapArrayVehicleIDIndex(next_vehicle_id)]
			if next_vehicle_info.progress*next_vehicle_info.distance < vehicle.prop.minimum_gap_size {
				return false
			}
		}

		if isVehicleInterLaneConflicting(&a.vehicle_transit_array, desired_lane, vehicle, mapsim) {
			return false
		}

		a.vehicle_log_channel <- LoggerVehicleEvent{VehicleID: vehicle.id, NodeID: a.road_node_id, Time: time, LaneID: vehicle.pos.lane_id, EventType: LoggerVehicleEventLaneOut}
		a.vehicle_log_channel <- LoggerVehicleEvent{VehicleID: vehicle.id, NodeID: a.road_node_id, Time: time, LaneID: desired_lane, EventType: LoggerVehicleEventLaneIn}

		return true
	}

	return false
}

func (a *TrafficLightIntersectionAgent) Poke(mapsim *Map, time SimTime, agent_channel chan StaticAgentFetchResult) {
	if time-a.params.time_interval_lastChange >= a.params.time_intervals[a.params.time_interval_index] {
		a.params.time_interval_index++
		a.params.time_interval_lastChange = time
	}
	if a.params.time_interval_index >= len(a.params.time_intervals) {
		a.params.time_interval_index = 0
	}
	agent_channel <- StaticAgentFetchResult{ID: a.road_node_id, Time: time, VehiclesProcessed: a.num_vehicles_processed}
	a.num_vehicles_processed = 0
}

func (a *TrafficLightIntersectionAgent) AddTransitingVehicle(v *Vehicle, m *Map, desired_pos VehiclePosition) {
	start_lane_id := v.pos.lane_id
	end_lane_id := desired_pos.lane_id
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] = &InterLaneVehicleTransit{vehicle_id: v.id, start_lane: start_lane_id, end_lane: end_lane_id, distance: CalculateDistance(m.lanes[start_lane_id].end_pos, m.lanes[end_lane_id].start_pos), progress: 0}
}

func (a *TrafficLightIntersectionAgent) RemoveTransitingVehicle(v *Vehicle, m *Map) {
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] = nil
}

func (a *TrafficLightIntersectionAgent) isVehicleInTransitInterLane(v *Vehicle, m *Map) bool {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)] != nil
}

func (a *TrafficLightIntersectionAgent) HasReachedEndInterLane(v *Vehicle, m *Map) bool {
	return AlmostEqual(a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress, 1) || (a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress > 1)
}

func (a *TrafficLightIntersectionAgent) GetInterLaneNextLane(v *Vehicle, m *Map) LaneID {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].end_lane
}

func (a *TrafficLightIntersectionAgent) GetInterLaneDistanceProgress(v *Vehicle, m *Map) (float64, float64) {
	return a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].distance, a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress
}

func (a *TrafficLightIntersectionAgent) FindInterLaneNextVehicleAhead(v *Vehicle, m *Map) *Vehicle {
	start := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].start_lane
	end := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].end_lane
	progress := a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress

	nearest_progress := 1.1
	nearest_map_vehicle_id := -1

	for map_vehicle_id, transit := range a.vehicle_transit_array {
		if transit != nil && transit.start_lane == start && transit.end_lane == end && transit.progress > progress && transit.progress < nearest_progress {
			nearest_map_vehicle_id = map_vehicle_id
			nearest_progress = transit.progress
		}
	}
	if nearest_map_vehicle_id == -1 {
		return nil
	}
	return m.vehicles.vehicle_array[nearest_map_vehicle_id]
}

func (a *TrafficLightIntersectionAgent) UpdateInterLaneVehicleProgress(v *Vehicle, m *Map, new_progress float64) {
	a.vehicle_transit_array[m.GetMapArrayVehicleIDIndex(v.id)].progress = new_progress
}
