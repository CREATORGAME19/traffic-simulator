package simulation

import "fmt"

type StaticAgent interface {
	Poke(mapsim *Map, time SimTime)
	Descriptor() StaticAgentType
	DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle)
	CanVehicleProceed(mapsim *Map, time SimTime, vehicle *Vehicle, desired_lane LaneID) bool
}

type StaticAgentType int

const (
	IntersectionAgentType             StaticAgentType = 0
	SpawnerAgentType                  StaticAgentType = 1
	SinkAgentType                     StaticAgentType = 2
	TrafficLightIntersectionAgentType StaticAgentType = 3
)

func NewStaticAgent(node_id RoadNodeID, agent_type StaticAgentType, agent_prop map[string]any) (StaticAgent, error) {
	switch agent_type {
	case IntersectionAgentType:
		return NewIntersectionAgent(node_id, &IntersectionAgentParams{}), nil
	case SpawnerAgentType:
		param, ok := agent_prop["spawn_rate"]
		if !ok {
			return nil, fmt.Errorf("Error: Spawner Agent Param expects 'spawn_rate' as a parameter. 'spawn_rate' not found as valid param!")
		}
		spawn_rate, ok := param.(float64)
		if !ok {
			return nil, fmt.Errorf("Error: Spawner Agent Param expects 'spawn_rate' as a float64 parameter.")
		}
		return NewSpawnerAgent(node_id, &SpawnerAgentParams{spawn_rate: spawn_rate, lastSpawnTime: 0}), nil
	case SinkAgentType:
		return NewSinkAgent(node_id, &SinkAgentParams{}), nil
	case TrafficLightIntersectionAgentType:
		param, ok := agent_prop["time_intervals"]
		if !ok {
			return nil, fmt.Errorf("Error: Traffic Light Intersection Agent Param expects 'time_intervals' as a parameter. 'time_intervals' not found as valid param!")
		}
		fmt.Println(param)
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
		return NewTrafficLightIntersectionAgent(node_id, &TrafficLightIntersectionAgentParams{time_intervals: time_intervals, time_interval_index: 0, time_interval_lastChange: 0}), nil
	default:
		return nil, fmt.Errorf("Error: Unknown Static Agent Type!")
	}
}

type IntersectionAgent struct {
	road_node_id RoadNodeID
	params       *IntersectionAgentParams
}

type IntersectionAgentParams struct {
}

func NewIntersectionAgent(node_id RoadNodeID, params *IntersectionAgentParams) *IntersectionAgent {
	return &IntersectionAgent{road_node_id: node_id, params: params}
}

func (a *IntersectionAgent) Descriptor() StaticAgentType {
	return IntersectionAgentType
}

func (a *IntersectionAgent) Poke(mapsim *Map, time SimTime) {
	return
}

func (a *IntersectionAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) { //Only Sink nodes destroy vehicles
	return
}

func (a *IntersectionAgent) CanVehicleProceed(mapsim *Map, time SimTime, vehicle *Vehicle, desired_lane LaneID) bool {
	return true
}

type SpawnerAgent struct {
	road_node_id RoadNodeID
	params       *SpawnerAgentParams
}

type SpawnerAgentParams struct {
	spawn_rate    float64
	lastSpawnTime SimTime
}

func NewSpawnerAgent(node_id RoadNodeID, params *SpawnerAgentParams) *SpawnerAgent {
	return &SpawnerAgent{road_node_id: node_id, params: params}
}

func (a *SpawnerAgent) Descriptor() StaticAgentType {
	return SpawnerAgentType
}

func (a *SpawnerAgent) Poke(mapsim *Map, time SimTime) {
	a.SpawnVehicles(mapsim, time)
}

func (a *SpawnerAgent) SpawnVehicles(mapsim *Map, time SimTime) {
	if mapsim.vehicles.next_empty >= mapsim.config_parameters.MAX_VEHICLES {
		//println("Vehicle limit reached!") //WARNING
		return
	}

	if (time - a.params.lastSpawnTime) >= SimTime(1/a.params.spawn_rate) {
		if mapsim.vehicles.next_empty >= mapsim.config_parameters.MAX_VEHICLES {
			//println("Vehicle limit reached!") //WARNING
			return
		}
		mapsim.vehicles.vehicle_array[mapsim.vehicles.next_empty] = CreateVehicle(VehicleID(mapsim.vehicles.next_empty), time, a.road_node_id, mapsim.FindADestinationNode())
		mapsim.vehicles.next_empty = FindNextEmptyVehicles(mapsim)
		a.params.lastSpawnTime = time
	}
}

func (a *SpawnerAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) { //Only Sink nodes destroy vehicles
	return
}

func (a *SpawnerAgent) CanVehicleProceed(mapsim *Map, time SimTime, vehicle *Vehicle, desired_lane LaneID) bool {
	return true
}

type SinkAgent struct {
	road_node_id RoadNodeID
	params       *SinkAgentParams
}

type SinkAgentParams struct {
}

func NewSinkAgent(node_id RoadNodeID, params *SinkAgentParams) *SinkAgent {
	return &SinkAgent{road_node_id: node_id, params: params}
}

func (a *SinkAgent) Descriptor() StaticAgentType {
	return SinkAgentType
}

func (a *SinkAgent) Poke(mapsim *Map, time SimTime) {
	return
}

func (a *SinkAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) {
	current_lane := vehicle.GetCurrentLane(mapsim)
	current_lane.RemoveVehicleFromQueue(vehicle)
	mapsim.vehicles.vehicle_array[vehicle.id] = nil
	mapsim.vehicles.next_empty = min(int(vehicle.id), mapsim.vehicles.next_empty)
}

func (a *SinkAgent) CanVehicleProceed(mapsim *Map, time SimTime, vehicle *Vehicle, desired_lane LaneID) bool {
	return true
}

type TrafficLightIntersectionAgent struct {
	road_node_id RoadNodeID
	params       *TrafficLightIntersectionAgentParams
}

type TrafficLightIntersectionAgentParams struct {
	time_intervals []SimTime
	time_interval_index int
	time_interval_lastChange SimTime
}

func NewTrafficLightIntersectionAgent(node_id RoadNodeID, params *TrafficLightIntersectionAgentParams) *TrafficLightIntersectionAgent {
	return &TrafficLightIntersectionAgent{road_node_id: node_id, params: params}
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
		return true
	}

	return false
}

func (a *TrafficLightIntersectionAgent) Poke(mapsim *Map, time SimTime) {
	if time-a.params.time_interval_lastChange >= a.params.time_intervals[a.params.time_interval_index] {
		a.params.time_interval_index++
		a.params.time_interval_lastChange = time
	}
	if a.params.time_interval_index >= len(a.params.time_intervals) {
		a.params.time_interval_index = 0
	}
}
