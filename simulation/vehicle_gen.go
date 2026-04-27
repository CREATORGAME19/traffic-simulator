package simulation

import (
	"math/rand/v2"
)

func CreateVehicle(id VehicleID, curr_time SimTime, origin RoadNodeID, destination RoadNodeID) *Vehicle {
	path := []RoadNodeID{origin}
	var v *Vehicle = nil
	if rand.Float64() <= 0 {
		v = NewVehicle(
			id,
			VehicleProp{
				max_speed:        13,
				max_acc:          (rand.Float64() * 2) + 1,
				max_decc:         -10,
				minimum_gap_size: 5,
				v2v_lookahead: 1,
			},
			destination,
			origin,
			CreateVehiclePath(&path),
			curr_time,
		)
	} else {
		v = NewVehicle(
			id,
			VehicleProp{
				max_speed:        13,
				max_acc:          (rand.Float64() * 2) + 1,
				max_decc:         -10,
				minimum_gap_size: 6,
				v2v_lookahead: 0,
			},
			destination,
			origin,
			CreateVehiclePath(&path),
			curr_time,
		)
	}
	return v
}
