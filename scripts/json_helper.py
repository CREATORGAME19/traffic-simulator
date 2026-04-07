import json
import math

def calculate_lane_offsets(input_filename, output_filename, total_gap=5):
    # The offset for each individual lane is half the total gap
    offset_dist = total_gap / 2.0 

    with open(input_filename, 'r') as f:
        data = json.load(f)

    # Create a dictionary of nodes for quick coordinate lookups
    nodes = {node['ID']: node['Position'] for node in data['road_nodes']}

    for lane in data['lanes']:
        from_id = lane['From_Node']
        to_id = lane['To_Node']
        
        pos_from = nodes[from_id]
        pos_to = nodes[to_id]
        
        # 1. Calculate direction vector
        dx = pos_to['x'] - pos_from['x']
        dy = pos_to['y'] - pos_from['y']
        
        # 2. Calculate length of the vector
        length = math.hypot(dx, dy)
        if length == 0:
            continue # Failsafe for nodes sitting on the exact same coordinate
            
        # 3. Normalize the vector
        ux = dx / length
        uy = dy / length
        
        # 4. Find the perpendicular right-hand normal vector 
        # (Based on your original JSON's lane shift direction)
        nx = -uy
        ny = ux
        
        # 5. Apply the 0.75 offset
        offset_x = nx * offset_dist
        offset_y = ny * offset_dist
        
        # 6. Update the lane's start and end positions, rounded to 4 decimal places
        lane['Start_Position']['x'] = round(pos_from['x'] + offset_x, 4)
        lane['Start_Position']['y'] = round(pos_from['y'] + offset_y, 4)
        lane['End_Position']['x'] = round(pos_to['x'] + offset_x, 4)
        lane['End_Position']['y'] = round(pos_to['y'] + offset_y, 4)

    with open(output_filename, 'w') as f:
        json.dump(data, f, indent=2)
    
    print(f"Successfully updated lane gaps to {total_gap} units. Saved to {output_filename}")

if __name__ == "__main__":
    calculate_lane_offsets('../example_maps/world7.json', '../example_maps/world_updated.json')
