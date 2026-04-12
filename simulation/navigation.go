package simulation

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand/v2"
)

type item[T any] struct {
	value    T       // The value of the item; arbitrary.
	priority float64 // The priority of the item in the queue.
}

// A priorityQueue implements heap.Interface and holds items.
type priorityQueue[T any] []*item[T]

func (pq priorityQueue[T]) Len() int { return len(pq) }

func (pq priorityQueue[T]) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq priorityQueue[T]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue[T]) Push(x any) {
	*pq = append(*pq, x.(*item[T]))
}

func (pq *priorityQueue[T]) Pop() any {
	old := *pq
	n := len(old)
	it := old[n-1]
	*pq = old[0 : n-1]
	return it
}

func (a *Vehicle) CalculateVehiclePath(mapsim *Map, start_node_id RoadNodeID) error {
	end_node_id := a.destination

	came_from := make([]*[]RoadNodeID, len(mapsim.nodes)) //Could probably have an array of sets here instead for better performance
	for i := 0; i < len(came_from); i++ {
		l := make([]RoadNodeID, 0)
		came_from[i] = &l
	}

	pq := &priorityQueue[RoadNodeID]{}
	heap.Init(pq)

	gScore := make([]float64, len(mapsim.nodes))
	for i := 0; i < len(gScore); i++ {
		gScore[i] = math.Inf(1)
	}
	gScore[start_node_id] = 0

	heap.Push(pq, &item[RoadNodeID]{value: start_node_id, priority: CalculateCostHeuristicFunction(mapsim, start_node_id, end_node_id)})
	for pq.Len() > 0 {
		current := heap.Pop(pq).(*item[RoadNodeID]).value

		if current == end_node_id {
			a.reconstructPath(mapsim, came_from, start_node_id, end_node_id)
			return nil
		}

		lanes_out := mapsim.nodes[current].lanes_out
		for l := range lanes_out {
			lane_id := lanes_out[l]
			next_node_id := mapsim.lanes[lane_id].to_node
			total_vehicle_capacity := mapsim.lanes[lane_id].distance / 3 //Estimate of vehicles
			new_G := gScore[current] + (mapsim.lanes[lane_id].distance * (1 + (float64(mapsim.lanes[lane_id].vehicle_queue.usage)) / total_vehicle_capacity))
			if new_G < gScore[next_node_id] {
				came_from[next_node_id] = &[]RoadNodeID{current}
				gScore[next_node_id] = new_G

				heap.Push(pq, &item[RoadNodeID]{value: next_node_id, priority: new_G + CalculateCostHeuristicFunction(mapsim, next_node_id, end_node_id)})
			} else if new_G == gScore[next_node_id] {
				*came_from[next_node_id] = append(*came_from[next_node_id], current)
				heap.Push(pq, &item[RoadNodeID]{value: next_node_id, priority: new_G + CalculateCostHeuristicFunction(mapsim, next_node_id, end_node_id)})
			}
		}
	}
	return fmt.Errorf("Error: No valid path found in vehicle sim!")
}

func CalculateCostHeuristicFunction(mapsim *Map, node_id RoadNodeID, end_node_id RoadNodeID) float64 {
	return CalculateDistance(mapsim.nodes[node_id].pos, mapsim.nodes[end_node_id].pos) //Calculates distance between curr_node and end_node
}

func (a *Vehicle) reconstructPath(mapsim *Map, came_from []*[]RoadNodeID, start_node_id RoadNodeID, end_node_id RoadNodeID) {
	path := []RoadNodeID{end_node_id}
	current := came_from[end_node_id]
	for !(ContainsRoadNodeID(current, start_node_id)) {
		chosen_node := (*current)[rand.IntN(len(*current))]
		path = append([]RoadNodeID{chosen_node}, path...)
		current = came_from[chosen_node]
	}
	if mapsim.config_parameters.INCREMENT_VEHICLE_PATH <= 0 {
		a.AddVehiclePath(path)
	} else {
		a.AddVehiclePath(path[0:min(mapsim.config_parameters.INCREMENT_VEHICLE_PATH, len(path))])
	}
}

func ContainsRoadNodeID(current *[]RoadNodeID, node_id RoadNodeID) bool {
	for n := range *current {
		if (*current)[n] == node_id {
			return true
		}
	}
	return false
}
