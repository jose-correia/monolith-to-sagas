import pandas as pd
import numpy as np
from datetime import datetime

from matplotlib import pyplot as plt

plt.style.use('ggplot')

CSV_FILE = "../../output/all-metrics-2021-05-02-23-50-02.csv"
CSV_ROWS = ["Codebase", "Feature", "Cluster", "CLIP", "CRIP", "CROP", "CWOP", "CIP", "CDDIP", "COP", "CPIF", "CIOF", "Orchestrator"]
PLOT_METRICS_INDIVIDUALLY = True

dataset = pd.read_csv(CSV_FILE, names=CSV_ROWS, skiprows=1)

features_to_plot = ["CLIP", "CRIP", "CROP", "CWOP", "CIP", "CDDIP", "COP", "CPIF"]

orchestrator_features = dict(csv_indexes=[])
non_orchestrator_features = dict(csv_indexes=[])

for feature in features_to_plot:
    orchestrator_features[feature] = []
    non_orchestrator_features[feature] = []

orchestrator_data = dataset.query("Orchestrator == 1")
for index in orchestrator_data.index:
    orchestrator_features["csv_indexes"].append(index)
    for feature in features_to_plot:
        orchestrator_features[feature].append(dataset[feature][index])

non_orchestrator_data = dataset.query("Orchestrator != 1")
for index in non_orchestrator_data.index:
    non_orchestrator_features["csv_indexes"].append(index)
    for feature in features_to_plot:
        non_orchestrator_features[feature].append(dataset[feature][index])


orchestrator_ids = ["Orchestrator" for _ in orchestrator_data.index]
non_orchestrator_ids = ["Not orchestrator" for _ in non_orchestrator_data.index]


if PLOT_METRICS_INDIVIDUALLY:
    row = 0
    column = 0

    fig, ax = plt.subplots(2, 4, figsize=(18, 8))
    for idx, feature in enumerate(features_to_plot):
        ax[row][column].scatter(orchestrator_ids, orchestrator_features[feature], s=10, color="red")
        ax[row][column].scatter(non_orchestrator_ids, non_orchestrator_features[feature], s=10, color="blue")

        ax[row][column].set_xlabel("Cluster type", fontsize=10)
        ax[row][column].set_ylabel(feature, fontsize=10)

        ax[row][column].set_axisbelow(True)
        ax[row][column].grid(True)

        if column == 3:
            column = 0
            row += 1
        else:
            column += 1

    fig.tight_layout()

else:
    for idx, feature in enumerate(features_to_plot):
        fig, ax = plt.subplots(2, 4, figsize=(18, 8))

        row = 0
        column = 0

        for ix2, feature_2 in enumerate(features_to_plot):
            if feature == feature_2:
                continue

            ax[row][column].scatter(orchestrator_features[feature], orchestrator_features[feature_2], s=10, color="red", label="orchestrator")
            ax[row][column].scatter(non_orchestrator_features[feature], non_orchestrator_features[feature_2], s=10, color="blue", label="not orchestrator")

            ax[row][column].set_xlabel(feature_2, fontsize=10)
            ax[row][column].set_ylabel(feature, fontsize=10)

            ax[row][column].set_axisbelow(True)
            ax[row][column].grid(True)
            ax[row][column].legend()

            if column == 3:
                column = 0
                row += 1
            else:
                column += 1
        
        fig.tight_layout()

plt.show()
