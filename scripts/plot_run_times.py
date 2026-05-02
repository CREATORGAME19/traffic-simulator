import matplotlib.pyplot as plt
import numpy as np
import statistics

groups = {
    'With Visualiser': [110.1, 107.4, 108.3, 107.9, 107.6],
    'Without Visualiser': [590.9, 509.5, 492.4, 488.5, 541.4],
}

colors = ['steelblue', 'darkorange']

for label, times in groups.items():
    mean_m = statistics.mean(times) / 60
    std_m  = (statistics.stdev(times) if len(times) > 1 else 0.0) / 60
    print(f"[{label}] n={len(times)} | Mean: {mean_m:.2f} min | Std: {std_m:.3f} min")

fig, ax = plt.subplots(figsize=(8, 6))

for i, ((label, times), color) in enumerate(zip(groups.items(), colors)):
    mean_m = statistics.mean(times) / 60
    std_m  = (statistics.stdev(times) if len(times) > 1 else 0.0) / 60
    ax.bar(i, mean_m, yerr=std_m, capsize=10,
           color=color, edgecolor='black', alpha=0.85, width=0.5, zorder=3,
           label=label)
    ax.text(i, mean_m + std_m + 0.05, f'{mean_m:.2f}',
            ha='center', va='bottom', fontsize=13, fontweight='bold')

ax.set_xticks(np.arange(len(groups)))
ax.set_xticklabels(list(groups.keys()), fontsize=14)
ax.tick_params(axis='y', labelsize=14)
ax.set_ylabel('Average Run Time (minutes)', fontsize=16)
ax.set_ylim(bottom=0)
ax.grid(axis='y', linestyle='--', alpha=0.6, zorder=0)
plt.tight_layout()
plt.show()
