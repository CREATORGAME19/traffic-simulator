import json
import os
import matplotlib.pyplot as plt

def plot_journey_times(log_file_paths):
    plot_data = {}
    all_journey_times = []

    # Loop through each provided log file
    for log_file_path in log_file_paths:
        # Extract just the filename (e.g., '6hourRun3.jsonl') for a cleaner legend
        run_name = os.path.basename(log_file_path)
        plot_data[run_name] = {'destroy_times': [], 'journey_times': []}
        
        # Reset tracking dictionaries per run to avoid ID collisions
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
                            
                            # Your existing route filter
                            if (dest_node != 45 or src_node != 9):
                                continue
                                
                            if spawn_time is not None and current_time is not None and src_node is not None:
                                journey_time = current_time - spawn_time
                                
                                run_journey_times.append(journey_time)
                                all_journey_times.append(journey_time)
                                
                                # Save data under the specific run name
                                plot_data[run_name]['destroy_times'].append(current_time)
                                plot_data[run_name]['journey_times'].append(journey_time)

                    except json.JSONDecodeError:
                        print(f"Skipping invalid JSON line in {run_name}: {line}")
                        continue
            
            # Print stats for this specific run
            if run_journey_times:
                avg = sum(run_journey_times) / len(run_journey_times)
                print(f"[{run_name}] Vehicles arrived: {len(run_journey_times)} | Avg journey time: {avg:.2f} s")
            else:
                print(f"[{run_name}] No matching events found.")

        except FileNotFoundError:
            print(f"Error: Could not find file {log_file_path}")
                
    if not all_journey_times:
        print(f"\nNo VEHICLE_DESTROY events found across any files for this route.")
        return

    # Print overall stats
    overall_average = sum(all_journey_times) / len(all_journey_times)
    print(f"\n--- OVERALL STATS ---")
    print(f"Total vehicles arriving: {len(all_journey_times)}")
    print(f"Overall average journey time: {overall_average:.2f} seconds")

    plt.figure(figsize=(10, 6))
    
    # Plot the scatter points for each run
    for run_name, axes in plot_data.items():
        if axes['destroy_times']: # Only plot if we actually found data for this run
            plt.scatter(
                axes['destroy_times'], 
                axes['journey_times'], 
                marker='o', 
                alpha=0.6,          # Slight transparency helps see overlapping points
                edgecolors='black',
                label="Before Traffic Light Change" if run_name == "6hourRun3.jsonl" else "After Traffic Light Change",     # Add the filename to the legend
                s=100               # Slightly smaller dots to reduce clutter
            )
    
    plt.xlabel('Simulation Time (seconds)', fontsize=16)
    plt.ylabel('Journey Time (seconds)', fontsize=16)
    #plt.title('Journey Times Comparison: Node 9 \u2192 Node 45', fontsize=14)
    plt.grid(True, linestyle='--', alpha=0.7)
    
    # Activate the legend
    plt.legend(title="Simulation Run")
    
    plt.tight_layout()
    plt.show()

if __name__ == "__main__":
    # Provide the list of files to compare here
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
        '../simulation_log.jsonl',
        '../logs/0%V2I6hour.jsonl'
    ] 
    plot_journey_times(log_file_names)