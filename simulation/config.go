package simulation

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const DEFAULT_MAX_VEHICLES = 10
const DEFAULT_MAX_RUNS = 200
const DEFAULT_INCREMENT_VEHICLE_PATH = 5

type MapConfig struct {
	Road_nodes []RoadNodeConfig `json:"road_nodes"`
	Lanes      []LaneConfig     `json:"lanes"`
}

type ConfigParameters struct {
	MAX_VEHICLES           int
	NUM_RUNS               int
	INCREMENT_VEHICLE_PATH int
}

type RoadNodeConfig struct {
	ID        RoadNodeID      `json:"ID"`
	Position  PositionConfig  `json:"Position"`
	Lanes_Out []LaneID        `json:"Lanes_Out"`
	Lanes_In  []LaneID        `json:"Lanes_In"`
	Agent     StaticAgentType `json:"Agent"`
	AgentProp map[string]any  `json:"AgentProp"`
}

type LaneConfig struct {
	ID             LaneID         `json:"ID"`
	Start_Position PositionConfig `json:"Start_Position"`
	End_Position   PositionConfig `json:"End_Position"`
	From_Node      RoadNodeID     `json:"From_Node"`
	To_Node        RoadNodeID     `json:"To_Node"`
}

type PositionConfig struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func ParseSimArgs(args []string) (*MapConfig, *ConfigParameters, error) {
	var map_config *MapConfig
	var config_parameters *ConfigParameters

	map_config = &MapConfig{}
	if len(args) < 2 || args[1] == "" {
		return nil, nil, fmt.Errorf("Error: No path to config file is provided!")
	}
	path := args[1]
	config_parameters = &ConfigParameters{MAX_VEHICLES: DEFAULT_MAX_VEHICLES, NUM_RUNS: DEFAULT_MAX_RUNS, INCREMENT_VEHICLE_PATH: DEFAULT_INCREMENT_VEHICLE_PATH}
	for i := 2; i < len(args); i++ {
		if err := ParseExtraArg(args[i], config_parameters); err != nil {
			return nil, nil, err
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(map_config); err != nil {
		return nil, nil, err
	}

	return map_config, config_parameters, nil
}

func ParseExtraArg(arg string, config_parameters *ConfigParameters) error {
	if arg == "" {
		return nil
	}
	res := strings.SplitN(arg, "=", 2)
	if len(res) != 2 {
		return fmt.Errorf("Invalid Config Parameter!")
	}
	key := res[0]
	value, err := strconv.Atoi(res[1])
	if err != nil {
		return fmt.Errorf("Invalid Config Parameter!")
	}
	switch key {
	case "MAX_VEHICLES":
		config_parameters.MAX_VEHICLES = value
	case "NUM_RUNS":
		config_parameters.NUM_RUNS = value
	case "INCREMENT_VEHICLE_PATH":
		config_parameters.INCREMENT_VEHICLE_PATH = value
	default:
		return fmt.Errorf("Unrecognised Config Parameter " + key + "!")
	}
	return nil
}
