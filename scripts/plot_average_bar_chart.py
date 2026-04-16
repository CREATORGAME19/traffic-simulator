import json
import os
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
import statistics

def plot_journey_times(log_file_paths):
    plot_data = {}
    all_journey_times = []

    # Loop through each provided log file
    for log_file_path in log_file_paths:
        run_name = os.path.basename(log_file_path)
        plot_data[run_name] = {'journey_times': []}
        
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
                                journey_time = current_time - spawn_time
                                
                                run_journey_times.append(journey_time)
                                all_journey_times.append(journey_time)
                                
                                plot_data[run_name]['journey_times'].append(journey_time)

                    except json.JSONDecodeError:
                        print(f"Skipping invalid JSON line in {run_name}: {line}")
                        continue
            
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

    # Prepare data for the bar chart
    run_names = []
    averages = []
    std_devs = []

    for run_name, axes in plot_data.items():
        j_times = axes['journey_times']
        if j_times:
            run_names.append(run_name)
            avg = sum(j_times) / len(j_times)
            averages.append(avg)
            sd = statistics.stdev(j_times) if len(j_times) > 1 else 0.0
            std_devs.append(sd)

    # --- COLOR MAPPING & GROUPING LOGIC ---
    color_map = {
        'Baseline': '#7F8C8D', # Gray
        '100%': '#2ECC71',     # Green
        '50%': '#F39C12',      # Orange
        '25%': '#E74C3C'       # Red
    }
    
    bar_colors = []
    x_positions = []
    current_x = 0
    previous_group = None
    
    # Dictionary to track the exact x-coordinates of bars in each group
    group_x_coords = {}

    for name in run_names:
        # 1. Determine the Group (for spacing and labeling)
        if '1lookahead' in name:
            current_group = '1 Lookahead'
        elif '2lookahead' in name:
            current_group = '2 Lookahead'
        elif '4lookahead' in name:
            current_group = '4 Lookahead'
        else:
            current_group = 'Baseline'

        # Add a gap if switching groups
        if previous_group is not None and current_group != previous_group:
            current_x += 1.0  

        x_positions.append(current_x)
        
        # Track the coordinate for this specific group
        if current_group not in group_x_coords:
            group_x_coords[current_group] = []
        group_x_coords[current_group].append(current_x)

        current_x += 1.0
        previous_group = current_group

        # 2. Determine the Percentage (for color)
        if '100%' in name:
            bar_colors.append(color_map['100%'])
        elif '50%' in name:
            bar_colors.append(color_map['50%'])
        elif '25%' in name:
            bar_colors.append(color_map['25%'])
        else:
            bar_colors.append(color_map['Baseline'])
    # ------------------------------------------------------------

    plt.figure(figsize=(14, 7))
    
    # Create the bars
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
    plt.ylabel('Average Journey Time (seconds)', fontsize=14)
    #plt.title('Average Journey Times by Lookahead & Adoption Rate', fontsize=16)
    
    plt.grid(axis='y', linestyle='--', alpha=0.7, zorder=0)
    
    # --- CUSTOM CENTERED X-TICKS ---
    tick_positions = []
    tick_labels = []
    
    # Calculate the center point for each group
    for g_name, coords in group_x_coords.items():
        center_x = sum(coords) / len(coords)
        tick_positions.append(center_x)
        tick_labels.append(g_name)
        
    # Apply the centered labels (no rotation needed since they are short!)
    plt.xticks(tick_positions, tick_labels, fontsize=12)
    # -------------------------------
    
    # --- CUSTOM LEGEND (Moved to lower right) ---
    legend_handles = [
        mpatches.Patch(color=color_map['Baseline'], label='Baseline (0%)'),
        mpatches.Patch(color=color_map['100%'], label='100% V2V Adoption'),
        mpatches.Patch(color=color_map['50%'], label='50% V2V Adoption'),
        mpatches.Patch(color=color_map['25%'], label='25% V2V Adoption')
    ]
    plt.legend(handles=legend_handles, title="Adoption Rate", fontsize=11, title_fontsize=12, loc='lower right')
    # --------------------------------------------

    # Add the exact average value on top of each bar
    for bar in bars:
        yval = bar.get_height()
        plt.text(
            bar.get_x() + bar.get_width()/2, 
            yval + (max(std_devs) * 0.1 if std_devs else 2),
            f'{yval:.1f}', 
            ha='center', 
            va='bottom', 
            fontsize=10, 
            fontweight='bold'
        )

    plt.tight_layout()
    plt.show()

if __name__ == "__main__":
    # Ensure your files are ordered exactly how you want them grouped left-to-right!
    log_file_names = [
        '../logs/6hourRun3.jsonl',
        '../logs/1lookahead100%3.jsonl',
        '../logs/1lookahead50%3.jsonl',
        '../logs/1lookahead25%2.jsonl',
        '../logs/2lookahead100%3.jsonl',
        '../logs/2lookahead50%3.jsonl',
        '../logs/2lookahead25%2.jsonl',
        '../logs/4lookahead100%4.jsonl',
        '../logs/4lookahead50%3.jsonl',
        '../logs/4lookahead25%2.jsonl',
    ] 
    plot_journey_times(log_file_names)