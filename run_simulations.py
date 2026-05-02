import os
import shutil
import subprocess
import time

total_runs = 5
target_dir = "new_logs2"

os.makedirs(target_dir, exist_ok=True)

overall_start = time.time()

for i in range(1, total_runs + 1):
    print(f"Starting run {i} of {total_runs}...")
    run_start = time.time()

    while True:
        result = subprocess.run([
            "go", "run", ".",
            "example_maps/world7_old.json",
            "NUM_RUNS=216000",
            "MAX_VEHICLES=1000",
            "SIM_RATE=10000"
        ])

        if result.returncode == 0:
            break
        else:
            print(f"Run {i} failed (exit code {result.returncode}), retrying...")

    run_elapsed = time.time() - run_start
    print(f"Run {i} took {run_elapsed:.1f}s ({run_elapsed/60:.2f} min)")

    source_file = "simulation_log.jsonl"
    dest_file = os.path.join(target_dir, f"RUN{i}lookahead1_25%2.jsonl")

    if os.path.exists(source_file):
        shutil.move(source_file, dest_file)
        print(f"Successfully moved log to {dest_file}")
    else:
        print(f"Warning: {source_file} was not found after run {i}!")

    print(f"Run {i} completed.\n")

total_elapsed = time.time() - overall_start
print(f"All {total_runs} runs completed in {total_elapsed:.1f}s ({total_elapsed/60:.2f} min)")