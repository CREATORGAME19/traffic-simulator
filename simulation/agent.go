package simulation

type StaticAgent interface {
	SpawnVehicles(mapsim *Map, time SimTime)
	DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle)
	Descriptor() StaticAgentType
}

type StaticAgentType int

const (
	IntersectionAgentType StaticAgentType = 0
	SpawnerAgentType      StaticAgentType = 1
	SinkAgentType         StaticAgentType = 2
)

func NewStaticAgent(node_id RoadNodeID, agent_type StaticAgentType, agent_prop map[string]any) StaticAgent {
	switch agent_type{
		case IntersectionAgentType:
			return NewIntersectionAgent(node_id, &IntersectionAgentParams{})
		case SpawnerAgentType:
			param, ok := agent_prop["spawn_rate"]
			if !ok {
				panic("Invalid Static Agent Params!")
			}
			spawn_rate, ok := param.(float64)
			if !ok {
				panic("Invalid Static Agent Params!")
			}
			return NewSpawnerAgent(node_id, &SpawnerAgentParams{spawn_rate: spawn_rate, accumulator: 0})
		case SinkAgentType:
			return NewSinkAgent(node_id, &SinkAgentParams{})
		default:
			return NewIntersectionAgent(node_id, &IntersectionAgentParams{}) //Default to IntersectionAgent if StaticAgentType is unclear.
	}
}

type IntersectionAgent struct {
	road_node_id RoadNodeID
	lastFetch    SimTime
	params       *IntersectionAgentParams
}

type IntersectionAgentParams struct {
}

func NewIntersectionAgent(node_id RoadNodeID, params *IntersectionAgentParams) *IntersectionAgent {
	return &IntersectionAgent{road_node_id: node_id, lastFetch: -1, params: params}
}

func (a *IntersectionAgent) Descriptor() StaticAgentType {
	return IntersectionAgentType
}

func (a *IntersectionAgent) SpawnVehicles(mapsim *Map, time SimTime) { //Only Spawner nodes spawn vehicles
	return
}

func (a *IntersectionAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) { //Only Sink nodes destroy vehicles
	return
}

type SpawnerAgent struct {
	road_node_id RoadNodeID
	lastFetch    SimTime
	params       *SpawnerAgentParams
}

type SpawnerAgentParams struct {
	spawn_rate  float64
	accumulator float64
}

func NewSpawnerAgent(node_id RoadNodeID, params *SpawnerAgentParams) *SpawnerAgent {
	return &SpawnerAgent{road_node_id: node_id, lastFetch: -1, params: params}
}

func (a *SpawnerAgent) Descriptor() StaticAgentType {
	return SpawnerAgentType
}

func (a *SpawnerAgent) SpawnVehicles(mapsim *Map, time SimTime) {
	a.lastFetch = time
	if mapsim.vehicles.next_empty >= mapsim.config_parameters.MAX_VEHICLES {
		//println("Vehicle limit reached!") //WARNING
		return
	}
	a.params.accumulator += a.params.spawn_rate
	for a.params.accumulator >= 1.0 {
		if mapsim.vehicles.next_empty >= mapsim.config_parameters.MAX_VEHICLES {
			//println("Vehicle limit reached!") //WARNING
			return
		}
		mapsim.vehicles.vehicle_array[mapsim.vehicles.next_empty] = CreateVehicle(VehicleID(mapsim.vehicles.next_empty), a.road_node_id, mapsim.FindADestinationNode(), make([]RoadNodeID, 0))
		mapsim.vehicles.next_empty = FindNextEmptyVehicles(mapsim)
		a.params.accumulator -= 1
	}
}

func (a *SpawnerAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) { //Only Sink nodes destroy vehicles
	return
}

type SinkAgent struct {
	road_node_id RoadNodeID
	lastFetch    SimTime
	params *SinkAgentParams
}

type SinkAgentParams struct {

}

func NewSinkAgent(node_id RoadNodeID, params *SinkAgentParams) *SinkAgent {
	return &SinkAgent{road_node_id: node_id, lastFetch: -1, params: params}
}

func (a *SinkAgent) Descriptor() StaticAgentType {
	return SinkAgentType
}

func (a *SinkAgent) SpawnVehicles(mapsim *Map, time SimTime) { //Only Spawner nodes spawn vehicles
	return
}

func (a *SinkAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) {
	a.lastFetch = time
	current_lane := vehicle.GetCurrentLane(mapsim)
	current_lane.RemoveVehicleFromQueue(vehicle)
	mapsim.vehicles.vehicle_array[vehicle.id] = nil
	mapsim.vehicles.next_empty = min(int(vehicle.id), mapsim.vehicles.next_empty)
}
