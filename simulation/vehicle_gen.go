package simulation

func CreateVehicle(id VehicleID, origin RoadNodeID, destination RoadNodeID) *Vehicle {
	path := []RoadNodeID{origin}
	v := NewVehicle(
			id,
			VehicleProp{
				max_speed:        31.29,
				max_acc:          1,
				minimum_gap_size: 3,
			},
			destination,
			origin,
			CreateVehiclePath(&path),
		)
	return v
}
