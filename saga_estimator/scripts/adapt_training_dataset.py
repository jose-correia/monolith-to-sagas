# This script changes the training dataset exported by the tool and
# sets as a possible orchestrator all the clusters that add a final complexity
# with a distance not far away from the best possible orchestrator
import pandas as pd


ACCEPTABLE_COMPLEXITY_DISTANCE = 0.30

TRAINING_FILE = "../output/ml_last.csv"
COMPLEXITIES_FILE = "../output/complexities_last.csv"


training_dataset = pd.read_csv(TRAINING_FILE)
complexities_dataset = pd.read_csv(COMPLEXITIES_FILE, usecols = ["Feature", "Orchestrator", "Final Functionality Complexity"])

print(training_dataset)

for idx in training_dataset.index:
    functionality = training_dataset["Feature"][idx]
    cluster = int(training_dataset["Cluster"][idx])

    functionality_complexities = complexities_dataset.query(f'Feature == "{functionality}"')
    if len(functionality_complexities.index) == 0:
        continue

    orchestrator_id = functionality_complexities["Orchestrator"][functionality_complexities.index[0]]
    orchestrator_complexity = functionality_complexities["Final Functionality Complexity"][functionality_complexities.index[0]]
    worst_cluster_complexity = functionality_complexities["Final Functionality Complexity"][functionality_complexities.index[-1]]

    if training_dataset["Cluster"][idx] == orchestrator_id:
        training_dataset.at[idx, "Orchestrator"] = 1
        continue
    
    complexities_row = complexities_dataset.query(f'Feature == "{functionality}" and Orchestrator == {cluster}')
    cluster_complexity = complexities_row["Final Functionality Complexity"][complexities_row.index[0]]

    if worst_cluster_complexity - orchestrator_complexity == 0:
        training_dataset.at[idx, "Orchestrator"] = 1
    else:
        distance = (cluster_complexity - orchestrator_complexity)/(worst_cluster_complexity - orchestrator_complexity)
        if distance <= ACCEPTABLE_COMPLEXITY_DISTANCE:
            training_dataset.at[idx, "Orchestrator"] = 1


training_dataset.to_csv(f'../output/adapted_{ACCEPTABLE_COMPLEXITY_DISTANCE}.csv', index=False, header=True)
