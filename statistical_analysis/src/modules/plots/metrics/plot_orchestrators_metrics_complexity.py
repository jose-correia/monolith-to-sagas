import pandas as pd
import numpy as np
from datetime import datetime

from matplotlib import pyplot as plt

plt.style.use('ggplot')

CSV_FILE = "../../output/all-complexities-2021-05-04-22-28-27.csv"
ADAPTED_CSV_FILE = "../../output/all-metrics-2021-05-04-22-28-27.csv"

CSV_ROWS = [
    "Codebase",
    "Feature",
    "Orchestrator",
    "Entities",
    "Initial System Complexity",
    "Final System Complexity",
    "System Complexity Reduction",
    "Initial Functionality Complexity",
    "Final Functionality Complexity",
    "Functionality Complexity Reduction",
    "Initial Invocations Count",
    "Final Invocations Count",
    "Total Invocation Merges",
    "Total Trace Sweeps w/ Merges",
    "Clusters with multiple invocations",
    "CLIP",
    "CRIP",
    "CROP",
    "CWOP",
    "CIP",
    "CDDIP",
    "COP",
    "CPIF",
    "CIOF",
    "SCCP",
    "FCCP",
]

ADAPTED_CSV_ROWS = [
    "Codebase",
    "Feature",
    "Cluster",
    "CLIP",
    "CRIP",
    "CROP",
    "CWOP",
    "CIP",
    "CDDIP",
    "COP",
    "CPIF",
    "CIOF",
    "SCCP",
    "FCCP", 
    "Orchestrator",
]

#PLOT_SPECIFIC_CODEBASE = "ldod-static"
PLOT_SPECIFIC_CODEBASE = None
# PLOT_SPECIFIC_FEATURE = "AdminController.removeTweets"
PLOT_SPECIFIC_FEATURE = None

ONLY_SHOW_BEST_AND_WORST = True
SHOW_SYSTEM_COMPLEXITY_REDUCTION = False


dataset = pd.read_csv(CSV_FILE, names=CSV_ROWS, skiprows=1)
adapted_dataset = pd.read_csv(ADAPTED_CSV_FILE, names=ADAPTED_CSV_ROWS, skiprows=1)

size = 6
if PLOT_SPECIFIC_FEATURE:
    size = 20
    dataset = dataset.query(f'Feature == "{PLOT_SPECIFIC_FEATURE}"')
elif PLOT_SPECIFIC_CODEBASE:
    size = 16
    dataset = dataset.query(f'Codebase == "{PLOT_SPECIFIC_CODEBASE}"')


# features_to_plot = ["CLIP", "CRIP", "CROP", "CWOP", "CIP", "CDDIP", "COP", "CPIF", "CIOF"]
features_to_plot = ["CLIP", "CRIP", "COP", "SCCP"]

label_map = {
    "CLIP": "Lock Invocation Probability",
    "CRIP": "Read Invocation Probability",
    "COP": "Access Probability",
    "SCCP": "System Complexity Contribution",
}

if not SHOW_SYSTEM_COMPLEXITY_REDUCTION:
    initial_row = dataset["Initial Functionality Complexity"]
    reduction_row = dataset["Functionality Complexity Reduction"]
    x_values = dataset["Final Functionality Complexity"]
else:
    initial_row = dataset["Initial System Complexity"]
    reduction_row = dataset["System Complexity Reduction"]
    x_values = dataset["Final System Complexity"]

best_clusters = {
    "metrics": {},
    "reductions": [],
    "color": "cornflowerblue",
    "regression_color": "red",
}

other_clusters = {
    "metrics": {},
    "reductions": [],
    "color": "cornflowerblue",
}

for metric in features_to_plot:
    best_clusters["metrics"][metric] = []
    other_clusters["metrics"][metric] = []

def set_metrics(cluster_dict, dataset, idx):
    reduction_percentage = (reduction_row[idx] * 100)/initial_row[idx]
    cluster_dict["reductions"].append(reduction_percentage)

    for metric in features_to_plot:
        cluster_dict["metrics"][metric].append(dataset[metric][idx])


last_feature = ""
index = 0
for idx in dataset.index:
    adapted_cluster = adapted_dataset.query(f'Feature == "{dataset["Feature"][idx]}" and Cluster == {dataset["Orchestrator"][idx]}')
    if len(adapted_cluster.index) == 0:
        continue

    if adapted_cluster["Orchestrator"][adapted_cluster.index[0]] == 1:
        set_metrics(best_clusters, dataset, idx)
    else:
        set_metrics(other_clusters, dataset, idx)
    
    index = idx

print(len(best_clusters["reductions"]))
print(len(other_clusters["reductions"]))

row = 0
column = 0
best_x = np.array(best_clusters["reductions"])
other_x = np.array(other_clusters["reductions"])

for idx, feature in enumerate(features_to_plot):
    fig, ax = plt.subplots(1, 1, figsize=(4, 4))
    best_y = np.array(best_clusters["metrics"][feature])
    best_m, best_b = np.polyfit(best_x, best_y, 1)
    ax.plot(best_x, best_m*best_x + best_b, '--', color=best_clusters["regression_color"])

    # other_y = np.array(other_clusters["metrics"][feature])
    # other_m, other_b = np.polyfit(other_x, other_y, 1)
    # ax.plot(other_x, other_m*other_x + other_b, color=other_clusters["color"])


    ax.scatter(best_clusters["reductions"], best_clusters["metrics"][feature], s=size, color=best_clusters["color"])
    #ax.scatter(other_clusters["reductions"], other_clusters["metrics"][feature], s=size, color=other_clusters["color"], label="other cluster")

    ax.set_xlabel("FRC reduction %", fontsize=10)

    label = label_map.get(feature) if label_map.get(feature) else feature
    ax.set_ylabel(label, fontsize=10)

    ax.set_xlim(0, 100)
    ax.set_ylim(-0.05, 1.05)

    # ax.legend()
    ax.set_axisbelow(True)
    ax.grid(True)

    if column == 2:
        column = 0
        row += 1
    else:
        column += 1

    fig.tight_layout()

plt.show()
