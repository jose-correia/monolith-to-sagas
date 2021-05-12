import matplotlib.pyplot as plt
import pandas as pd
import numpy as np

plt.style.use('ggplot')
operations_data = [8,7,96,73,2,54,13,37,30]
ids = [1,2,3,4,5,6,7,8,9]

series = pd.Series(operations_data)


# Plot the figure.
plt.figure()
ax = series.plot(kind='bar', color="cornflowerblue")

ax.set_xlabel('Functionality ID', fontsize=13)
ax.set_ylabel('Merge operations executed', fontsize=13)
ax.set_xticklabels(ids, rotation=0)
ax.set_yscale('log')

rects = ax.patches


for rect, label in zip(rects, operations_data):
    height = rect.get_height()
    ax.text(rect.get_x() + rect.get_width() / 2, height + 0.5, label,
            ha='center', va='bottom')

plt.tight_layout()
plt.show()