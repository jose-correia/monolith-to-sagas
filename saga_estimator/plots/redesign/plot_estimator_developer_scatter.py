import matplotlib.pyplot as plt
import numpy as np

FUNCTIONALITY_NAMES = [
    "removeTweets",
    "getTaxonomy",
    "createLinearVE",
    "signup",
    "approveParticipant",
    "associateCategory",
    "deleteTaxonomy",
    "dissociate",
    "mergeCategories",
]

FUNCTIONALITY_IDS = [1,2,3,4,5,6,7,8,9]

ESTIMATOR_FRC = {
    "initial": [
        151,
        317,
        2978,
        1550,
        193,
        1813,
        261,
        806,
        485,
    ],
    "final": [
        109,
        159,
        263,
        314,
        113,
        642,
        187,
        358,
        187,
    ],
}

DEVELOPER_FRC = {
    "initial": [
        134,
        317,
        1790,
        1490,
        190,
        1803,
        253,
        772,
        453,
    ],
    "final": [
        82,
        192,
        383,
        376,
        147,
        662,
        164,
        489,
        253
    ],
}

plt.style.use('ggplot')

fig, ax = plt.subplots(1, 1, figsize=(8, 4))

ax.scatter(ESTIMATOR_FRC["initial"], ESTIMATOR_FRC["final"], s=35, color="red", label="estimator")

for name, y, x in zip(FUNCTIONALITY_IDS, ESTIMATOR_FRC["final"], ESTIMATOR_FRC["initial"]):
    plt.annotate(name, xy=(x,y), xytext=(0,np.sqrt(y)/2.+5), textcoords="offset points", ha="center", va="top", fontsize=8)


ax.scatter(DEVELOPER_FRC["initial"], DEVELOPER_FRC["final"], s=35, color="blue", label="developer")

for name, y, x in zip(FUNCTIONALITY_IDS, DEVELOPER_FRC["final"], DEVELOPER_FRC["initial"]):
    plt.annotate(name, xy=(x,y), xytext=(0,np.sqrt(y)/2.+5), textcoords="offset points", ha="center", va="top", fontsize=8)

ax.set_xlabel("Initial FRC", fontsize=13)
ax.set_ylabel("Final FRC", fontsize=13)
ax.set_xlim([0, 3200])
ax.set_ylim([50, 750])

ax.legend()
ax.set_axisbelow(True)
ax.grid(True)

fig.tight_layout()
plt.show()
