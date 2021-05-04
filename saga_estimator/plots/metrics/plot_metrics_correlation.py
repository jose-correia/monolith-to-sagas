import pandas as pd
import numpy as np
from datetime import datetime

from matplotlib import pyplot as plt
import seaborn as sns

plt.style.use('ggplot')

CSV_FILE = "../../output/estimator_with_metrics_all_codebases.csv"
CSV_ROWS = ["Codebase", "Feature", "Cluster", "CLIP", "CRIP", "CROP", "CWOP", "CIP", "CDDIP", "COP", "CPIF", "CIOF", "Orchestrator"]
PLOT_METRICS_INDIVIDUALLY = True

dataset = pd.read_csv(CSV_FILE, names=CSV_ROWS, skiprows=1)

features_to_plot = ["CLIP", "CRIP", "CROP", "CWOP", "CIP", "CDDIP", "COP", "CPIF", "CIOF"]

corr = dataset.corr()

sns.heatmap(corr, 
            xticklabels=corr.columns.values,
            yticklabels=corr.columns.values)

plt.show()