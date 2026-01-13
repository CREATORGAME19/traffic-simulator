package simulation

func CreateVehicle(id VehicleID, origin RoadNodeID, destination RoadNodeID) *Vehicle {
	v := NewVehicle(
			id,
			VehicleProp{
				max_speed:        0.5,
				max_acc:          1,
				minimum_gap_size: 3,
			},
			destination,
			origin,
		)
	return v
}
