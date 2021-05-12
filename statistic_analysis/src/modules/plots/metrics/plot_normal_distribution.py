# importing numpy as np
import numpy as np
 
# importing pyplot as plt
import matplotlib.pyplot as plt
import scipy.stats

import pandas as pd
import numpy as np

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

dataset = pd.read_csv(CSV_FILE, names=CSV_ROWS, skiprows=1)
adapted_dataset = pd.read_csv(ADAPTED_CSV_FILE, names=ADAPTED_CSV_ROWS, skiprows=1)

size = 6

merges_row = dataset["Total Invocation Merges"]
initial_invocations_row = dataset["Initial Invocations Count W/ Empties"]

initial_row = dataset["Initial Functionality Complexity"]
reduction_row = dataset["Functionality Complexity Reduction"]
x_values = dataset["Final Functionality Complexity"]

best_clusters = {
    "reductions": [],
    "merges": [],
    "color": "red",
}

other_clusters = {
    "reductions": [],
    "merges": [],
    "color": "cornflowerblue",
}

def set_metrics(cluster_dict, dataset, idx):
    reduction_percentage = (reduction_row[idx] * 100)/initial_row[idx]

    if reduction_percentage <= 0:
        return

    cluster_dict["reductions"].append(reduction_percentage)

    cluster_dict["merges"].append((merges_row[idx]*100)/initial_invocations_row[idx])


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


values = np.array(best_clusters["reductions"])

y = scipy.stats.norm.pdf(values,values.mean(),values.std())

print(values.std())
plt.plot(values,y, color='coral')

plt.grid()

plt.xlim(0,100)
plt.ylim(0,0.02)

plt.title('How to plot a normal distribution in python with matplotlib',fontsize=10)

plt.xlabel('x')
plt.ylabel('Normal Distribution')


plt.show()


# # plotting histograph
# plt.hist(values, 1000)

# # plotting mean line
# plt.axvline(values.mean(), color='k', linestyle='dashed', linewidth=2)

# # showing the plot
# plt.show()