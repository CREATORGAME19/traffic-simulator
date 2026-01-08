package simulation

type StaticAgent interface {
	SpawnVehicles(mapsim *Map, time SimTime)
	DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle)
	Descriptor() StaticAgentType
}

type StaticAgentType string

const (
	IntersectionAgentType StaticAgentType = "intersection"
	SpawnerAgentType StaticAgentType = "spawner"
	SinkAgentType StaticAgentType = "sink"
)

type IntersectionAgent struct {
	road_node_id int64
	lastFetch SimTime
}

func NewIntersectionAgent(node_id int64) *IntersectionAgent {
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
	road_node_id int64
	lastFetch SimTime
	spawn_rate float64
	accumulator float64
}

func NewSpawnerAgent(node_id int64, spawn_rate float64) *SpawnerAgent{
	return &SpawnerAgent{road_node_id: node_id, lastFetch: -1, spawn_rate: spawn_rate, accumulator: 0}
}

func (a *SpawnerAgent) Descriptor() StaticAgentType {
	return SpawnerAgentType
}

func (a *SpawnerAgent) SpawnVehicles(mapsim *Map, time SimTime) {
	a.lastFetch = time
	if mapsim.vehicles.next_empty >= MAX_VEHICLES {
		println("Vehicle limit reached!")
		return
	}
	a.accumulator += a.spawn_rate
	if a.accumulator >= 1.0 {
		mapsim.vehicles.vehicle_array[mapsim.vehicles.next_empty] = CreateVehicle(int(mapsim.vehicles.next_empty),int(a.road_node_id),mapsim.FindADestinationNode())
		mapsim.vehicles.next_empty = int64(FindNextEmptyVehicles(mapsim))
		a.accumulator -= 1
	}
}

func (a *SpawnerAgent) DestroyVehicle(mapsim *Map, time SimTime, vehicle *Vehicle) { //Only Sink nodes destroy vehicles
	return
}

type SinkAgent struct {
	road_node_id int64
	lastFetch SimTime
}

func NewSinkAgent(node_id int64) *SinkAgent{
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
	mapsim.vehicles.next_empty = min(int64(vehicle.id),mapsim.vehicles.next_empty)
}