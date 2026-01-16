package simulation

import "fmt"

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

func NewStaticAgent(node_id RoadNodeID, agent_type StaticAgentType, agent_prop map[string]any) (StaticAgent,error) {
	switch agent_type{
		case IntersectionAgentType:
			return NewIntersectionAgent(node_id, &IntersectionAgentParams{}),nil
		case SpawnerAgentType:
			param, ok := agent_prop["spawn_rate"]
			if !ok {
				return nil, fmt.Errorf("Error: Spawner Agent Param expects 'spawn_rate' as a parameter. 'spawn_rate' not found as valid param!")
			}
			spawn_rate, ok := param.(float64)
			if !ok {
				return nil, fmt.Errorf("Error: Spawner Agent Param expects 'spawn_rate' as a float64 parameter.")
			}
			return NewSpawnerAgent(node_id, &SpawnerAgentParams{spawn_rate: spawn_rate, lastSpawnTime: 0}),nil
		case SinkAgentType:
			return NewSinkAgent(node_id, &SinkAgentParams{}),nil
		default:
			return nil, fmt.Errorf("Error: Unknown Static Agent Type!")
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
	lastSpawnTime SimTime
}

func NewSpawnerAgent(node_id RoadNodeID, params *SpawnerAgentParams) *SpawnerAgent {
	return &SpawnerAgent{road_node_id: node_id, lastFetch: -1, params: params}
}

func (a *SpawnerAgent) Descriptor() StaticAgentType {
	return SpawnerAgentType
}

func (a *SpawnerAgent) SpawnVehicles(mapsim *Map, time SimTime) {
	if mapsim.vehicles.next_empty >= mapsim.config_parameters.MAX_VEHICLES {
		//println("Vehicle limit reached!") //WARNING
		return
	}

	if (time-a.params.lastSpawnTime) >= SimTime(1/a.params.spawn_rate)  {
		if mapsim.vehicles.next_empty >= mapsim.config_parameters.MAX_VEHICLES {
			//println("Vehicle limit reached!") //WARNING
			return
		}
		mapsim.vehicles.vehicle_array[mapsim.vehicles.next_empty] = CreateVehicle(VehicleID(mapsim.vehicles.next_empty), a.road_node_id, mapsim.FindADestinationNode())
		mapsim.vehicles.next_empty = FindNextEmptyVehicles(mapsim)
		a.params.lastSpawnTime = time
	}
	a.lastFetch = time
	
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
	current_lane := vehicle.GetCurrentLane(mapsim)
	current_lane.RemoveVehicleFromQueue(vehicle)
	mapsim.vehicles.vehicle_array[vehicle.id] = nil
	mapsim.vehicles.next_empty = min(int(vehicle.id), mapsim.vehicles.next_empty)
	a.lastFetch = time
}
