package simulation

func CreateVehicles(m *Map) []*Vehicle {
	agents := []*Vehicle{
		NewVehicle(
			0,
			VehicleProp{
				max_speed: 0.5,
				max_acc:   0.1,
			},
			1,
			0,
			m,
		),
	}
	return agents
}
