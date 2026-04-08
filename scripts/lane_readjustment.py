import json
import math

def setback_lanes(input_filename, output_filename, pullback_dist=3.5):
    with open(input_filename, 'r') as f:
        data = json.load(f)

    for lane in data['lanes']:
        sx = lane['Start_Position']['x']
        sy = lane['Start_Position']['y']
        ex = lane['End_Position']['x']
        ey = lane['End_Position']['y']
        
        # 1. Calculate direction vector of the lane
        dx = ex - sx
        dy = ey - sy
        
        # 2. Calculate the total length of the lane
        length = math.hypot(dx, dy)
        
        # Failsafe: If a lane is very short, don't pull it back so far that it inverts
        if length <= 2 * pullback_dist:
            print(f"Warning: Lane {lane['ID']} is too short to pullback by {pullback_dist}. Skipping.")
            continue
            
        # 3. Normalize the vector (length of 1)
        ux = dx / length
        uy = dy / length
        
        # 4. Apply the pullback offset
        # Move the start position FORWARD along the vector
        lane['Start_Position']['x'] = round(sx + (ux * pullback_dist), 4)
        lane['Start_Position']['y'] = round(sy + (uy * pullback_dist), 4)
        
        # Move the end position BACKWARD along the vector
        lane['End_Position']['x'] = round(ex - (ux * pullback_dist), 4)
        lane['End_Position']['y'] = round(ey - (uy * pullback_dist), 4)

    with open(output_filename, 'w') as f:
        json.dump(data, f, indent=2)
    
    print(f"Successfully pulled back lane coordinates by {pullback_dist} units. Saved to {output_filename}")

if __name__ == "__main__":
    # Save the JSON you provided as 'world7.json' in the same folder as this script
    setback_lanes('../example_maps/world7.json', '../example_maps/world7_setback.json', pullback_dist=5.5)