package simulation

type AgentPosition struct {
	x int
	y int
}

type Agent struct {
	prop        AgentProp
	pos         Position
	destination Position
	origin      Position
}

type AgentProp struct {
	//Agent Properties here
}

func NewAgent(prop AgentProp, destination Position, origin Position) Agent {
	return Agent{
		prop:        prop,
		pos:         origin,
		destination: destination,
		origin:      origin,
	}
}

func StartAgentSimulation() {

}
