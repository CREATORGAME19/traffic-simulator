package simulation

import (
	"math"
	"testing"
)

// ============================================================
// Test helpers
// ============================================================

func newTestConfig() *ConfigParameters {
	return &ConfigParameters{
		MAX_VEHICLES:           10,
		NUM_RUNS:               100,
		INCREMENT_VEHICLE_PATH: 5,
		SIM_RATE:               2.0,
		RECORD_INTERVAL:        1.0,
		LOG_FILE:               "test.jsonl",
	}
}

// newTestMap builds a 3-node linear map (IDs match slice indices):
//
//	node0 (spawner) --lane0(0,0→10,0)--> node1 (intersection) --lane1(10,0→20,0)--> node2 (sink)
func newTestMap() (*Map, chan LoggerVehicleEvent) {
	cfg := newTestConfig()
	logCh := make(chan LoggerVehicleEvent, 256)

	lane0 := NewLane(0, Position{0, 0}, Position{10, 0}, 0, 1, 13.0, cfg)
	lane1 := NewLane(1, Position{10, 0}, Position{20, 0}, 1, 2, 13.0, cfg)

	node0 := NewRoadNode(0, Position{0, 0}, []LaneID{0}, []LaneID{}, 1.0,
		NewSpawnerAgent(0, &SpawnerAgentParams{spawn_rate: 0, reduced_spawn_rate: 0, nextSpawnTime: 0}, logCh, cfg))
	node1 := NewRoadNode(1, Position{10, 0}, []LaneID{1}, []LaneID{0}, 1.0,
		NewIntersectionAgent(1, &IntersectionAgentParams{}, logCh, cfg))
	node2 := NewRoadNode(2, Position{20, 0}, []LaneID{}, []LaneID{1}, 1.0,
		NewSinkAgent(2, &SinkAgentParams{}, logCh, cfg))

	return InitialiseMap([]RoadNode{node0, node1, node2}, []Lane{lane0, lane1}, cfg), logCh
}

// newTrafficLightMap builds a 4-node map with a 2-phase traffic light at node2:
//
//	node0 (spawner) --lane0--> node2 (TL, phases: [lane0 green, lane1 green]) --lane2--> node3 (sink)
//	node1 (spawner) --lane1--> node2
func newTrafficLightMap() (*Map, chan LoggerVehicleEvent) {
	cfg := newTestConfig()
	logCh := make(chan LoggerVehicleEvent, 256)

	lane0 := NewLane(0, Position{0, 0}, Position{10, 0}, 0, 2, 13.0, cfg)
	lane1 := NewLane(1, Position{0, 10}, Position{10, 10}, 1, 2, 13.0, cfg)
	lane2 := NewLane(2, Position{10, 0}, Position{20, 0}, 2, 3, 13.0, cfg)

	tl := NewTrafficLightIntersectionAgent(2, &TrafficLightIntersectionAgentParams{
		time_intervals:           []SimTime{10, 10},
		time_interval_index:      0,
		time_interval_lastChange: 0,
	}, logCh, cfg)

	node0 := NewRoadNode(0, Position{0, 0}, []LaneID{0}, []LaneID{}, 1.0,
		NewSpawnerAgent(0, &SpawnerAgentParams{}, logCh, cfg))
	node1 := NewRoadNode(1, Position{0, 10}, []LaneID{1}, []LaneID{}, 1.0,
		NewSpawnerAgent(1, &SpawnerAgentParams{}, logCh, cfg))
	node2 := NewRoadNode(2, Position{10, 0}, []LaneID{2}, []LaneID{0, 1}, 1.0, tl)
	node3 := NewRoadNode(3, Position{20, 0}, []LaneID{}, []LaneID{2}, 1.0,
		NewSinkAgent(3, &SinkAgentParams{}, logCh, cfg))

	return InitialiseMap([]RoadNode{node0, node1, node2, node3}, []Lane{lane0, lane1, lane2}, cfg), logCh
}

// newTestVehicle creates a vehicle with minimum_gap_size=5 and no V2V.
func newTestVehicle(id VehicleID, origin, dest RoadNodeID, laneID LaneID, progress float64) *Vehicle {
	path := []RoadNodeID{}
	return &Vehicle{
		id:          id,
		prop:        NewVehicleProp(13.0, 3.0, -10.0, 5.0, 0),
		pos:         VehiclePosition{lane_id: laneID, progress: progress},
		destination: dest,
		origin:      origin,
		speed:       0,
		acc:         0,
		path:        &VehiclePath{path_array: &path, curr_index: 0},
		lastFetch:   0,
		spawnTime:   0,
	}
}

// placeVehicle registers a vehicle in both the map's vehicle array and a lane queue.
func placeVehicle(m *Map, v *Vehicle, laneID LaneID) {
	idx := m.GetMapArrayVehicleIDIndex(v.id)
	m.vehicles.vehicle_array[idx] = v
	m.lanes[laneID].vehicle_queue.vehicles[idx] = v
	m.lanes[laneID].vehicle_queue.usage++
}

// ============================================================
// CalculateDistance
// ============================================================

func TestCalculateDistance(t *testing.T) {
	cases := []struct {
		name     string
		p1, p2   Position
		wantDist float64
	}{
		{"same point", Position{3, 4}, Position{3, 4}, 0},
		{"horizontal right", Position{0, 0}, Position{5, 0}, 5},
		{"horizontal left", Position{5, 0}, Position{0, 0}, 5},
		{"vertical up", Position{0, 0}, Position{0, 12}, 12},
		{"3-4-5 triangle", Position{0, 0}, Position{3, 4}, 5},
		{"5-12-13 triangle", Position{0, 0}, Position{5, 12}, 13},
		{"origin to origin", Position{0, 0}, Position{0, 0}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateDistance(tc.p1, tc.p2)
			if math.Abs(got-tc.wantDist) > 1e-9 {
				t.Errorf("expected %f, got %f", tc.wantDist, got)
			}
		})
	}
}

// ============================================================
// AlmostEqual
// ============================================================

func TestAlmostEqual(t *testing.T) {
	const tol = 0.00000001 // 1e-8 — the internal tolerance
	cases := []struct {
		name string
		a, b float64
		want bool
	}{
		{"identical", 1.0, 1.0, true},
		{"both zero", 0.0, 0.0, true},
		{"negative equal", -1.0, -1.0, true},
		{"within tolerance", 1.0, 1.0 + 1e-10, true},
		{"at tolerance boundary", 1.0, 1.0 + tol, true},
		{"just outside tolerance", 1.0, 1.0 + 1e-7, false},
		{"clearly different", 1.0, 2.0, false},
		{"near zero", 0.0, 1e-10, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := AlmostEqual(tc.a, tc.b); got != tc.want {
				t.Errorf("AlmostEqual(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// ============================================================
// ParseExtraArg
// ============================================================

func TestParseExtraArg(t *testing.T) {
	t.Run("valid parameters are applied", func(t *testing.T) {
		cases := []struct {
			arg   string
			check func(*ConfigParameters) bool
		}{
			{"MAX_VEHICLES=50", func(c *ConfigParameters) bool { return c.MAX_VEHICLES == 50 }},
			{"NUM_RUNS=1000", func(c *ConfigParameters) bool { return c.NUM_RUNS == 1000 }},
			{"INCREMENT_VEHICLE_PATH=3", func(c *ConfigParameters) bool { return c.INCREMENT_VEHICLE_PATH == 3 }},
			{"SIM_RATE=4.0", func(c *ConfigParameters) bool { return c.SIM_RATE == 4.0 }},
			{"RECORD_INTERVAL=2.0", func(c *ConfigParameters) bool { return c.RECORD_INTERVAL == 2.0 }},
			{"LOG_FILE=out.jsonl", func(c *ConfigParameters) bool { return c.LOG_FILE == "out.jsonl" }},
		}
		for _, tc := range cases {
			t.Run(tc.arg, func(t *testing.T) {
				cfg := newTestConfig()
				if err := ParseExtraArg(tc.arg, cfg); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !tc.check(cfg) {
					t.Errorf("config field not updated correctly after %q", tc.arg)
				}
			})
		}
	})

	t.Run("empty string is a no-op", func(t *testing.T) {
		cfg := newTestConfig()
		if err := ParseExtraArg("", cfg); err != nil {
			t.Errorf("unexpected error for empty arg: %v", err)
		}
	})

	t.Run("invalid parameters return an error", func(t *testing.T) {
		cases := []struct {
			name string
			arg  string
		}{
			{"missing equals sign", "MAX_VEHICLES50"},
			{"unknown key", "UNKNOWN_KEY=5"},
			{"zero SIM_RATE", "SIM_RATE=0"},
			{"negative SIM_RATE", "SIM_RATE=-1"},
			{"zero RECORD_INTERVAL", "RECORD_INTERVAL=0"},
			{"non-numeric value", "MAX_VEHICLES=abc"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				cfg := newTestConfig()
				if err := ParseExtraArg(tc.arg, cfg); err == nil {
					t.Errorf("expected error for %q, got nil", tc.arg)
				}
			})
		}
	})
}

// ============================================================
// NewLane
// ============================================================

func TestNewLane(t *testing.T) {
	cfg := newTestConfig()

	t.Run("distance is computed from endpoints", func(t *testing.T) {
		cases := []struct {
			name     string
			start    Position
			end      Position
			wantDist float64
		}{
			{"horizontal 10", Position{0, 0}, Position{10, 0}, 10},
			{"vertical 5", Position{0, 0}, Position{0, 5}, 5},
			{"3-4-5 diagonal", Position{0, 0}, Position{3, 4}, 5},
			{"zero length", Position{5, 5}, Position{5, 5}, 0},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				lane := NewLane(0, tc.start, tc.end, 0, 1, 13.0, cfg)
				if math.Abs(lane.distance-tc.wantDist) > 1e-9 {
					t.Errorf("expected distance %f, got %f", tc.wantDist, lane.distance)
				}
			})
		}
	})

	t.Run("fields are stored correctly", func(t *testing.T) {
		lane := NewLane(7, Position{1, 2}, Position{4, 6}, 3, 5, 11.0, cfg)
		if lane.id != 7 {
			t.Errorf("id: expected 7, got %d", lane.id)
		}
		if lane.from_node != 3 {
			t.Errorf("from_node: expected 3, got %d", lane.from_node)
		}
		if lane.to_node != 5 {
			t.Errorf("to_node: expected 5, got %d", lane.to_node)
		}
		if lane.speed_limit != 11.0 {
			t.Errorf("speed_limit: expected 11.0, got %f", lane.speed_limit)
		}
		if lane.vehicle_queue == nil {
			t.Error("vehicle_queue must not be nil")
		}
	})
}

// ============================================================
// InitialiseMap
// ============================================================

func TestInitialiseMap(t *testing.T) {
	m, _ := newTestMap()
	cfg := newTestConfig()

	t.Run("node and lane counts", func(t *testing.T) {
		if len(m.nodes) != 3 {
			t.Errorf("expected 3 nodes, got %d", len(m.nodes))
		}
		if len(m.lanes) != 2 {
			t.Errorf("expected 2 lanes, got %d", len(m.lanes))
		}
	})

	t.Run("vehicle array capacity matches MAX_VEHICLES", func(t *testing.T) {
		if len(m.vehicles.vehicle_array) != cfg.MAX_VEHICLES {
			t.Errorf("expected %d, got %d", cfg.MAX_VEHICLES, len(m.vehicles.vehicle_array))
		}
	})

	t.Run("vehicle array is initially empty", func(t *testing.T) {
		for i, v := range m.vehicles.vehicle_array {
			if v != nil {
				t.Errorf("slot %d should be nil on a fresh map", i)
			}
		}
	})

	t.Run("next_empty starts at 0", func(t *testing.T) {
		if m.vehicles.next_empty != 0 {
			t.Errorf("expected next_empty=0, got %d", m.vehicles.next_empty)
		}
	})
}

// ============================================================
// Map helpers
// ============================================================

func TestGetMapArrayVehicleIDIndex(t *testing.T) {
	m, _ := newTestMap() // MAX_VEHICLES = 10
	cases := []struct {
		id   VehicleID
		want int
	}{
		{0, 0},
		{5, 5},
		{9, 9},
		{10, 0}, // wraps
		{15, 5},
		{23, 3},
	}
	for _, tc := range cases {
		if got := m.GetMapArrayVehicleIDIndex(tc.id); got != tc.want {
			t.Errorf("GetMapArrayVehicleIDIndex(%d) = %d, want %d", tc.id, got, tc.want)
		}
	}
}

func TestGetRealVehicleID(t *testing.T) {
	m, _ := newTestMap()

	t.Run("fresh map returns slot id unchanged", func(t *testing.T) {
		if got := m.GetRealVehicleID(3); got != 3 {
			t.Errorf("expected 3, got %d", got)
		}
	})

	t.Run("after index increment applies generation offset", func(t *testing.T) {
		m.vehicles.vehicle_id_index_array[3] = 2
		// 3 + 2 * MAX_VEHICLES(10) = 23
		if got := m.GetRealVehicleID(3); got != 23 {
			t.Errorf("expected 23, got %d", got)
		}
	})
}

func TestFindNextEmptyVehicles(t *testing.T) {
	t.Run("all slots empty returns 0", func(t *testing.T) {
		m, _ := newTestMap()
		if got := FindNextEmptyVehicles(m); got != 0 {
			t.Errorf("expected 0, got %d", got)
		}
	})

	t.Run("slot 0 occupied returns 1", func(t *testing.T) {
		m, _ := newTestMap()
		m.vehicles.vehicle_array[0] = newTestVehicle(0, 0, 2, 0, 0)
		if got := FindNextEmptyVehicles(m); got != 1 {
			t.Errorf("expected 1, got %d", got)
		}
	})

	t.Run("first three slots occupied returns 3", func(t *testing.T) {
		m, _ := newTestMap()
		for i := range 3 {
			m.vehicles.vehicle_array[i] = newTestVehicle(VehicleID(i), 0, 2, 0, 0)
		}
		if got := FindNextEmptyVehicles(m); got != 3 {
			t.Errorf("expected 3, got %d", got)
		}
	})

	t.Run("all slots full returns MAX_VEHICLES", func(t *testing.T) {
		m, _ := newTestMap()
		for i := range m.config_parameters.MAX_VEHICLES {
			m.vehicles.vehicle_array[i] = newTestVehicle(VehicleID(i), 0, 2, 0, 0)
		}
		if got := FindNextEmptyVehicles(m); got != m.config_parameters.MAX_VEHICLES {
			t.Errorf("expected %d, got %d", m.config_parameters.MAX_VEHICLES, got)
		}
	})
}

func TestFindADestinationNode(t *testing.T) {
	m, _ := newTestMap()
	// Only node2 is a SinkAgent, so all calls must return 2.
	for range 20 {
		if dest := m.FindADestinationNode(); dest != 2 {
			t.Errorf("expected destination 2, got %d", dest)
		}
	}
}

// ============================================================
// Vehicle construction
// ============================================================

func TestNewVehicleProp(t *testing.T) {
	p := NewVehicleProp(13.0, 3.0, -10.0, 5.0, 2)
	if p.max_speed != 13.0 {
		t.Errorf("max_speed: expected 13.0, got %f", p.max_speed)
	}
	if p.max_acc != 3.0 {
		t.Errorf("max_acc: expected 3.0, got %f", p.max_acc)
	}
	if p.max_decc != -10.0 {
		t.Errorf("max_decc: expected -10.0, got %f", p.max_decc)
	}
	if p.minimum_gap_size != 5.0 {
		t.Errorf("minimum_gap_size: expected 5.0, got %f", p.minimum_gap_size)
	}
	if p.v2v_lookahead != 2 {
		t.Errorf("v2v_lookahead: expected 2, got %d", p.v2v_lookahead)
	}
}

func TestNewVehicle(t *testing.T) {
	path := []RoadNodeID{0}
	prop := NewVehicleProp(13.0, 3.0, -10.0, 5.0, 0)
	v := NewVehicle(42, prop, 2, 0, CreateVehiclePath(&path), SimTime(100))

	t.Run("id is stored", func(t *testing.T) {
		if v.id != 42 {
			t.Errorf("expected id 42, got %d", v.id)
		}
	})
	t.Run("destination is stored", func(t *testing.T) {
		if v.destination != 2 {
			t.Errorf("expected destination 2, got %d", v.destination)
		}
	})
	t.Run("origin is stored", func(t *testing.T) {
		if v.origin != 0 {
			t.Errorf("expected origin 0, got %d", v.origin)
		}
	})
	t.Run("initial speed is zero", func(t *testing.T) {
		if v.speed != 0 {
			t.Errorf("expected speed 0, got %f", v.speed)
		}
	})
	t.Run("initial acceleration is zero", func(t *testing.T) {
		if v.acc != 0 {
			t.Errorf("expected acc 0, got %f", v.acc)
		}
	})
	t.Run("initial lane_id is -1 (unplaced)", func(t *testing.T) {
		if v.pos.lane_id != -1 {
			t.Errorf("expected lane_id -1, got %d", v.pos.lane_id)
		}
	})
	t.Run("spawn time is stored", func(t *testing.T) {
		if v.spawnTime != 100 {
			t.Errorf("expected spawnTime 100, got %v", v.spawnTime)
		}
	})
	t.Run("lastFetch is -1 (never fetched)", func(t *testing.T) {
		if v.lastFetch != -1 {
			t.Errorf("expected lastFetch -1, got %v", v.lastFetch)
		}
	})
}

func TestNewVehicleNodePos(t *testing.T) {
	pos := NewVehicleNodePos(5) // node argument is intentionally ignored
	if pos.lane_id != -1 {
		t.Errorf("expected lane_id -1, got %d", pos.lane_id)
	}
	if pos.progress != 0 {
		t.Errorf("expected progress 0, got %f", pos.progress)
	}
}

// ============================================================
// Vehicle physics
// ============================================================

func TestCalculateSpeed(t *testing.T) {
	cases := []struct {
		name      string
		initSpeed float64
		acc       float64
		delta     SimTime
		want      float64
	}{
		{"at rest no acceleration", 0, 0, 1, 0},
		{"constant speed", 10, 0, 5, 10},
		{"accelerating", 5, 2, 3, 11},  // 5 + 2*3
		{"decelerating", 10, -3, 2, 4}, // 10 - 3*2
		{"zero time delta", 7, 4, 0, 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := newTestVehicle(0, 0, 2, 0, 0)
			v.speed = tc.initSpeed
			v.acc = tc.acc
			if got := v.CalculateSpeed(tc.delta); math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("expected %f, got %f", tc.want, got)
			}
		})
	}
}

func TestCalculateDistanceTravelled(t *testing.T) {
	cases := []struct {
		name  string
		speed float64
		acc   float64
		delta SimTime
		want  float64
	}{
		{"at rest", 0, 0, 5, 0},
		{"constant speed", 10, 0, 2, 20},          // 10*2
		{"accelerating from rest", 0, 2, 2, 4},    // 0.5*2*4
		{"speed and acceleration", 5, 2, 3, 24},   // 5*3 + 0.5*2*9
		{"zero time delta", 20, 5, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := newTestVehicle(0, 0, 2, 0, 0)
			v.speed = tc.speed
			v.acc = tc.acc
			if got := v.CalculateDistanceTravelled(tc.delta); math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("expected %f, got %f", tc.want, got)
			}
		})
	}
}

func TestHasReachedEndLane(t *testing.T) {
	cases := []struct {
		name     string
		progress float64
		want     bool
	}{
		{"at start", 0.0, false},
		{"mid lane", 0.5, false},
		{"near end but outside tolerance", 1.0 - 1e-7, false},
		{"within float tolerance of 1.0", 1.0 - 1e-10, true},
		{"exactly 1.0", 1.0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := newTestVehicle(0, 0, 2, 0, tc.progress)
			if got := v.HasReachedEndLane(); got != tc.want {
				t.Errorf("progress=%v: expected %v, got %v", tc.progress, tc.want, got)
			}
		})
	}
}

func TestIsLaneFreeAtPos(t *testing.T) {
	t.Run("empty lane is free", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0.0)
		if !v.isLaneFreeAtPos(m, VehiclePosition{lane_id: 0, progress: 0.0}) {
			t.Error("expected empty lane to be free")
		}
	})

	t.Run("vehicle too close blocks entry", func(t *testing.T) {
		m, _ := newTestMap()
		// Blocker at (1,0) on lane0 (length=10); gap from (0,0) = 1 < minimum_gap_size=5
		blocker := newTestVehicle(1, 0, 2, 0, 0.1)
		placeVehicle(m, blocker, 0)

		v := newTestVehicle(0, 0, 2, 0, 0.0)
		if v.isLaneFreeAtPos(m, VehiclePosition{lane_id: 0, progress: 0.0}) {
			t.Error("expected entry to be blocked when vehicle is within minimum gap")
		}
	})

	t.Run("vehicle far enough ahead allows entry", func(t *testing.T) {
		m, _ := newTestMap()
		// Far vehicle at (9,0); gap from (0,0) = 9 >= minimum_gap_size=5
		far := newTestVehicle(1, 0, 2, 0, 0.9)
		placeVehicle(m, far, 0)

		v := newTestVehicle(0, 0, 2, 0, 0.0)
		if !v.isLaneFreeAtPos(m, VehiclePosition{lane_id: 0, progress: 0.0}) {
			t.Error("expected entry to be free when vehicle is beyond minimum gap")
		}
	})

	t.Run("exactly at gap boundary is free", func(t *testing.T) {
		m, _ := newTestMap()
		// Vehicle at (5,0); gap from (0,0) = 5 == minimum_gap_size=5 → free (>=)
		boundary := newTestVehicle(1, 0, 2, 0, 0.5)
		placeVehicle(m, boundary, 0)

		v := newTestVehicle(0, 0, 2, 0, 0.0)
		if !v.isLaneFreeAtPos(m, VehiclePosition{lane_id: 0, progress: 0.0}) {
			t.Error("expected entry to be free exactly at gap boundary")
		}
	})
}

func TestHasReachedDestination(t *testing.T) {
	t.Run("at end of lane leading to destination", func(t *testing.T) {
		m, _ := newTestMap()
		// lane1 ends at node2; vehicle destination = 2
		v := newTestVehicle(0, 0, 2, 1, 1.0)
		if !v.hasReachedDestination(m) {
			t.Error("expected vehicle at end of lane leading to its destination to have arrived")
		}
	})

	t.Run("at end of lane leading to wrong node", func(t *testing.T) {
		m, _ := newTestMap()
		// lane0 ends at node1; vehicle destination = 2
		v := newTestVehicle(0, 0, 2, 0, 1.0)
		if v.hasReachedDestination(m) {
			t.Error("expected vehicle at end of lane leading to wrong node not to have arrived")
		}
	})

	t.Run("mid-lane even if correct destination lane", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 1, 0.5)
		if v.hasReachedDestination(m) {
			t.Error("expected mid-lane vehicle not to have reached destination")
		}
	})
}

// ============================================================
// VehiclePath
// ============================================================

func TestContainsRoadNodeID(t *testing.T) {
	cases := []struct {
		name   string
		slice  []RoadNodeID
		target RoadNodeID
		want   bool
	}{
		{"empty slice", []RoadNodeID{}, 1, false},
		{"single element found", []RoadNodeID{5}, 5, true},
		{"single element not found", []RoadNodeID{5}, 3, false},
		{"first of many", []RoadNodeID{1, 2, 3}, 1, true},
		{"last of many", []RoadNodeID{1, 2, 3}, 3, true},
		{"middle of many", []RoadNodeID{1, 2, 3}, 2, true},
		{"not present in many", []RoadNodeID{1, 2, 3}, 99, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ContainsRoadNodeID(&tc.slice, tc.target); got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestVehiclePath(t *testing.T) {
	t.Run("empty path has no next node", func(t *testing.T) {
		v := newTestVehicle(0, 0, 2, 0, 0)
		if v.NextNodeVehiclePathExists() {
			t.Error("expected no next node on empty path")
		}
	})

	t.Run("AddVehiclePath makes nodes available", func(t *testing.T) {
		v := newTestVehicle(0, 0, 2, 0, 0)
		v.AddVehiclePath([]RoadNodeID{1, 2})
		if !v.NextNodeVehiclePathExists() {
			t.Error("expected path to exist after add")
		}
		if got := v.GetNextNodeVehiclePath(); got != 1 {
			t.Errorf("expected first node 1, got %d", got)
		}
	})

	t.Run("IncrementVehiclePathIndex advances to next node", func(t *testing.T) {
		v := newTestVehicle(0, 0, 2, 0, 0)
		v.AddVehiclePath([]RoadNodeID{1, 2})
		v.IncrementVehiclePathIndex()
		if got := v.GetNextNodeVehiclePath(); got != 2 {
			t.Errorf("expected node 2 after increment, got %d", got)
		}
	})

	t.Run("incrementing past end exhausts path", func(t *testing.T) {
		v := newTestVehicle(0, 0, 2, 0, 0)
		v.AddVehiclePath([]RoadNodeID{1})
		v.IncrementVehiclePathIndex()
		if v.NextNodeVehiclePathExists() {
			t.Error("expected path to be exhausted after incrementing past end")
		}
	})

	t.Run("AddVehiclePath appends to existing entries", func(t *testing.T) {
		v := newTestVehicle(0, 0, 2, 0, 0)
		v.AddVehiclePath([]RoadNodeID{1})
		v.AddVehiclePath([]RoadNodeID{2, 3})
		v.IncrementVehiclePathIndex()
		v.IncrementVehiclePathIndex()
		if got := v.GetNextNodeVehiclePath(); got != 3 {
			t.Errorf("expected node 3 at index 2, got %d", got)
		}
	})

	t.Run("CreateVehiclePath points to the provided slice", func(t *testing.T) {
		nodes := []RoadNodeID{5, 6, 7}
		path := CreateVehiclePath(&nodes)
		if path.curr_index != 0 {
			t.Errorf("expected curr_index 0, got %d", path.curr_index)
		}
		if path.path_array != &nodes {
			t.Error("expected path_array to reference the provided slice")
		}
	})
}

// ============================================================
// ConvertVehiclePosToXY
// ============================================================

func TestConvertVehiclePosToXY(t *testing.T) {
	m, _ := newTestMap()
	cases := []struct {
		name        string
		laneID      LaneID
		progress    float64
		wantX, wantY float64
	}{
		{"lane0 at start", 0, 0.0, 0, 0},
		{"lane0 midpoint", 0, 0.5, 5, 0},
		{"lane0 at end", 0, 1.0, 10, 0},
		{"lane0 quarter mark", 0, 0.25, 2.5, 0},
		{"lane1 at start", 1, 0.0, 10, 0},
		{"lane1 midpoint", 1, 0.5, 15, 0},
		{"lane1 at end", 1, 1.0, 20, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ConvertVehiclePosToXY(m, VehiclePosition{lane_id: tc.laneID, progress: tc.progress})
			if math.Abs(got.x-tc.wantX) > 1e-9 || math.Abs(got.y-tc.wantY) > 1e-9 {
				t.Errorf("expected (%.1f, %.1f), got (%f, %f)", tc.wantX, tc.wantY, got.x, got.y)
			}
		})
	}
}

// ============================================================
// Lane queue
// ============================================================

func TestLaneQueue(t *testing.T) {
	t.Run("add increments usage", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0)
		m.vehicles.vehicle_array[0] = v
		m.lanes[0].AddVehicleToQueue(v, m)
		if m.lanes[0].vehicle_queue.usage != 1 {
			t.Errorf("expected usage 1, got %d", m.lanes[0].vehicle_queue.usage)
		}
	})

	t.Run("remove decrements usage back to zero", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0)
		m.vehicles.vehicle_array[0] = v
		m.lanes[0].AddVehicleToQueue(v, m)
		m.lanes[0].RemoveVehicleFromQueue(v, m)
		if m.lanes[0].vehicle_queue.usage != 0 {
			t.Errorf("expected usage 0, got %d", m.lanes[0].vehicle_queue.usage)
		}
	})

	t.Run("remove clears the vehicle slot", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0)
		m.vehicles.vehicle_array[0] = v
		m.lanes[0].AddVehicleToQueue(v, m)
		m.lanes[0].RemoveVehicleFromQueue(v, m)
		if m.lanes[0].vehicle_queue.vehicles[0] != nil {
			t.Error("expected vehicle slot to be nil after removal")
		}
	})

	t.Run("added vehicle is findable in the queue", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0.5)
		m.vehicles.vehicle_array[0] = v
		m.lanes[0].AddVehicleToQueue(v, m)
		if m.lanes[0].vehicle_queue.vehicles[0] != v {
			t.Error("expected vehicle to be present in queue after add")
		}
	})
}

func TestFindNextVehicleAheadFromPos(t *testing.T) {
	t.Run("returns nil on empty lane", func(t *testing.T) {
		m, _ := newTestMap()
		if got := m.lanes[0].FindNextVehicleAheadFromPos(m, VehiclePosition{lane_id: 0, progress: 0}); got != nil {
			t.Error("expected nil on empty lane")
		}
	})

	t.Run("finds vehicle at or ahead of query progress", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0.8)
		placeVehicle(m, v, 0)
		got := m.lanes[0].FindNextVehicleAheadFromPos(m, VehiclePosition{lane_id: 0, progress: 0.3})
		if got != v {
			t.Error("expected to find vehicle ahead of query position")
		}
	})

	t.Run("does not find vehicle behind query progress", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0.2)
		placeVehicle(m, v, 0)
		got := m.lanes[0].FindNextVehicleAheadFromPos(m, VehiclePosition{lane_id: 0, progress: 0.5})
		if got != nil {
			t.Error("expected nil when no vehicle is at or ahead of query position")
		}
	})
}

func TestFindNextVehicleAhead(t *testing.T) {
	t.Run("returns nil when alone on lane", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0.0)
		placeVehicle(m, v, 0)
		if got := m.lanes[0].FindNextVehicleAhead(m, v); got != nil {
			t.Error("expected nil when no other vehicle is present")
		}
	})

	t.Run("finds the vehicle directly ahead", func(t *testing.T) {
		m, _ := newTestMap()
		v0 := newTestVehicle(0, 0, 2, 0, 0.2)
		v1 := newTestVehicle(1, 0, 2, 0, 0.7)
		placeVehicle(m, v0, 0)
		placeVehicle(m, v1, 0)
		if got := m.lanes[0].FindNextVehicleAhead(m, v0); got != v1 {
			t.Error("expected v1 to be found ahead of v0")
		}
	})

	t.Run("returns nearest of two vehicles ahead", func(t *testing.T) {
		m, _ := newTestMap()
		v0 := newTestVehicle(0, 0, 2, 0, 0.1)
		v1 := newTestVehicle(1, 0, 2, 0, 0.5) // nearer
		v2 := newTestVehicle(2, 0, 2, 0, 0.9) // farther
		placeVehicle(m, v0, 0)
		placeVehicle(m, v1, 0)
		placeVehicle(m, v2, 0)
		if got := m.lanes[0].FindNextVehicleAhead(m, v0); got != v1 {
			t.Error("expected nearest vehicle ahead (v1) to be returned")
		}
	})

	t.Run("does not return vehicle behind", func(t *testing.T) {
		m, _ := newTestMap()
		v0 := newTestVehicle(0, 0, 2, 0, 0.8)
		v1 := newTestVehicle(1, 0, 2, 0, 0.2) // behind v0
		placeVehicle(m, v0, 0)
		placeVehicle(m, v1, 0)
		if got := m.lanes[0].FindNextVehicleAhead(m, v0); got != nil {
			t.Errorf("expected nil, got vehicle at progress %f", got.pos.progress)
		}
	})
}

// ============================================================
// Navigation
// ============================================================

func TestCalculateCostHeuristicFunction(t *testing.T) {
	m, _ := newTestMap()
	// node0=(0,0), node1=(10,0), node2=(20,0)
	cases := []struct {
		name     string
		from, to RoadNodeID
		want     float64
	}{
		{"same node", 1, 1, 0},
		{"node0 to node1", 0, 1, 10},
		{"node0 to node2", 0, 2, 20},
		{"node1 to node2", 1, 2, 10},
		{"node2 to node0 (reverse)", 2, 0, 20},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateCostHeuristicFunction(m, tc.from, tc.to)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("expected %f, got %f", tc.want, got)
			}
		})
	}
}

func TestCalculateVehiclePath(t *testing.T) {
	t.Run("finds path on reachable map", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0) // dest = node2
		if err := v.CalculateVehiclePath(m, 0); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !v.NextNodeVehiclePathExists() {
			t.Fatal("expected a path to exist after CalculateVehiclePath")
		}
		// The first hop from node0 must be node1 (the only route)
		if next := v.GetNextNodeVehiclePath(); next != 1 {
			t.Errorf("expected first hop = node1, got %d", next)
		}
	})

	t.Run("full path reaches destination", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0)
		if err := v.CalculateVehiclePath(m, 0); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		v.IncrementVehiclePathIndex() // skip node1
		if got := v.GetNextNodeVehiclePath(); got != 2 {
			t.Errorf("expected second hop = node2 (destination), got %d", got)
		}
	})

	t.Run("returns error when no path exists", func(t *testing.T) {
		m, _ := newTestMap()
		// node1 → node2 (sink, no lanes_out) → no path back to node0
		v := newTestVehicle(0, 1, 0, 1, 0.5)
		if err := v.CalculateVehiclePath(m, 1); err == nil {
			t.Error("expected error when destination is unreachable")
		}
	})
}

// ============================================================
// CalculateAcceleration
// ============================================================

func TestCalculateAcceleration(t *testing.T) {
	t.Run("free road produces positive acceleration", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0)
		v.speed = 0
		acc := v.CalculateAcceleration(m, 1000, nil, SimTime(1), 13.0)
		if acc <= 0 {
			t.Errorf("expected positive acceleration on free road, got %f", acc)
		}
	})

	t.Run("at max speed produces near-zero acceleration", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0)
		v.speed = 13.0
		acc := v.CalculateAcceleration(m, 1000, nil, SimTime(1), 13.0)
		// ((max_speed - speed) / max_speed) * max_acc = 0
		if math.Abs(acc) > 1e-9 {
			t.Errorf("expected ~0 acceleration at max speed, got %f", acc)
		}
	})

	t.Run("gap less than stopping gap causes deceleration", func(t *testing.T) {
		m, _ := newTestMap()
		leader := newTestVehicle(1, 0, 2, 0, 0.5)
		leader.speed = 0

		follower := newTestVehicle(0, 0, 2, 0, 0.4)
		follower.speed = 10
		// stopping_gap = max(5, 2*10) = 20; gap = 1.0 < 20 → decelerate
		acc := follower.CalculateAcceleration(m, 1.0, leader, SimTime(1), 13.0)
		if acc >= 0 {
			t.Errorf("expected negative acceleration when too close, got %f", acc)
		}
	})

	t.Run("gap in easing zone adjusts speed toward leader", func(t *testing.T) {
		m, _ := newTestMap()
		leader := newTestVehicle(1, 0, 2, 0, 0.9) // at (9,0)
		leader.speed = 5

		follower := newTestVehicle(0, 0, 2, 0, 0.1) // at (1,0)
		follower.speed = 1
		// stopping_gap = max(5, 2*1) = 5; gap = 8 → in easing zone (5, 12.5)
		// new_acc = (5^2 - 1^2) / (2 * (5 + 8 - 5)) = 24/16 = 1.5
		acc := follower.CalculateAcceleration(m, 8.0, leader, SimTime(1), 13.0)
		if math.Abs(acc-1.5) > 1e-9 {
			t.Errorf("expected easing-zone acceleration 1.5, got %f", acc)
		}
	})

	t.Run("acceleration is clamped to max_acc", func(t *testing.T) {
		m, _ := newTestMap()
		v := newTestVehicle(0, 0, 2, 0, 0)
		v.speed = 0
		acc := v.CalculateAcceleration(m, 1000, nil, SimTime(1), 13.0)
		if acc > v.prop.max_acc {
			t.Errorf("acceleration %f exceeds max_acc %f", acc, v.prop.max_acc)
		}
	})

	t.Run("deceleration is clamped to max_decc", func(t *testing.T) {
		m, _ := newTestMap()
		leader := newTestVehicle(1, 0, 2, 0, 0.5)
		leader.speed = 0

		follower := newTestVehicle(0, 0, 2, 0, 0.4)
		follower.speed = 13.0
		acc := follower.CalculateAcceleration(m, 0.5, leader, SimTime(1), 13.0)
		if acc < follower.prop.max_decc {
			t.Errorf("acceleration %f is below max_decc %f", acc, follower.prop.max_decc)
		}
	})
}

// ============================================================
// Inter-lane transit (IntersectionAgent state machine)
// ============================================================

func TestInterLaneTransit(t *testing.T) {
	m, _ := newTestMap()
	agent := m.nodes[1].agent // IntersectionAgent at node1
	v := newTestVehicle(0, 0, 2, 0, 1.0)
	m.vehicles.vehicle_array[0] = v
	desired := VehiclePosition{lane_id: 1, progress: 0}

	// Verify precondition outside a subtest so failures are fatal.
	if agent.isVehicleInTransitInterLane(v, m) {
		t.Fatal("precondition failed: vehicle should not be in transit before any add")
	}

	agent.AddTransitingVehicle(v, m, desired)

	t.Run("in transit after add", func(t *testing.T) {
		if !agent.isVehicleInTransitInterLane(v, m) {
			t.Error("expected vehicle to be in transit after AddTransitingVehicle")
		}
	})

	t.Run("end lane ID is correct", func(t *testing.T) {
		if lane := agent.GetInterLaneNextLane(v, m); lane != 1 {
			t.Errorf("expected end lane 1, got %d", lane)
		}
	})

	t.Run("initial progress is zero", func(t *testing.T) {
		_, prog := agent.GetInterLaneDistanceProgress(v, m)
		if prog != 0 {
			t.Errorf("expected initial progress 0, got %f", prog)
		}
	})

	t.Run("has not reached end at progress 0", func(t *testing.T) {
		if agent.HasReachedEndInterLane(v, m) {
			t.Error("expected HasReachedEndInterLane=false at progress 0")
		}
	})

	agent.UpdateInterLaneVehicleProgress(v, m, 1.0)

	t.Run("progress is updated correctly", func(t *testing.T) {
		_, prog := agent.GetInterLaneDistanceProgress(v, m)
		if prog != 1.0 {
			t.Errorf("expected progress 1.0 after update, got %f", prog)
		}
	})

	t.Run("reached end after progress set to 1", func(t *testing.T) {
		if !agent.HasReachedEndInterLane(v, m) {
			t.Error("expected HasReachedEndInterLane=true after progress=1")
		}
	})

	agent.RemoveTransitingVehicle(v, m)

	t.Run("not in transit after remove", func(t *testing.T) {
		if agent.isVehicleInTransitInterLane(v, m) {
			t.Error("expected vehicle to no longer be in transit after RemoveTransitingVehicle")
		}
	})

	t.Run("end lane is -1 after remove", func(t *testing.T) {
		if lane := agent.GetInterLaneNextLane(v, m); lane != -1 {
			t.Errorf("expected -1 after remove, got %d", lane)
		}
	})
}

// ============================================================
// Traffic light agent
// ============================================================

func TestTrafficLightCanVehicleProceed(t *testing.T) {
	t.Run("vehicle on green lane can proceed", func(t *testing.T) {
		m, _ := newTrafficLightMap()
		// index=0 → lanes_in[0]=lane0 is green
		v := newTestVehicle(0, 0, 3, 0, 1.0) // on lane0
		m.vehicles.vehicle_array[0] = v
		if !m.nodes[2].agent.CanVehicleProceed(m, SimTime(0), v, 2) {
			t.Error("expected green-lane vehicle to be permitted to proceed")
		}
	})

	t.Run("vehicle on red lane is blocked", func(t *testing.T) {
		m, _ := newTrafficLightMap()
		// index=0 → lanes_in[0]=lane0 is green; lane1 is red
		v := newTestVehicle(1, 1, 3, 1, 1.0) // on lane1
		m.vehicles.vehicle_array[1] = v
		if m.nodes[2].agent.CanVehicleProceed(m, SimTime(0), v, 2) {
			t.Error("expected red-lane vehicle to be blocked")
		}
	})
}

func TestTrafficLightPoke_PhaseCycling(t *testing.T) {
	m, _ := newTrafficLightMap()
	tl := m.nodes[2].agent.(*TrafficLightIntersectionAgent)
	agentCh := make(chan StaticAgentFetchResult, 4)

	t.Run("initial phase index is 0", func(t *testing.T) {
		if tl.params.time_interval_index != 0 {
			t.Errorf("expected initial index 0, got %d", tl.params.time_interval_index)
		}
	})

	m.nodes[2].agent.Poke(m, SimTime(5), agentCh)
	<-agentCh
	t.Run("index unchanged before interval expires (t=5, interval=10)", func(t *testing.T) {
		if tl.params.time_interval_index != 0 {
			t.Errorf("expected index 0, got %d", tl.params.time_interval_index)
		}
	})

	m.nodes[2].agent.Poke(m, SimTime(10), agentCh)
	<-agentCh
	t.Run("index advances to 1 when interval expires (t=10)", func(t *testing.T) {
		if tl.params.time_interval_index != 1 {
			t.Errorf("expected index 1, got %d", tl.params.time_interval_index)
		}
	})

	m.nodes[2].agent.Poke(m, SimTime(15), agentCh)
	<-agentCh
	t.Run("index unchanged while second phase still active (t=15)", func(t *testing.T) {
		if tl.params.time_interval_index != 1 {
			t.Errorf("expected index 1, got %d", tl.params.time_interval_index)
		}
	})

	m.nodes[2].agent.Poke(m, SimTime(20), agentCh)
	<-agentCh
	t.Run("index wraps back to 0 after all phases complete (t=20)", func(t *testing.T) {
		if tl.params.time_interval_index != 0 {
			t.Errorf("expected index 0 after wrap, got %d", tl.params.time_interval_index)
		}
	})
}
