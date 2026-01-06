package simulation

func CreateVehicles(m *Map) []*Vehicle {
	vehicles := []*Vehicle{
		NewVehicle(
			0,
			VehicleProp{
				max_speed: 0.5,
				max_acc:   0.1,
			},
			2,
			0,
			m,
		),
	}
	return vehicles
}
