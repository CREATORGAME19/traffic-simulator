import json
import matplotlib.pyplot as plt

def plot_journey_times(log_file_path, target_destination=7):
    spawn_times = {}
    source_nodes = {}
    destination_nodes = {}
    
    plot_data = {}
    
    all_journey_times = []

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
                    
                    if dest_node != target_destination:
                        continue
                    
                    if spawn_time is not None and current_time is not None and src_node is not None:
                        journey_time = current_time - spawn_time
                        
                        all_journey_times.append(journey_time)
                        
                        if src_node not in plot_data:
                            plot_data[src_node] = {'destroy_times': [], 'journey_times': []}
                            
                        plot_data[src_node]['destroy_times'].append(current_time)
                        plot_data[src_node]['journey_times'].append(journey_time)

            except json.JSONDecodeError:
                print(f"Skipping invalid JSON line: {line}")
                continue
                
    if not all_journey_times:
        print(f"No VEHICLE_DESTROY events found ending at destination node {target_destination}.")
        return

    average_journey = sum(all_journey_times) / len(all_journey_times)
    print(f"Total vehicles arriving at Node {target_destination}: {len(all_journey_times)}")
    print(f"Average journey time to Node {target_destination}: {average_journey:.2f} seconds")

    plt.figure(figsize=(10, 6))
    
    for src_node, axes in sorted(plot_data.items()):
        plt.scatter(
            axes['destroy_times'], 
            axes['journey_times'], 
            marker='o', 
            alpha=0.7, 
            edgecolors='black',
            label=f'Source Node {src_node}'
        )
    
    plt.xlabel('Simulation Time (seconds)', fontsize=12)
    plt.ylabel('Journey Time (seconds)', fontsize=12)
    plt.title(f'Journey Times for Vehicles terminating at Destination Node {target_destination}', fontsize=14)
    plt.grid(True, linestyle='--', alpha=0.7)
    
    plt.legend(title="Source Node")
    
    plt.tight_layout()
    plt.show()

if __name__ == "__main__":
    log_file_name = 'simulation_log.jsonl' 
    plot_journey_times(log_file_name, target_destination=7)