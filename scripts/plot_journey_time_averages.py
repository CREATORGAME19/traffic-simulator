import json
import math
import matplotlib.pyplot as plt
import matplotlib.ticker as ticker
import statistics 
import numpy as np 

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
    9: "Barton Road",
    1: "Node 1",
    34: "Node 34",
    7: "Node 7"
}

ALLOWED_ROUTES = {
    (9, None),    # all routes from Barton Road (node 9)
    (1, 45),      # node 1 -> Newmarket Road
    (34, 41),     # node 34 -> Trumpington Road
    (7, 41),      # node 7 -> Trumpington Road
    (7, 43),      # node 7 -> Milton Road
}

def is_allowed_route(src, dest):
    if (src, None) in ALLOWED_ROUTES:
        return True
    return (src, dest) in ALLOWED_ROUTES

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

    interval_size = 600
    simulation_start_hour = 6  # 6:00 AM
    
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

                            if not is_allowed_route(src_node, dest_node):
                                continue

                            if (src_node, dest_node) in excluded_pairs:
                                filtered_count += 1
                                continue
                            
                            if spawn_time is not None and current_time is not None and src_node is not None and dest_node is not None:
                                journey_time = current_time - spawn_time
                                all_journey_times.append(journey_time)
                                
                                route_key = (src_node, dest_node) 
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
        print(f"No valid VEHICLE_DESTROY events found for the specified routes.")
        return

    print("\n--- Overall Average Journey Times per Route (Console Only) ---")
    for route_key, times in sorted(overall_route_times.items()):
        avg_time = sum(times) / len(times)
        stdev_time = statistics.stdev(times) if len(times) > 1 else 0.0
            
        src_name = NODE_NAMES.get(route_key[0], f"Node {route_key[0]}")
        dest_name = NODE_NAMES.get(route_key[1], f"Node {route_key[1]}")
        
        print(f"{src_name} → {dest_name}: {avg_time:.2f}s ± {stdev_time:.2f}s (Total vehicles: {len(times)})")
    print("-----------------------------------------------\n")

    average_journey = sum(all_journey_times) / len(all_journey_times)
    print(f"Filtered out {filtered_count} immediate arrivals across {len(log_file_paths)} runs.")
    print(f"Total valid vehicles arriving: {len(all_journey_times)}")
    print(f"Overall average valid journey time: {average_journey:.2f} seconds")

    # --- Plotting ---
    plt.figure(figsize=(12, 7))
    
    all_intervals = set()
    for intervals in binned_route_times.values():
        all_intervals.update(intervals.keys())
        
    if not all_intervals:
        return
        
    global_min_interval = min(all_intervals)
    global_max_interval = max(all_intervals)
    
    common_intervals = np.arange(global_min_interval, global_max_interval + interval_size, interval_size)
    
    all_route_interpolated_averages = []
    coverage_masks = []

    for route_key, intervals in binned_route_times.items():
        actual_intervals = sorted(intervals.keys())
        if not actual_intervals:
            continue
            
        actual_averages = [
            sum(intervals[interval]) / len(intervals[interval])
            for interval in actual_intervals
        ]
        
        if len(actual_intervals) == 1:
            route_interpolated = np.full_like(common_intervals, np.nan, dtype=float)
            idx = np.searchsorted(common_intervals, actual_intervals[0])
            if idx < len(route_interpolated):
                route_interpolated[idx] = actual_averages[0]
            coverage_mask = ~np.isnan(route_interpolated)
        else:
            route_interpolated = np.interp(common_intervals, actual_intervals, actual_averages)
            coverage_mask = (common_intervals >= actual_intervals[0]) & \
                            (common_intervals <= actual_intervals[-1])

        all_route_interpolated_averages.append(route_interpolated)
        coverage_masks.append(coverage_mask)

    if all_route_interpolated_averages:
        stacked = np.array(all_route_interpolated_averages)
        coverage = np.array(coverage_masks)

        masked = np.where(coverage, stacked, np.nan)
        final_aggregate_averages = np.nanmean(masked, axis=0) / 60
        final_aggregate_stdev = np.nanstd(masked, axis=0) / 60

        def seconds_to_clock(s):
            total_minutes = int(simulation_start_hour * 60 + s / 60)
            h = total_minutes // 60
            m = total_minutes % 60
            return f"{h:02d}:{m:02d}"

        x_labels = [seconds_to_clock(s) for s in common_intervals]

        plt.errorbar(
            range(len(common_intervals)),
            final_aggregate_averages,
            yerr=final_aggregate_stdev,
            marker='o',
            linestyle='-',
            linewidth=2,
            markersize=6,
            capsize=4,
        )

        plt.xticks(range(len(common_intervals)), x_labels, rotation=45, ha='right')

    plt.xlabel('Simulated Time of Day', fontsize=16)
    plt.ylabel('Aggregate Journey Time (minutes)', fontsize=16)
    plt.grid(True, linestyle='--', alpha=0.7)
    plt.legend(loc='upper left')
    plt.tight_layout()
    plt.show()

if __name__ == "__main__":
    log_file_names = [
        '../usb_logs/RUN1trafficlight.jsonl',
        '../usb_logs/RUN2trafficlight.jsonl',
        '../usb_logs/RUN3trafficlight.jsonl',
        '../usb_logs/RUN4trafficlight.jsonl',
        '../usb_logs/RUN5trafficlight.jsonl'
    ]
    map_file_name = '../example_maps/world7.json'
    
    plot_journey_times(log_file_names, map_file_name)