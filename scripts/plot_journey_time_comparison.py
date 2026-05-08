import json
import os
import matplotlib.pyplot as plt
import numpy as np

NODE_NAMES = {
    39: "Madingley Rd",
    37: "Huntington Rd",
    38: "Histon Rd",
    43: "Milton Rd",
    45: "Newmarket Rd",
    44: "Mill Rd",
    42: "Hills Rd",
    41: "Trumpington Rd",
    55: "Central Market",
    54: "Grand Arcade",
    56: "Grafton",
    9:  "Barton Rd",
    1:  "Huntington Rd",
    2:  "Histon Rd",
    7:  "Madingley Rd",
    34: "Milton Rd",
    23: "Trumpington Rd",
    25: "Hills Rd",
    35: "Mill Rd",
    36: "Newmarket Rd",
    50: "Grand Arcade",
    51: "Central Market",
    52: "Grafton",
}

ALLOWED_ROUTES = {
    (9,  41), (9,  55),   # Barton Rd → Trumpington Rd / Central Market
    (25, 41), (25, 55),   # Hills Rd → Trumpington Rd / Central Market
    (35, 41), (35, 55),   # Mill Rd → Trumpington Rd / Central Market
    (36, 41), (36, 55),   # Newmarket Rd → Trumpington Rd / Central Market
}

def collect_route_times(log_file_paths):
    route_times = {}

    for log_file_path in log_file_paths:
        run_name = os.path.basename(log_file_path)
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

                            if spawn_time is None or current_time is None or src_node is None or dest_node is None:
                                continue

                            journey_time = (current_time - spawn_time) / 60
                            key = (src_node, dest_node)
                            if key not in route_times:
                                route_times[key] = []
                            route_times[key].append(journey_time)

                    except json.JSONDecodeError:
                        continue
        except FileNotFoundError:
            print(f"Warning: File not found - {log_file_path}")

    return route_times


def plot_comparison(baseline_files, traffic_light_files):
    print("Collecting baseline runs...")
    baseline = collect_route_times(baseline_files)

    print("Collecting traffic light runs...")
    traffic_light = collect_route_times(traffic_light_files)

    # Find routes present in both sets, filtered to allowed routes
    common_routes = sorted(set(baseline.keys()) & set(traffic_light.keys()) & ALLOWED_ROUTES)

    if not common_routes:
        print("No common routes found between the two file sets.")
        return

    labels = []
    pct_changes = []
    baseline_avgs = []
    tl_avgs = []

    for route in common_routes:
        base_avg = sum(baseline[route]) / len(baseline[route])
        tl_avg   = sum(traffic_light[route]) / len(traffic_light[route])
        pct = ((tl_avg - base_avg) / base_avg) * 100

        src_name  = NODE_NAMES.get(route[0], f"Node {route[0]}")
        dest_name = NODE_NAMES.get(route[1], f"Node {route[1]}")
        labels.append(f"{src_name}\n→ {dest_name}")
        pct_changes.append(pct)
        baseline_avgs.append(base_avg)
        tl_avgs.append(tl_avg)

        print(f"{src_name} → {dest_name}: {base_avg:.2f} min (base) vs {tl_avg:.2f} min (TL) | {pct:+.1f}%")

    # Sort bars by ascending percentage change
    sorted_data = sorted(zip(pct_changes, labels, baseline_avgs, tl_avgs))
    pct_changes   = [d[0] for d in sorted_data]
    labels        = [d[1] for d in sorted_data]
    baseline_avgs = [d[2] for d in sorted_data]
    tl_avgs       = [d[3] for d in sorted_data]

    colors = ['green' if p < 0 else 'red' for p in pct_changes]

    x = np.arange(len(labels))
    fig, ax = plt.subplots(figsize=(max(16, len(labels) * 1.4), 9))
    bars = ax.bar(x, pct_changes, color=colors, edgecolor='black', linewidth=0.8)

    ax.axhline(0, color='black', linewidth=1.0)
    ax.set_ylabel('Average Journey Time Change (%)', fontsize=20)
    ax.set_xticks(x)
    ax.set_xticklabels(labels, fontsize=23, ha='right', rotation=35)
    ax.tick_params(axis='y', labelsize=26)
    ax.grid(True, axis='y', linestyle='--', alpha=0.6)

    for bar, pct in zip(bars, pct_changes):
        ypos = bar.get_height() + 0.3 if pct >= 0 else bar.get_height() - 2.0
        ax.text(bar.get_x() + bar.get_width() / 2, ypos, f'{pct:+.1f}%',
                ha='center', va='bottom', fontsize=19, fontweight='bold')

    ax.legend(handles=[
        plt.Rectangle((0, 0), 1, 1, color='green', label='Improvement (faster)'),
        plt.Rectangle((0, 0), 1, 1, color='red',   label='Degradation (slower)'),
    ], fontsize=21)

    plt.tight_layout()
    plt.show()


if __name__ == "__main__":
    baseline_files = [
        '../new_logs2/RUN1.jsonl',
        '../new_logs2/RUN2.jsonl',
        '../new_logs2/RUN3.jsonl',
        '../new_logs2/RUN4.jsonl',
        '../new_logs2/RUN5.jsonl',
    ]
    traffic_light_files = [
        '../new_logs2/RUN1trafficlight.jsonl',
        '../new_logs2/RUN2trafficlight.jsonl',
        '../new_logs2/RUN3trafficlight.jsonl',
        '../new_logs2/RUN4trafficlight.jsonl',
        '../new_logs2/RUN5trafficlight.jsonl',
    ]
    plot_comparison(baseline_files, traffic_light_files)
