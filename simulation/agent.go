package simulation

type StaticAgent interface {
	SpawnVehicles(mapsim *Map, time SimTime)
	DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle)
	Descriptor() StaticAgentType
}

type StaticAgentType int

const (
	IntersectionAgentType StaticAgentType = 0
	SpawnerAgentType StaticAgentType = 1
	SinkAgentType StaticAgentType = 2
)

type IntersectionAgent struct {
	road_node_id RoadNodeID
	lastFetch SimTime
}

func NewIntersectionAgent(node_id RoadNodeID) *IntersectionAgent {
	return &IntersectionAgent{road_node_id: node_id, lastFetch: -1}
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
	lastFetch SimTime
	spawn_rate float64
	accumulator float64
}

func NewSpawnerAgent(node_id RoadNodeID, spawn_rate float64) *SpawnerAgent{
	return &SpawnerAgent{road_node_id: node_id, lastFetch: -1, spawn_rate: spawn_rate, accumulator: 0}
}

func (a *SpawnerAgent) Descriptor() StaticAgentType {
	return SpawnerAgentType
}

func (a *SpawnerAgent) SpawnVehicles(mapsim *Map, time SimTime) {
	a.lastFetch = time
	if mapsim.vehicles.next_empty >= MAX_VEHICLES {
		//println("Vehicle limit reached!") //WARNING
		return
	}
	a.accumulator += a.spawn_rate
	for a.accumulator >= 1.0 {
		if mapsim.vehicles.next_empty >= MAX_VEHICLES {
			//println("Vehicle limit reached!") //WARNING
			return
		}
		mapsim.vehicles.vehicle_array[mapsim.vehicles.next_empty] = CreateVehicle(VehicleID(mapsim.vehicles.next_empty),a.road_node_id,mapsim.FindADestinationNode(),make([]RoadNodeID, 0))
		mapsim.vehicles.next_empty = FindNextEmptyVehicles(mapsim)
		a.accumulator -= 1
	}
}

func (a *SpawnerAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) { //Only Sink nodes destroy vehicles
	return
}

type SinkAgent struct {
	road_node_id RoadNodeID
	lastFetch SimTime
}

func NewSinkAgent(node_id RoadNodeID) *SinkAgent{
	return &SinkAgent{road_node_id: node_id, lastFetch: -1}
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
	mapsim.vehicles.next_empty = min(int(vehicle.id),mapsim.vehicles.next_empty)
}