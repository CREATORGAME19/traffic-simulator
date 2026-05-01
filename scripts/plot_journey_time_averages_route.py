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
    (9, 45),   # Barton Road -> Newmarket Road
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

def collect_binned_times_per_run(log_file_path, excluded_pairs):
    """Collect binned route times for a single log file."""
    interval_size = 600
    binned_route_times = {}
    all_journey_times = []
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
                            continue
                        if spawn_time is not None and current_time is not None and src_node is not None and dest_node is not None:
                            journey_time = current_time - spawn_time
                            all_journey_times.append(journey_time)

                            route_key = (src_node, dest_node)
                            interval_start = int(current_time // interval_size) * interval_size
                            if route_key not in binned_route_times:
                                binned_route_times[route_key] = {}
                            if interval_start not in binned_route_times[route_key]:
                                binned_route_times[route_key][interval_start] = []
                            binned_route_times[route_key][interval_start].append(journey_time)

                except json.JSONDecodeError:
                    continue
    except FileNotFoundError:
        print(f"Warning: File not found - {log_file_path}")

    return binned_route_times, all_journey_times


def compute_aggregate_across_runs(all_runs_binned, common_intervals):
    """
    For each timestep, compute the mean and stdev of per-run averages.
    This gives realistic error bars reflecting run-to-run variability.
    """
    run_series = []
    for binned_route_times in all_runs_binned:
        interval_totals = {}
        interval_counts = {}
        for route_key, intervals in binned_route_times.items():
            for t, times in intervals.items():
                interval_totals[t] = interval_totals.get(t, 0) + sum(times)
                interval_counts[t] = interval_counts.get(t, 0) + len(times)

        if not interval_totals:
            continue

        actual_intervals = sorted(interval_totals.keys())
        actual_averages  = [interval_totals[t] / interval_counts[t] for t in actual_intervals]

        if len(actual_intervals) == 1:
            run_interp = np.full_like(common_intervals, np.nan, dtype=float)
            idx = np.searchsorted(common_intervals, actual_intervals[0])
            if idx < len(run_interp):
                run_interp[idx] = actual_averages[0]
        else:
            run_interp = np.interp(common_intervals, actual_intervals, actual_averages,
                                   left=np.nan, right=np.nan)
            run_interp = np.where(
                (common_intervals >= actual_intervals[0]) & (common_intervals <= actual_intervals[-1]),
                run_interp, np.nan
            )

        run_series.append(run_interp)

    if not run_series:
        return None, None

    stacked = np.array(run_series)
    final_averages = np.nanmean(stacked, axis=0) / 60
    final_stdev    = np.nanstd(stacked,  axis=0) / 60
    return final_averages, final_stdev


def plot_journey_times(baseline_paths, trafficlight_paths, map_file_path):
    excluded_pairs = get_excluded_pairs(map_file_path)
    interval_size = 600
    simulation_start_hour = 6

    def seconds_to_clock(s):
        total_minutes = int(simulation_start_hour * 60 + s / 60)
        h = total_minutes // 60
        m = total_minutes % 60
        return f"{h:02d}:{m:02d}"

    # Collect per-run data
    baseline_runs = []
    baseline_jt   = []
    for p in baseline_paths:
        b, jt = collect_binned_times_per_run(p, excluded_pairs)
        baseline_runs.append(b)
        baseline_jt.extend(jt)

    tl_runs = []
    tl_jt   = []
    for p in trafficlight_paths:
        b, jt = collect_binned_times_per_run(p, excluded_pairs)
        tl_runs.append(b)
        tl_jt.extend(jt)

    if not baseline_jt and not tl_jt:
        print("No valid VEHICLE_DESTROY events found for either group.")
        return

    # Build common timeline
    all_intervals = set()
    for run in baseline_runs + tl_runs:
        for intervals in run.values():
            all_intervals.update(intervals.keys())

    if not all_intervals:
        return

    common_intervals = np.arange(min(all_intervals), max(all_intervals) + interval_size, interval_size)
    x_labels = [seconds_to_clock(s) for s in common_intervals]

    avg_baseline, std_baseline = compute_aggregate_across_runs(baseline_runs, common_intervals)
    avg_tl,       std_tl       = compute_aggregate_across_runs(tl_runs,       common_intervals)

    if baseline_jt:
        print(f"[Baseline]      Total vehicles: {len(baseline_jt)} | Avg: {np.mean(baseline_jt)/60:.2f} min")
    if tl_jt:
        print(f"[Traffic Light] Total vehicles: {len(tl_jt)} | Avg: {np.mean(tl_jt)/60:.2f} min")

    plt.figure(figsize=(12, 7))

    if avg_baseline is not None:
        valid = ~np.isnan(avg_baseline)
        plt.errorbar(
            np.where(valid)[0], avg_baseline[valid], yerr=std_baseline[valid],
            marker='o', linestyle='-', linewidth=2, markersize=4, capsize=4,
            color='blue', label='Before Traffic Light Change'
        )

    if avg_tl is not None:
        valid = ~np.isnan(avg_tl)
        plt.errorbar(
            np.where(valid)[0], avg_tl[valid], yerr=std_tl[valid],
            marker='o', linestyle='-', linewidth=2, markersize=4, capsize=4,
            color='orange', label='After Traffic Light Change'
        )

    plt.xticks(range(len(common_intervals)), x_labels, rotation=45, ha='right')
    plt.xlabel('Simulated Time of Day', fontsize=16)
    plt.ylabel('Average Journey Time (minutes)', fontsize=16)
    plt.grid(True, linestyle='--', alpha=0.7)
    plt.legend(fontsize=12)
    plt.tight_layout()
    plt.show()


if __name__ == "__main__":
    baseline_files = [
        '../usb_logs/RUN1.jsonl',
        '../usb_logs/RUN2.jsonl',
        '../usb_logs/RUN3.jsonl',
        '../usb_logs/RUN4.jsonl',
        '../usb_logs/RUN5.jsonl',
    ]
    trafficlight_files = [
        '../usb_logs/RUN1trafficlight.jsonl',
        '../usb_logs/RUN2trafficlight.jsonl',
        '../usb_logs/RUN3trafficlight.jsonl',
        '../usb_logs/RUN4trafficlight.jsonl',
        '../usb_logs/RUN5trafficlight.jsonl',
    ]
    map_file_name = '../example_maps/world7.json'

    plot_journey_times(baseline_files, trafficlight_files, map_file_name)