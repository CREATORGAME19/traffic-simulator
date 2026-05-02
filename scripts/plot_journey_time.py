import json
import os
import matplotlib.pyplot as plt
import numpy as np

def plot_journey_times(log_file_paths):
    plot_data = {}
    all_journey_times = []

    simulation_start_hour = 6   # 6:00 AM
    simulation_end_hour   = 12  # 12:00 PM

    def seconds_to_clock(s):
        total_minutes = int(simulation_start_hour * 60 + s / 60)
        h = total_minutes // 60
        m = total_minutes % 60
        return f"{h:02d}:{m:02d}"

    plot_data[0] = {'destroy_times': [], 'journey_times': []}
    for log_file_path in log_file_paths:
        run_name = os.path.basename(log_file_path)
        #plot_data[run_name] = {'destroy_times': [], 'journey_times': []}
        
        spawn_times = {}
        source_nodes = {}
        destination_nodes = {}
        
        run_journey_times = []

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
                            
                            if (dest_node != 45 or src_node != 9):
                                continue
                                
                            if spawn_time is not None and current_time is not None and src_node is not None:
                                journey_time = (current_time - spawn_time) / 60  # convert to minutes
                                
                                run_journey_times.append(journey_time)
                                all_journey_times.append(journey_time)
                                
                                plot_data[0]['destroy_times'].append(current_time)
                                plot_data[0]['journey_times'].append(journey_time)

                    except json.JSONDecodeError:
                        print(f"Skipping invalid JSON line in {run_name}: {line}")
                        continue
            
            if run_journey_times:
                avg = sum(run_journey_times) / len(run_journey_times)
                print(f"[{run_name}] Vehicles arrived: {len(run_journey_times)} | Avg journey time: {avg:.2f} min")
            else:
                print(f"[{run_name}] No matching events found.")

        except FileNotFoundError:
            print(f"Error: Could not find file {log_file_path}")
                
    if not all_journey_times:
        print(f"\nNo VEHICLE_DESTROY events found across any files for this route.")
        return

    overall_average = sum(all_journey_times) / len(all_journey_times)
    print(f"\n--- OVERALL STATS ---")
    print(f"Total vehicles arriving: {len(all_journey_times)}")
    print(f"Overall average journey time: {overall_average:.2f} minutes")

    plt.figure(figsize=(10, 6))
    
    for run_name, axes in plot_data.items():
        if axes['destroy_times']:
            plt.scatter(
                axes['destroy_times'], 
                axes['journey_times'], 
                marker='o', 
                alpha=0.6,
                edgecolors='black',
                label="Before Traffic Light Change" if run_name == "6hourRun3.jsonl" else "After Traffic Light Change",
                s=100
            )

    # Fixed x-axis: 6:00 to 12:00 in 10-minute (600s) steps
    x_start = 0
    x_end   = (simulation_end_hour - simulation_start_hour) * 3600  # 21600s
    step    = 600  # 10 minutes in seconds

    tick_positions = np.arange(x_start, x_end + step, step)
    tick_labels    = [seconds_to_clock(s) for s in tick_positions]

    plt.xlim(x_start, x_end)
    plt.ylim(bottom=0)
    plt.xticks(tick_positions, tick_labels, rotation=45, ha='right')

    plt.xlabel('Simulated Time of Day', fontsize=16)
    plt.ylabel('Journey Time (minutes)', fontsize=16)
    plt.grid(True, linestyle='--', alpha=0.7)
    plt.tight_layout()
    plt.show()

if __name__ == "__main__":
    log_file_names = [
        #'../logs/6hourRun3.jsonl',
        #'../logs/6hourRun5.jsonl',
        #'../logs/1lookahead100%.jsonl',
        #'../logs/1lookahead100%3.jsonl',
        #'../logs/1lookahead50%3.jsonl',
        #'../logs/1lookahead25%2.jsonl',
        #'../logs/2lookahead100%3.jsonl',
        #'../logs/2lookahead50%3.jsonl',
        #'../logs/2lookahead25%2.jsonl',
        #'../logs/4lookahead100%4.jsonl',
        #'../logs/4lookahead50%3.jsonl',
        #'../logs/2lookahead50%.jsonl',
        #'../logs/1lookahead50%.jsonl',
        #'../logs/4lookahead25%2.jsonl',
        #'../logs/2lookahead25%.jsonl',
        #'../logs/1lookahead25%.jsonl',
        '../new_logs2/RUN2.jsonl',
        #'../new_logs2/RUN2.jsonl',
        #'../new_logs2/RUN3.jsonl',
        #'../new_logs2/RUN4.jsonl',
        #'../new_logs2/RUN5.jsonl'
        #'../new_logs2/RUN1trafficlight.jsonl',
        #'../new_logs2/RUN2trafficlight.jsonl',
        #'../new_logs2/RUN3trafficlight.jsonl',
        #'../new_logs2/RUN4trafficlight.jsonl',
        #'../new_logs2/RUN5trafficlight.jsonl'
        #'../new_logs2/RUN1trafficlight.jsonl',
        #'../new_logs2/RUN1V2I10.jsonl'
        #'../logs/RUN2.jsonl',
        #'../logs/RUN3.jsonl',
        #'../logs/RUN4.jsonl',
        #'../logs/RUN5.jsonl',
        #'../simulation_log.jsonl',
        #'../logs/0%V2I6hour.jsonl'
    ] 
    plot_journey_times(log_file_names)