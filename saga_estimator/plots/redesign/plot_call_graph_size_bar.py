import matplotlib.pyplot as plt
import numpy as np

graph_size_data = (
    (14, 6),
    (11, 4),
    (108, 12),
    (80, 7),
    (6, 4),
    (60, 6),
    (17, 3),
    (43, 6),
    (33, 3),
)

plt.style.use('ggplot')

fig, ax = plt.subplots()

dim = len(graph_size_data[0])
w = 0.75
dimw = w / dim

x = np.arange(len(graph_size_data))
for i in range(len(graph_size_data[0])):
    y = [d[i] for d in graph_size_data]
    label = "initial" if i == 0 else "final"
    color = "coral" if label == "initial" else "cornflowerblue"
    b = ax.bar(x + i * dimw, y, dimw, bottom=0, label=label, color=color)

ax.set_xticks(x + dimw / 2)
ax.set_xticklabels(map(str, x+1))
ax.set_yscale('log')

ax.set_xlabel('Functionality ID', fontsize=13)
ax.set_ylabel('Call graph size', fontsize=13)

for patch in ax.patches:
    ax.annotate(str(round(patch.get_height(),2)), (patch.get_x() * 1.005, patch.get_height() * 1.020))

ax.legend()

fig.tight_layout()

plt.show()