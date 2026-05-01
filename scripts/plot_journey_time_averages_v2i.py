import json
import math
import matplotlib.pyplot as plt
import matplotlib.ticker as ticker
import statistics
import numpy as np

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
    run_series = []
    for binned_route_times in all_runs_binned:
        interval_totals = {}
        interval_counts = {}
        for _, intervals in binned_route_times.items():
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


def plot_journey_times(groups, map_file_path):
    """
    groups: list of dicts with keys:
        - 'label': str
        - 'paths': list of file paths
        - 'color': (optional) matplotlib color string
    """
    excluded_pairs = get_excluded_pairs(map_file_path)
    interval_size = 600
    simulation_start_hour = 6

    default_colors = plt.rcParams['axes.prop_cycle'].by_key()['color']

    def seconds_to_clock(s):
        total_minutes = int(simulation_start_hour * 60 + s / 60)
        h = total_minutes // 60
        m = total_minutes % 60
        return f"{h:02d}:{m:02d}"

    # Collect per-run data for each group
    group_runs = []
    group_jt = []
    for group in groups:
        runs, jt = [], []
        for p in group['paths']:
            b, t = collect_binned_times_per_run(p, excluded_pairs)
            runs.append(b)
            jt.extend(t)
        group_runs.append(runs)
        group_jt.append(jt)

    if all(len(jt) == 0 for jt in group_jt):
        print("No valid VEHICLE_DESTROY events found for any group.")
        return

    # Build common timeline across all groups
    all_intervals = set()
    for runs in group_runs:
        for run in runs:
            for intervals in run.values():
                all_intervals.update(intervals.keys())

    if not all_intervals:
        return

    common_intervals = np.arange(min(all_intervals), max(all_intervals) + interval_size, interval_size)
    x_labels = [seconds_to_clock(s) for s in common_intervals]

    plt.figure(figsize=(12, 7))

    for i, (group, runs, jt) in enumerate(zip(groups, group_runs, group_jt)):
        label = group['label']
        color = group.get('color', default_colors[i % len(default_colors)])

        if jt:
            print(f"[{label}] Total vehicles: {len(jt)} | Avg: {np.mean(jt)/60:.2f} min")

        avg, std = compute_aggregate_across_runs(runs, common_intervals)
        if avg is not None:
            valid = ~np.isnan(avg)
            plt.errorbar(
                np.where(valid)[0], avg[valid], yerr=std[valid],
                marker='o', linestyle='-', linewidth=2, markersize=4, capsize=4,
                color=color, label=label
            )

    plt.xticks(range(len(common_intervals)), x_labels, rotation=45, ha='right')
    plt.xlabel('Simulated Time of Day', fontsize=16)
    plt.ylabel('Average Journey Time (minutes)', fontsize=16)
    plt.grid(True, linestyle='--', alpha=0.7)
    plt.legend(fontsize=12)
    plt.tight_layout()
    plt.show()


if __name__ == "__main__":
    map_file_name = '../example_maps/world7.json'

    groups = [
        {
            'label': 'Baseline (No V2I)',
            'color': 'blue',
            'paths': [
                '../usb_logs/RUN1.jsonl',
                '../usb_logs/RUN2.jsonl',
                '../usb_logs/RUN3.jsonl',
                '../usb_logs/RUN4.jsonl',
                '../usb_logs/RUN5.jsonl',
            ],
        },
        {
            'label': 'alpha=1',
            'color': 'orange',
            'paths': [
                '../usb_logs/RUN1V2I1.jsonl',
            ],
        },
        {
            'label': 'alpha=10',
            'color': 'green',
            'paths': [
                '../usb_logs/RUN1V2I10.jsonl',
                '../usb_logs/RUN2V2I10.jsonl',
                '../usb_logs/RUN3V2I10.jsonl'
            ],
        },
        #{
        #    'label': 'V2I Penetration 10',
        #    'color': 'red',
        #    'paths': [
        #        '../usb_logs/RUN1V2I10.jsonl',
        #    ],
        #},
    ]

    plot_journey_times(groups, map_file_name)
