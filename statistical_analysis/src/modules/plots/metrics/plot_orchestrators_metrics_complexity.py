import numpy as np
from matplotlib import pyplot as plt
from scipy.stats import gaussian_kde

plt.style.use('ggplot')



def plot_metric_per_complexity_reduction(data):
    # features_to_plot = ["CLIP", "CRIP", "CROP", "CWOP", "CIP", "CDDIP", "COP", "CPIF", "CIOF"]
    features_to_plot = ["CLIP", "CRIP", "COP", "SCCP"]

    label_map = {
        "CLIP": "Lock Invocation Probability",
        "CRIP": "Read Invocation Probability",
        "COP": "Access Probability",
        "SCCP": "System Complexity Contribution",
    }

    best_x = np.array(best_clusters["reductions"])
    other_x = np.array(other_clusters["reductions"])
    for idx, feature in enumerate(features_to_plot):
        fig, ax = plt.subplots(1, 1, figsize=(4, 4))

        best_y = np.array(best_clusters["metrics"][feature])
        best_m, best_b = np.polyfit(best_x, best_y, 1)
        ax.plot(best_x, best_m*best_x + best_b, '--', color=best_clusters["regression_color"])

        # Calculate the point density
        xy = np.vstack([best_x, best_y])
        z = gaussian_kde(xy)(xy)

        ax.scatter(best_clusters["reductions"], best_clusters["metrics"][feature],  c=z, s=6)

        ax.set_xlabel("FRC reduction %", fontsize=10)

        label = label_map.get(feature) if label_map.get(feature) else feature
        ax.set_ylabel(label, fontsize=10)

        ax.set_xlim(0, 100)
        ax.set_ylim(-0.05, 1.05)

        # ax.legend()
        ax.set_axisbelow(True)
        ax.grid(True)

        fig.tight_layout()

    plt.show()
