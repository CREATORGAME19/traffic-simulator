import json
import os
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
import statistics


SOURCE_NODE = 9   # Barton Road
DEST_NODE   = 45  # Newmarket Road


def collect_per_run_averages(file_paths):
    """Returns one average journey time per file for the Barton Rd → Newmarket Rd route."""
    per_run_averages = []

    for log_file_path in file_paths:
        run_name = os.path.basename(log_file_path)
        journey_times = []
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

                            if src_node != SOURCE_NODE or dest_node != DEST_NODE:
                                continue
                            if spawn_time is not None and current_time is not None:
                                journey_times.append((current_time - spawn_time) / 60)

                    except json.JSONDecodeError:
                        print(f"Skipping invalid JSON line in {run_name}: {line}")
        except FileNotFoundError:
            print(f"Error: Could not find file {log_file_path}")

        if journey_times:
            per_run_averages.append(sum(journey_times) / len(journey_times))
        else:
            print(f"Warning: no matching events in {run_name}")

    return per_run_averages


def plot_journey_times(groups):
    """
    groups: list of (label, [file_paths]) tuples.
    Each group produces one bar, pooling all files in that group.
    """
    run_names = []
    averages = []
    std_devs = []

    for label, file_paths in groups:
        run_avgs = collect_per_run_averages(file_paths)
        if run_avgs:
            avg = sum(run_avgs) / len(run_avgs)
            sd = statistics.stdev(run_avgs) if len(run_avgs) > 1 else 0.0
            run_names.append(label)
            averages.append(avg)
            std_devs.append(sd)
            print(f"[{label}] Runs: {len(run_avgs)} | Mean of run avgs: {avg:.2f} min | Std: {sd:.2f} min")
        else:
            print(f"[{label}] No matching events found.")

    if not averages:
        print("No VEHICLE_DESTROY events found across any files for this route.")
        return

    # --- COLOR MAPPING & GROUPING LOGIC ---
    color_map = {
        'Baseline': '#7F8C8D',
        '100%': '#2ECC71',
        '50%': '#F39C12',
        '25%': '#E74C3C'
    }

    bar_colors = []
    x_positions = []
    current_x = 0
    previous_group = None
    group_x_coords = {}

    for name in run_names:
        if 'lookahead1' in name or 'Lookahead 1' in name:
            current_group = 'Lookahead 1'
        elif 'lookahead2' in name or 'Lookahead 2' in name:
            current_group = 'Lookahead 2'
        elif 'lookahead4' in name or 'Lookahead 4' in name:
            current_group = 'Lookahead 4'
        else:
            current_group = 'Baseline'

        if previous_group is not None and current_group != previous_group:
            current_x += 1.0

        x_positions.append(current_x)

        if current_group not in group_x_coords:
            group_x_coords[current_group] = []
        group_x_coords[current_group].append(current_x)

        current_x += 1.0
        previous_group = current_group

        if '100%' in name:
            bar_colors.append(color_map['100%'])
        elif '50%' in name:
            bar_colors.append(color_map['50%'])
        elif '25%' in name:
            bar_colors.append(color_map['25%'])
        else:
            bar_colors.append(color_map['Baseline'])

    plt.figure(figsize=(14, 7))

    bars = plt.bar(
        x_positions,
        averages,
        yerr=std_devs,
        capsize=5,
        color=bar_colors,
        edgecolor='black',
        alpha=0.85,
        width=0.8,
        zorder=3
    )

    plt.xlabel('Lookahead Range (Number of Vehicles)', fontsize=14)
    plt.ylabel('Average Journey Time (minutes)', fontsize=14)

    plt.grid(axis='y', linestyle='--', alpha=0.7, zorder=0)

    tick_positions = []
    tick_labels = []
    for g_name, coords in group_x_coords.items():
        tick_positions.append(sum(coords) / len(coords))
        tick_labels.append(g_name)

    plt.xticks(tick_positions, tick_labels, fontsize=12)

    legend_handles = [
        mpatches.Patch(color=color_map['Baseline'], label='Baseline (0%)'),
        mpatches.Patch(color=color_map['100%'], label='100% V2V Adoption'),
        mpatches.Patch(color=color_map['50%'], label='50% V2V Adoption'),
        mpatches.Patch(color=color_map['25%'], label='25% V2V Adoption')
    ]
    plt.legend(handles=legend_handles, title="Adoption Rate", fontsize=11, title_fontsize=12, loc='lower right')

    for bar in bars:
        yval = bar.get_height()
        plt.text(
            bar.get_x() + bar.get_width() / 2,
            yval + (max(std_devs) * 0.1 if std_devs else 0.02),
            f'{yval:.1f}',
            ha='center',
            va='bottom',
            fontsize=10,
            fontweight='bold'
        )

    plt.tight_layout()
    plt.show()


if __name__ == "__main__":
    RUNS = ['RUN1', 'RUN2', 'RUN3', 'RUN4', 'RUN5']
    BASE = '../new_logs2'

    def files(suffix):
        return [f'{BASE}/{r}{suffix}.jsonl' for r in RUNS]

    groups = [
        ('Baseline',               files('')),
        ('Lookahead 1 – 100%',     files('lookahead1_100%')),
        ('Lookahead 1 – 50%',      files('lookahead1_50%')),
        ('Lookahead 1 – 25%',      files('lookahead1_25%2')),
        ('Lookahead 2 – 100%',     files('lookahead2_100%')),
        ('Lookahead 2 – 50%',      files('lookahead2_50%')),
        ('Lookahead 2 – 25%',      files('lookahead2_25%')),
        ('Lookahead 4 – 100%',     files('lookahead4_100%')),
        ('Lookahead 4 – 50%',      files('lookahead4_50%')),
        ('Lookahead 4 – 25%',      files('lookahead4_25%')),
    ]
    plot_journey_times(groups)
