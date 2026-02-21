import json
import matplotlib.pyplot as plt

def plot_journey_times(log_file_path, target_destination=7):
    # Dictionaries to track vehicle properties
    spawn_times = {}
    source_nodes = {}
    destination_nodes = {}
    
    # We will group plot data by source_node_id
    # Format: { source_node_id: {'destroy_times': [], 'journey_times': []} }
    plot_data = {}
    
    # Track all journey times for the overall average (filtered by destination)
    all_journey_times = []

    # Read the JSONL (JSON Lines) file
    with open(log_file_path, 'r') as file:
        for line in file:
            line = line.strip()
            if not line:
                continue
                
            try:
                data = json.loads(line)
                
                # Extract common properties
                vid = data.get("vehicle_id")
                current_time = data.get("time")
                event_type = data.get("event_type", "").upper()
                
                # Track vehicle properties when it appears in the log
                if vid is not None:
                    # Track spawn time
                    if "spawn_time" in data:
                        spawn_times[vid] = data["spawn_time"]
                    elif event_type in ["VEHICLE SPAWN", "VEHICLE_SPAWN"]:
                        spawn_times[vid] = current_time
                        
                    # Track source node
                    if "source_node_id" in data:
                        source_nodes[vid] = data["source_node_id"]
                        
                    # Track destination node
                    if "destination_node_id" in data:
                        destination_nodes[vid] = data["destination_node_id"]

                # Detect vehicle destroy events
                if event_type in ["VEHICLE DESTROY", "VEHICLE_DESTROY"]:
                    # Fallback to dictionaries if the event itself is missing the data
                    spawn_time = data.get("spawn_time", spawn_times.get(vid))
                    src_node = data.get("source_node_id", source_nodes.get(vid))
                    dest_node = data.get("destination_node_id", destination_nodes.get(vid))
                    
                    # --- FILTER: Only process if the destination matches our target ---
                    if dest_node != target_destination:
                        continue
                    
                    # If we have all required info, calculate and store the journey
                    if spawn_time is not None and current_time is not None and src_node is not None:
                        journey_time = current_time - spawn_time
                        
                        # Add to the overall list
                        all_journey_times.append(journey_time)
                        
                        # Create a new group for this source node if it doesn't exist
                        if src_node not in plot_data:
                            plot_data[src_node] = {'destroy_times': [], 'journey_times': []}
                            
                        # Add to the group
                        plot_data[src_node]['destroy_times'].append(current_time)
                        plot_data[src_node]['journey_times'].append(journey_time)

            except json.JSONDecodeError:
                print(f"Skipping invalid JSON line: {line}")
                continue
                
    if not all_journey_times:
        print(f"No VEHICLE_DESTROY events found ending at destination node {target_destination}.")
        return

    # Calculate and print the overall average for this specific destination
    average_journey = sum(all_journey_times) / len(all_journey_times)
    print(f"Total vehicles arriving at Node {target_destination}: {len(all_journey_times)}")
    print(f"Average journey time to Node {target_destination}: {average_journey:.2f} seconds")

    # Plot the results
    plt.figure(figsize=(10, 6))
    
    # Plot each source node group separately to assign colors and build the legend
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
    
    # Add a legend to show which color is which source node
    plt.legend(title="Source Node")
    
    plt.tight_layout()
    plt.show()

if __name__ == "__main__":
    # Replace 'logs.json' with the actual path to your log file
    log_file_name = 'simulation_log.jsonl' 
    plot_journey_times(log_file_name, target_destination=7)