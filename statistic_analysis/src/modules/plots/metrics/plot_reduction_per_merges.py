import pandas as pd
import numpy as np
from datetime import datetime

from matplotlib import pyplot as plt

plt.style.use('ggplot')

CSV_FILE = "../../output/all-complexities-2021-05-09-11-12-32.csv"
ADAPTED_CSV_FILE = "../../output/all-metrics-2021-05-09-11-12-32.csv"

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
    "Initial Invocations Count W/ Empties",
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
features_to_plot = ["CLIP", "CRIP", "CROP", "CWOP", "CIP", "CDDIP", "COP", "CPIF"]

merges_row = dataset["Total Invocation Merges"]
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
    "merges": [],
    "color": "red",
}

other_clusters = {
    "metrics": {},
    "reductions": [],
    "merges": [],
    "color": "cornflowerblue",
}

for metric in features_to_plot:
    best_clusters["metrics"][metric] = []
    other_clusters["metrics"][metric] = []

def set_metrics(cluster_dict, dataset, idx):
    reduction_percentage = (reduction_row[idx] * 100)/initial_row[idx]

    if reduction_percentage <= 0:
        return

    cluster_dict["reductions"].append(reduction_percentage)

    cluster_dict["merges"].append(merges_row[idx])

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


# PLOT REDUCTION PER MERGES PLOTS
fig, ax = plt.subplots(1, 1, figsize=(4, 4))

def func(x, a, b, c, d, g):
    return ( ( (a-d) / ( (1+( (x/c)** b )) **g) ) + d )

from scipy.optimize import curve_fit
popt, _ = curve_fit(func, best_clusters["merges"], best_clusters["reductions"])

a, b, c, d, g = popt

# define a sequence of inputs between the smallest and largest known inputs
from numpy import arange
x_line = arange(min(best_clusters["merges"]), max(best_clusters["merges"]), 1)
# calculate the output for the range
y_line = func(x_line, a, b, c, d, g)
# create a line plot for the mapping function
ax.plot(x_line, y_line, '--', color='red')


# ax.plot(best_clusters["merges"], func(best_clusters["merges"], *popt), 'r-', label="Fitted Curve")

ax.scatter(best_clusters["merges"], best_clusters["reductions"], s=size, color=other_clusters["color"])
# ax.scatter(other_clusters["merges"], other_clusters["reductions"], s=size, color=other_clusters["color"], label="bad cluster")

ax.set_xlabel("Merge Operations", fontsize=10)
ax.set_ylabel("FRC reduction %", fontsize=10)

ax.set_xlim(0, 200)
ax.set_ylim(0, 100)

ax.legend()
ax.set_axisbelow(True)
ax.grid(True)

fig.tight_layout()

invocations_reduction = (dataset["Total Invocation Merges"].mean() * 100)/dataset["Initial Invocations Count"].mean()
print(invocations_reduction)
plt.show()
