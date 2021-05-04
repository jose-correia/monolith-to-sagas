import pandas as pd
import numpy as np
from datetime import datetime

from matplotlib import pyplot as plt
import seaborn as sns

plt.style.use('ggplot')

CSV_FILE = "../output/all-metrics-2021-05-02-23-50-02.csv"
CSV_ROWS = ["Codebase", "Feature", "Cluster", "CLIP", "CRIP", "CROP", "CWOP", "CIP", "CDDIP", "COP", "CPIF", "CIOF", "Orchestrator"]
PLOT_METRICS_INDIVIDUALLY = True

dataset = pd.read_csv(CSV_FILE, names=CSV_ROWS, skiprows=1)

features_to_plot = ["CLIP", "CRIP", "CROP", "CWOP", "CIP", "CDDIP", "COP", "CPIF", "CIOF"]

# Transform int to id
dataset[["Feature"]] = dataset[["Feature"]].apply(lambda x: pd.factorize(x)[0] + 1)

X = dataset.drop('Orchestrator', axis=1).drop("Codebase", axis=1).drop("Cluster", axis=1).drop("Feature", axis=1)
y = dataset['Orchestrator']

from sklearn.model_selection import train_test_split
X_train, X_test, y_train, y_test = train_test_split(X, y, test_size = 0.30)


from sklearn.svm import SVC
# svclassifier = SVC(kernel='linear')
svclassifier = SVC(kernel='poly', degree=8)
#svclassifier = SVC(kernel='rbf')
#svclassifier = SVC(kernel='sigmoid')
svclassifier.fit(X_train, y_train)

y_pred = svclassifier.predict(X_test)

from sklearn.metrics import classification_report, confusion_matrix
print(confusion_matrix(y_test,y_pred))
print(classification_report(y_test,y_pred))
