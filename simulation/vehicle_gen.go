package simulation

func CreateVehicles(m *Map) []*Vehicle {
	vehicles := []*Vehicle{
		NewVehicle(
			0,
			VehicleProp{
				max_speed:        0.5,
				max_acc:          1,
				minimum_gap_size: 3,
			},
			2,
			0,
			m,
		),
	}
	return vehicles
}
