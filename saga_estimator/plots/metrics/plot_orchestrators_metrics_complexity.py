import pandas as pd
import numpy as np
from datetime import datetime

from matplotlib import pyplot as plt

plt.style.use('ggplot')

CSV_FILE = "../../output/all-complexities-2021-05-02-23-50-02.csv"
ADAPTED_CSV_FILE = "../../output/all-metrics-2021-05-02-23-50-02.csv"

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
features_to_plot = ["CLIP", "CRIP", "CROP", "CWOP", "CIP", "CDDIP", "COP", "CPIF"]

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
    "color": "red",
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
    # if dataset["Feature"][idx] != last_feature:
    #     set_metrics(best_clusters, dataset, idx)

    #     if ONLY_SHOW_BEST_AND_WORST:
    #         if last_feature != "":
    #             set_metrics(other_clusters, dataset, idx-1)

    #     last_feature = dataset["Feature"][idx]

    # elif not ONLY_SHOW_BEST_AND_WORST:
    #     set_metrics(other_clusters, dataset, idx)
    
    index = idx

print(len(best_clusters["reductions"]))
print(len(other_clusters["reductions"]))

row = 0
column = 0
best_x = np.array(best_clusters["reductions"])
other_x = np.array(other_clusters["reductions"])

fig, ax = plt.subplots(2, 4, figsize=(18, 8))
for idx, feature in enumerate(features_to_plot):
    best_y = np.array(best_clusters["metrics"][feature])
    best_m, best_b = np.polyfit(best_x, best_y, 1)
    ax[row][column].plot(best_x, best_m*best_x + best_b, color=best_clusters["color"])

    other_y = np.array(other_clusters["metrics"][feature])
    other_m, other_b = np.polyfit(other_x, other_y, 1)
    ax[row][column].plot(other_x, other_m*other_x + other_b, color=other_clusters["color"])



    ax[row][column].scatter(best_clusters["reductions"], best_clusters["metrics"][feature], s=size, color=best_clusters["color"], label="best cluster")
    ax[row][column].scatter(other_clusters["reductions"], other_clusters["metrics"][feature], s=size, color=other_clusters["color"], label="other cluster")

    ax[row][column].set_xlabel("FRC reduction %", fontsize=10)
    ax[row][column].set_ylabel(feature, fontsize=10)

    ax[row][column].legend()
    ax[row][column].set_axisbelow(True)
    ax[row][column].grid(True)

    if column == 3:
        column = 0
        row += 1
    else:
        column += 1

    fig.tight_layout()

plt.show()
