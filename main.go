package main

import (
	"fmt"
	"os"
	. "traffic-simulator/simulation"
)

func main() {
	args := os.Args
	map_config, config_parameters, err := ParseSimArgs(args)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(map_config)
	RunController(map_config, config_parameters)
}