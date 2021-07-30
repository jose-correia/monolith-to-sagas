from typing import List
from ..datasets.helpers import read_dataset, mean, stdev, correlation
from ..plots.metrics.plot_reduction_per_merge_percentage import plot_complexity_reduction_per_merge_percentage
import numpy as np


class RowData:
    initial_frc_row="Initial Functionality Complexity"
    final_frc_row="Final Functionality Complexity"
    frc_reduction_row="Functionality Complexity Reduction"
    initial_sac_row="Initial System Complexity"
    final_sac_row="Final System Complexity"
    sac_reduction_row="System Complexity Reduction"
    initial_invocations_row="Initial Invocations Count W/ Empties"
    final_invocations_row="Final Invocations Count"
    merges_row="Total Invocation Merges"
    functionality_name="Feature"
    sweeps_row="Total Trace Sweeps w/ Merges"
    initial_accesses_row="Initial Accesses count"
    final_accesses_row="Final Accesses count"
    clip="CLIP"
    crip="CRIP"
    crop="CROP"
    cwop="CWOP"
    cip="CIP"
    cop="COP"
    cddip="CDDIP"
    sccp="SCCP"
    fccp="FCCP"



class Extraction:

    def __init__(
        self,
        complexities_csv: str,
        complexities_csv_rows: List[str],
        training_csv: str,
        training_csv_rows: List[str],
        codebase: str = None,
        functionality: str = None,
        features: List[str] = [],
        use_system_complexity: bool = False,
    ):
        self.complexities_csv = complexities_csv
        self.complexities_csv_rows = complexities_csv_rows
        self.training_csv = training_csv
        self.training_csv_rows = training_csv_rows
        self.codebase = codebase
        self.functionality = functionality
        self.features = features

        self.complexities_dataset = None
        self.training_dataset = None
        
        self.rows = RowData()

        self._read_datasets()
    
    def _read_datasets(self):
        self.complexities_dataset = read_dataset(self.complexities_csv, self.complexities_csv_rows, self.functionality, self.codebase)
        self.training_dataset = read_dataset(self.training_csv, self.training_csv_rows, self.functionality, self.codebase)

    def do_stuff(self, best_clusters, other_clusters):
        print("Functionalities: " + str(best_clusters.total_features_including_zero_complexity_reduction))
        
        print("\nInitial FRC:")
        print("Average " + str(mean(best_clusters.initial_frc_complexities)))
        print("Stdev " + str(stdev(best_clusters.initial_frc_complexities)))

        print("\nFinal FRC:")
        print("Average " + str(mean(best_clusters.final_frc_complexities)))
        print("Stdev " + str(stdev(best_clusters.final_frc_complexities)))

        print("\nFRC Reduction %:")
        print("Average " + str(mean(best_clusters.frc_reductions)))
        print("Stdev " + str(stdev(best_clusters.frc_reductions)))


        print("\nInitial SAC:")
        print("Average " + str(mean(best_clusters.initial_sac_complexities)))
        print("Stdev " + str(stdev(best_clusters.initial_sac_complexities)))

        print("\nFinal SAC:")
        print("Average " + str(mean(best_clusters.final_sac_complexities)))
        print("Stdev " + str(stdev(best_clusters.final_sac_complexities)))

        print("\nSAC Reduction %:")
        print("Average " + str(mean(best_clusters.sac_reductions)))
        print("Stdev " + str(stdev(best_clusters.sac_reductions)))


        print("\nInitial invocations count:")
        print("Average " + str(mean(best_clusters.initial_invocations_count)))
        print("Stdev " + str(stdev(best_clusters.initial_invocations_count)))

        print("\nFinal invocations count:")
        print("Average " + str(mean(best_clusters.final_invocations_count)))
        print("Stdev " + str(stdev(best_clusters.final_invocations_count)))

        print("\nMerges count:")
        print("Average " + str(mean(best_clusters.merges)))
        print("Stdev " + str(stdev(best_clusters.merges)))


        print("\nInitial accesses count:")
        print("Average " + str(mean(best_clusters.initial_accesses_count)))
        print("Stdev " + str(stdev(best_clusters.initial_accesses_count)))

        print("\nFinal accesses count:")
        print("Average " + str(mean(best_clusters.final_accesses_count)))
        print("Stdev " + str(stdev(best_clusters.final_accesses_count)))

        print("\nAccess reduction %:")
        print("Average " + str(mean(best_clusters.access_reduction_percentage)))
        print("Stdev " + str(stdev(best_clusters.access_reduction_percentage)))


        print("\nMerge %:")
        print("Average " + str(mean(best_clusters.merge_percentages)))
        print("Stdev " + str(stdev(best_clusters.merge_percentages)))

        print("\nSweeps:")
        print("Average " + str(mean(best_clusters.sweeps)))
        print("Stdev " + str(stdev(best_clusters.sweeps)))

        print("\nCRIP Correlation with FRC reduction %")
        print(
            correlation(
                best_clusters.crip,
                best_clusters.frc_reductions,
            )
        )

        print("\nCIP Correlation with FRC reduction %")
        print(
            correlation(
                best_clusters.cip,
                best_clusters.frc_reductions,
            )
        )

        print("\nCROP Correlation with FRC reduction %")
        print(
            correlation(
                best_clusters.crop,
                best_clusters.frc_reductions,
            )
        )

        print("\nCWOP Correlation with FRC reduction %")
        print(
            correlation(
                best_clusters.cwop,
                best_clusters.frc_reductions,
            )
        )

        print("\CDDIP Correlation with FRC reduction %")
        print(
            correlation(
                best_clusters.cddip,
                best_clusters.frc_reductions,
            )
        )

        print("\nCOP Correlation with FRC reduction %")
        print(
            correlation(
                best_clusters.cop,
                best_clusters.frc_reductions,
            )
        )


        print("\nSCCP Correlation with FRC reduction %")
        print(
            correlation(
                best_clusters.fccp,
                best_clusters.frc_reductions,
            )
        )

        print("\FCCP Correlation with FRC reduction %")
        print(
            correlation(
                best_clusters.fccp,
                best_clusters.frc_reductions,
            )
        )

        print("\nCRIP Correlation with SAC reduction %")
        print(
            correlation(
                best_clusters.crip,
                best_clusters.sac_reductions,
            )
        )

        print("\nCOP Correlation with SAC reduction %")
        print(
            correlation(
                best_clusters.cop,
                best_clusters.sac_reductions,
            )
        )

        print("\nSCCP Correlation with SAC reduction %")
        print(
            correlation(
                best_clusters.sccp,
                best_clusters.sac_reductions,
            )
        )

        print("\FCCP Correlation with SAC reduction %")
        print(
            correlation(
                best_clusters.fccp,
                best_clusters.sac_reductions,
            )
        )

        print("\nCIP Correlation with SAC reduction %")
        print(
            correlation(
                best_clusters.cip,
                best_clusters.sac_reductions,
            )
        )

        print("\nCROP Correlation with SAC reduction %")
        print(
            correlation(
                best_clusters.crop,
                best_clusters.sac_reductions,
            )
        )

        print("\nCWOP Correlation with SAC reduction %")
        print(
            correlation(
                best_clusters.cwop,
                best_clusters.sac_reductions,
            )
        )

        print("\CDDIP Correlation with SAC reduction %")
        print(
            correlation(
                best_clusters.cddip,
                best_clusters.sac_reductions,
            )
        )

        print("\nCRIP Correlation with SAC+FRC reduction %")
        print(
            correlation(
                best_clusters.crip,
                np.add(best_clusters.sac_reductions, best_clusters.frc_reductions),
            )
        )

        print("\nCOP Correlation with SAC+FRC reduction %")
        print(
            correlation(
                best_clusters.cop,
                np.add(best_clusters.sac_reductions, best_clusters.frc_reductions),
            )
        )

        print("\nSCCP Correlation with SAC+FRC reduction %")
        print(
            correlation(
                best_clusters.sccp,
                np.add(best_clusters.sac_reductions, best_clusters.frc_reductions),
            )
        )

        print("\FCCP Correlation with SAC+FRC reduction %")
        print(
            correlation(
                best_clusters.fccp,
                np.add(best_clusters.sac_reductions, best_clusters.frc_reductions),
            )
        )

        print("\nCIP Correlation with SAC+FRC reduction %")
        print(
            correlation(
                best_clusters.cip,
                np.add(best_clusters.sac_reductions, best_clusters.frc_reductions),
            )
        )

        print("\nCROP Correlation with SAC+FRC reduction %")
        print(
            correlation(
                best_clusters.crop,
                np.add(best_clusters.sac_reductions, best_clusters.frc_reductions),
            )
        )

        print("\nCWOP Correlation with SAC+FRC reduction %")
        print(
            correlation(
                best_clusters.cwop,
                np.add(best_clusters.sac_reductions, best_clusters.frc_reductions),
            )
        )

        print("\CDDIP Correlation with SAC+FRC reduction %")
        print(
            correlation(
                best_clusters.cddip,
                np.add(best_clusters.sac_reductions, best_clusters.frc_reductions),
            )
        )

        plot_complexity_reduction_per_merge_percentage(best_clusters)
