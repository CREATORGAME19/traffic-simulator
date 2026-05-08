import json
import math
import matplotlib.pyplot as plt
import matplotlib.cm as cm
import matplotlib.colors as mcolors
import matplotlib.lines as mlines

def plot_map_heatmap(log_file_paths, map_file_path):
    with open(map_file_path, 'r') as f:
        map_data = json.load(f)

    lanes = {}
    for lane in map_data.get('lanes', []):
        lanes[lane['ID']] = {
            'start_x': lane['Start_Position']['x'],
            'start_y': lane['Start_Position']['y'],
            'end_x': lane['End_Position']['x'],
            'end_y': lane['End_Position']['y']
        }

    lane_times = {lane_id: [] for lane_id in lanes.keys()}

    for log_file in log_file_paths:
        active_vehicles = {}

        try:
            with open(log_file, 'r') as file:
                for line in file:
                    line = line.strip()
                    if not line: continue

                    try:
                        data = json.loads(line)
                        vid = data.get("vehicle_id")
                        current_time = data.get("time")
                        event_type = data.get("event_type", "").upper()
                        lane_id = data.get("lane_id")

                        if vid is None or current_time is None:
                            continue

                        if event_type in ["VEHICLE ENTER LANE", "VEHICLE SPAWN", "VEHICLE_SPAWN"] and lane_id is not None:
                            active_vehicles[vid] = {"lane_id": lane_id, "time": current_time}

                        elif event_type in ["VEHICLE EXIT LANE", "VEHICLE_DESTROY", "VEHICLE DESTROY"]:
                            if vid in active_vehicles:
                                entry_data = active_vehicles.pop(vid)
                                past_lane_id = entry_data["lane_id"]
                                entry_time = entry_data["time"]

                                traversal_time = current_time - entry_time
                                if past_lane_id in lane_times:
                                    lane_times[past_lane_id].append(traversal_time)

                            if event_type == "VEHICLE EXIT LANE" and lane_id is not None:
                                active_vehicles[vid] = {"lane_id": lane_id, "time": current_time}

                    except json.JSONDecodeError:
                        continue
        except FileNotFoundError:
            print(f"Warning: File not found - {log_file}")

    avg_lane_times = {}
    for lid, times in lane_times.items():
        if times:
            avg_lane_times[lid] = (sum(times) / len(times)) / 60  # convert to minutes
        else:
            avg_lane_times[lid] = 0

    target_lanes = [38, 41, 59, 62]
    print("\n--- Average Transit Times for Specific Lanes ---")
    for lane_id in target_lanes:
        avg_time = avg_lane_times.get(lane_id, 0)
        vehicle_count = len(lane_times.get(lane_id, []))
        print(f"Lane {lane_id}: {avg_time:.2f} minutes (Total vehicles: {vehicle_count})")
    print("------------------------------------------------\n")

    valid_times = [t for t in avg_lane_times.values() if t > 0]
    if not valid_times:
        print("No valid lane traversal times found. Please check your event_type strings in the script.")
        return

    min_time = min(valid_times)
    max_time = 5.0  # cap gradient at 5 minutes

    cmap = cm.get_cmap('RdYlGn_r')
    norm = mcolors.Normalize(vmin=min_time, vmax=max_time)

    fig, ax = plt.subplots(figsize=(14, 12))

    offset_distance = 10.0

    for lid, lane_data in lanes.items():
        avg_time = avg_lane_times.get(lid, 0)

        if avg_time == 0:
            color = 'lightgrey'
            zorder = 1
        else:
            color = cmap(norm(avg_time))
            zorder = 2

        dx = lane_data['end_x'] - lane_data['start_x']
        dy = lane_data['end_y'] - lane_data['start_y']
        length = math.hypot(dx, dy)
        
        if length > 0:
            nx = -dy / length
            ny = dx / length
            
            plot_start_x = lane_data['start_x'] + (nx * offset_distance)
            plot_start_y = lane_data['start_y'] + (ny * offset_distance)
            plot_end_x = lane_data['end_x'] + (nx * offset_distance)
            plot_end_y = lane_data['end_y'] + (ny * offset_distance)
        else:
            plot_start_x = lane_data['start_x']
            plot_start_y = lane_data['start_y']
            plot_end_x = lane_data['end_x']
            plot_end_y = lane_data['end_y']

        ax.plot(
            [plot_start_x, plot_end_x],
            [plot_start_y, plot_end_y],
            color=color,
            linewidth=4,
            zorder=zorder,
            solid_capstyle='round'
        )

    for node in map_data.get('road_nodes', []):
        agent_type = node.get('Agent', 0)
        
        if agent_type in [1, 2]:
            node_color = 'blue'
        elif agent_type == 3:
            node_color = 'cyan'
        else:
            node_color = 'black'

        ax.scatter(
            node['Position']['x'],
            node['Position']['y'],
            color=node_color,
            s=75,
            zorder=4
        )

    sm = plt.cm.ScalarMappable(cmap=cmap, norm=norm)
    sm.set_array([])
    cbar = plt.colorbar(sm, ax=ax, fraction=0.03, pad=0.04)
    cbar.set_label('Average Road Traversal Time (minutes)', fontsize=19)
    cbar.ax.tick_params(labelsize=17)

    legend_elements = [
        mlines.Line2D([], [], color='blue', marker='o', linestyle='None', markersize=10, label='Spawn/Sink Agent'),
        mlines.Line2D([], [], color='cyan', marker='o', linestyle='None', markersize=10, label='Traffic Light Agent'),
        mlines.Line2D([], [], color='black', marker='o', linestyle='None', markersize=10, label='Intersection Agent')
    ]
    ax.legend(handles=legend_elements, loc='upper right', title="Agent Types",
              prop={'size': 19}, title_fontsize=19)

    ax.set_aspect('equal', 'box')
    ax.axis('off')  # Remove axes

    plt.tight_layout()
    plt.show()

if __name__ == "__main__":
    log_file_names = [
        '../new_logs2/RUN1.jsonl',
        '../new_logs2/RUN2.jsonl',
        '../new_logs2/RUN3.jsonl',
        '../new_logs2/RUN4.jsonl',
        '../new_logs2/RUN5.jsonl'
    ]
    map_file_name = '../example_maps/world7.json'
    
    plot_map_heatmap(log_file_names, map_file_name)