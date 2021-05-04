import matplotlib.pyplot as plt
import pandas as pd
import numpy as np

plt.style.use('ggplot')
operations_data = [2,5,1,1]
ids = [0,1,2,3]

series = pd.Series(operations_data)


# Plot the figure.
plt.figure()
ax = series.plot(kind='bar', color="cornflowerblue")

ax.set_xlabel('Number of clusters with more than 1 invocation', fontsize=13)
ax.set_ylabel('Number of functionalities', fontsize=13)
ax.set_xticklabels(ids, rotation=0)

plt.tight_layout()
plt.show()