package simulation

import (
	"math/rand/v2"
)

func CreateVehicle(id VehicleID, origin RoadNodeID, destination RoadNodeID) *Vehicle {
	path := []RoadNodeID{origin}
	v := NewVehicle(
			id,
			VehicleProp{
				max_speed:        15,
				max_acc:          (rand.Float64()*2)+1,
				minimum_gap_size: 6,
			},
			destination,
			origin,
			CreateVehiclePath(&path),
		)
	return v
}
