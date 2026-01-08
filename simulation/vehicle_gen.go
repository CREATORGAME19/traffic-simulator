package simulation

func CreateVehicle(id int, origin int, destination int) *Vehicle {
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
