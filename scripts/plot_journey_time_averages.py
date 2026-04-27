import json
import math
import matplotlib.pyplot as plt
import statistics # NEW: Imported to calculate standard deviation

# Dictionary mapping node IDs to their real-world names
NODE_NAMES = {
    39: "Madingley Road",
    37: "Huntington Road",
    38: "Histon Road",
    43: "Milton Road",
    45: "Newmarket Road",
    44: "Mill Road",
    42: "Hills Road",
    41: "Trumpington Road",
    55: "Central Cambridge Market",
    54: "Grand Arcade Car Park",
    56: "Grafton Car Park",
    9: "Barton Road"
}

def get_excluded_pairs(map_file_path, distance_threshold=1.0):
    with open(map_file_path, 'r') as f:
        map_data = json.load(f)
        
    spawns = {}
    sinks = {}
    
    for node in map_data.get('road_nodes', []):
        if node['Agent'] == 1:
            spawns[node['ID']] = node['Position']
        elif node['Agent'] == 2:
            sinks[node['ID']] = node['Position']
            
    excluded_pairs = set()
    
    for spawn_id, s_pos in spawns.items():
        for sink_id, d_pos in sinks.items():
            dist = math.hypot(s_pos['x'] - d_pos['x'], s_pos['y'] - d_pos['y'])
            if dist < distance_threshold: 
                excluded_pairs.add((spawn_id, sink_id))
                
    return excluded_pairs

def plot_journey_times(log_file_paths, map_file_path):
    excluded_pairs = get_excluded_pairs(map_file_path)

    interval_size = 100
    binned_route_times = {} 
    
    overall_route_times = {} 
    
    all_journey_times = []
    filtered_count = 0

    for log_file_path in log_file_paths:
        
        spawn_times = {}
        source_nodes = {}
        destination_nodes = {}

        try:
            with open(log_file_path, 'r') as file:
                for line in file:
                    line = line.strip()
                    if not line:
                        continue
                        
                    try:
                        data = json.loads(line)
                        
                        vid = data.get("vehicle_id")
                        current_time = data.get("time")
                        event_type = data.get("event_type", "").upper()
                        
                        if vid is not None:
                            if "spawn_time" in data:
                                spawn_times[vid] = data["spawn_time"]
                            elif event_type in ["VEHICLE SPAWN", "VEHICLE_SPAWN"]:
                                spawn_times[vid] = current_time
                                
                            if "source_node_id" in data:
                                source_nodes[vid] = data["source_node_id"]
                                
                            if "destination_node_id" in data:
                                destination_nodes[vid] = data["destination_node_id"]

                        if event_type in ["VEHICLE DESTROY", "VEHICLE_DESTROY"]:
                            spawn_time = data.get("spawn_time", spawn_times.get(vid))
                            src_node = data.get("source_node_id", source_nodes.get(vid))
                            dest_node = data.get("destination_node_id", destination_nodes.get(vid))
                            
                            if src_node != 9: # Barton Road filter
                                continue

                            if (src_node, dest_node) in excluded_pairs:
                                filtered_count += 1
                                continue
                            
                            if spawn_time is not None and current_time is not None and src_node is not None and dest_node is not None:
                                journey_time = current_time - spawn_time
                                all_journey_times.append(journey_time)
                                
                                route_key = (src_node,dest_node) #(src_node, dest_node)
                                
                                if route_key not in overall_route_times:
                                    overall_route_times[route_key] = []
                                overall_route_times[route_key].append(journey_time)
                                
                                interval_start = int(current_time // interval_size) * interval_size
                                
                                if route_key not in binned_route_times:
                                    binned_route_times[route_key] = {}
                                    
                                if interval_start not in binned_route_times[route_key]:
                                    binned_route_times[route_key][interval_start] = []
                                    
                                binned_route_times[route_key][interval_start].append(journey_time)

                    except json.JSONDecodeError:
                        print(f"Skipping invalid JSON line in {log_file_path}: {line}")
                        continue
        except FileNotFoundError:
            print(f"Warning: File not found - {log_file_path}")
            continue
                
    if not all_journey_times:
        print(f"No valid VEHICLE_DESTROY events found starting from node 9.")
        return

    # NEW: Calculate and print standard deviation alongside the average
    print("\n--- Overall Average Journey Times per Route ---")
    for route_key, times in sorted(overall_route_times.items()):
        avg_time = sum(times) / len(times)
        
        # Standard deviation requires at least 2 data points
        if len(times) > 1:
            stdev_time = statistics.stdev(times)
        else:
            stdev_time = 0.0
            
        src_name = NODE_NAMES.get(route_key[0], f"Node {route_key[0]}")
        dest_name = NODE_NAMES.get(route_key[1], f"Node {route_key[1]}")
        
        # Formatted string including standard deviation (±)
        print(f"{src_name} \u2192 {dest_name}: {avg_time:.2f}s \u00B1 {stdev_time:.2f}s (Total vehicles: {len(times)})")
    print("-----------------------------------------------\n")

    average_journey = sum(all_journey_times) / len(all_journey_times)
    print(f"Filtered out {filtered_count} immediate arrivals across {len(log_file_paths)} runs.")
    print(f"Total valid vehicles arriving: {len(all_journey_times)}")
    print(f"Overall average valid journey time: {average_journey:.2f} seconds")

    plt.figure(figsize=(12, 7))
    
    for route, intervals in binned_route_times.items():
        plot_intervals = sorted(intervals.keys())
        plot_averages = [
            sum(intervals[interval]) / len(intervals[interval])
            for interval in plot_intervals
        ]
        
        src_name = NODE_NAMES.get(route[0], f"Node {route[0]}")
        dest_name = NODE_NAMES.get(route[1], f"Node {route[1]}")
        
        plt.plot(
            plot_intervals, 
            plot_averages,    
            marker='o',         
            linestyle='-',
            linewidth=2,
            markersize=5,
            label=f"{src_name} \u2192 {dest_name}" 
        )
    
    plt.xlabel(f'Simulation Time (seconds) - {interval_size}s Intervals', fontsize=16)
    plt.ylabel('Average Journey Time (seconds)', fontsize=16)
    
    plt.title('Average Journey Time from Barton Road Over Time', fontsize=16)
    plt.grid(True, linestyle='--', alpha=0.7)
    
    plt.legend(title="Destinations", bbox_to_anchor=(1.05, 1), loc='upper left')
    
    plt.tight_layout()
    plt.show()

if __name__ == "__main__":
    log_file_names = [
        #'../logs/6hourRun4.jsonl'
        '../simulation_log.jsonl'
        #'../logs/0%V2I6hour.jsonl'
    ]
    map_file_name = '../example_maps/world7.json'
    
    plot_journey_times(log_file_names, map_file_name)