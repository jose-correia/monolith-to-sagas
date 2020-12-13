import json
import sys

from gspread_formatting import CellFormat, Color, TextFormat, format_cell_range
from google_sheets import (
    COLUMNS,
    open_sheet,
    get_or_create_worksheet,
    add_to_cell,
)

SHEET_NAME = "Trace extractions"
FEATURE_NAME_CELL = "A1"
LINE_TO_INSERT_FEATURE_METRICS_TITLES = 3

entity_operation_id = 0


class EntityMetrics():

    def __init__(self):
        self.total_read_operations = 0                          # Total number of read operations
        self.total_write_operations = 0                         # Total number of write operations
        self.total_operations = 0                               # Total number of times that an entity is accessed
        self.average_operations_per_cluster_invocation = 0.0    # Average number of times that an entity is accessed per cluster invocation
        self.total_pivot_entity_operations = 0                  # Total number of other entities called between this entity operations
        self.average_pivot_entity_operations = 0.0              # Average number of other entities accessed between entity operations


class Entity():
    
    def __init__(self, cluster_name, name):
        self.cluster_name = cluster_name
        self.name = name
        self.metrics = EntityMetrics()
        self.operation_ids = list()
        self.invocation_ids = list()


class ClusterMetrics():
    
    def __init__(self):
        self.total_lock_invocations = 0                 # Total number of invocations that have one or more write operations
        self.total_read_invocations = 0                 # Total number of invocations with only read operations, and no semantic lock
        self.total_invocations = 0                      # Total number of times that a cluster appears in the trace
        self.total_read_operations = 0                  # Total number of read operations in any given entity
        self.total_write_operations = 0                 # Total number of write operations in any given entity
        self.total_operations = 0                       # Total number of opperations in the entities of the cluster
        self.average_operations_per_invocation = 0.0    # Average number of total entity operations per cluster invocations
        self.average_reads_per_invocation = 0.0         # Average number of read entity operations per cluster invocations
        self.average_writes_per_invocation = 0.0        # Average number of write entity operations per cluster invocations
        self.total_pivot_invocations = 0                # Total number of external clusters accessed between all cluster invocations
        self.average_pivot_invocations = 0.0            # Average number of other clusters called between each cluster invocation


class Cluster():

    def __init__(self, name):
        self.name = name
        self.metrics = ClusterMetrics()
        self.entities = dict()
        self.invocation_ids = list()

    def get_or_create_entity(self, entity_name: str) -> Entity:
        if not self.entities.get(entity_name):
            self.entities[entity_name] = Entity(cluster_name=self.name, name=entity_name)
        
        return self.entities[entity_name]


class FeatureMetrics():

    def __init__(self):
        self.complexity = 0.0
        self.total_clusters = 0
        self.total_cluster_invocations = 0


class Feature():

    def __init__(self, name):
        self.name = name
        self.metrics = FeatureMetrics()
        self.clusters = dict()

    def get_or_create_cluster(self, cluster_name: str) -> Cluster:
        if not self.clusters.get(cluster_name):
            self.clusters[cluster_name] = Cluster(name=cluster_name)
        
        return self.clusters[cluster_name]


def calculate_invocation_metrics(cluster: Cluster, trace_invocation): 
    global entity_operation_id
    has_semantick_lock = False

    entity_operations = json.loads(trace_invocation["accessedEntities"])
    for entity_operation in entity_operations:
        entity_name = entity_operation[0]
        operation = entity_operation[1]

        entity = cluster.get_or_create_entity(entity_name)
        entity.invocation_ids.append(cluster.invocation_ids[len(cluster.invocation_ids) - 1])

        entity.operation_ids.append(entity_operation_id)
        entity_operation_id += 1

        if operation == "R":
            cluster.metrics.total_read_operations += 1
            entity.metrics.total_read_operations += 1

        elif operation == "W":
            cluster.metrics.total_write_operations += 1
            entity.metrics.total_write_operations += 1
            has_semantick_lock = True

    if has_semantick_lock:
        cluster.metrics.total_lock_invocations += 1
    else:
        cluster.metrics.total_read_invocations += 1


def calculate_final_cluster_metrics(feature: Feature, cluster: Cluster):
    cluster.metrics.total_invocations = cluster.metrics.total_lock_invocations + cluster.metrics.total_read_invocations
    cluster.metrics.total_operations = cluster.metrics.total_read_operations + cluster.metrics.total_write_operations

    cluster.metrics.average_operations_per_invocation = float(cluster.metrics.total_operations / cluster.metrics.total_invocations)
    
    cluster.metrics.total_pivot_invocations = feature.metrics.total_cluster_invocations - cluster.metrics.total_invocations - cluster.invocation_ids[0] - (feature.metrics.total_cluster_invocations - cluster.invocation_ids[len(cluster.invocation_ids) - 1] - 1) # WE HAVE TO REMOVE THE NUMBER OF INVOCATIONS BEFORE THE FIRST ONE AND AFTER THE LAST ONE OF THIS CLUSTER

    cluster.metrics.average_reads_per_invocation = float(cluster.metrics.total_read_operations / cluster.metrics.total_invocations)
    cluster.metrics.average_writes_per_invocation = float(cluster.metrics.total_write_operations / cluster.metrics.total_invocations)

    if cluster.metrics.total_invocations > 1:
        cluster.metrics.average_pivot_invocations = float(cluster.metrics.total_pivot_invocations / (cluster.metrics.total_invocations - 1))


def calculate_final_entity_metrics(cluster: Cluster, entity: Entity):
    entity.metrics.total_operations = entity.metrics.total_read_operations + entity.metrics.total_write_operations

    entity.metrics.average_operations_per_cluster_invocation = float(entity.metrics.total_operations / cluster.metrics.total_invocations)

    # entity.metrics.average_pivot_entity_operations


def calculate_final_metrics(feature: Feature):
    for _, cluster in feature.clusters.items():
        calculate_final_cluster_metrics(feature, cluster)

        for _, entity in cluster.entities.items():
            calculate_final_entity_metrics(cluster, entity)


def report_to_sheet(feature: Feature):
    sheet = open_sheet(SHEET_NAME)
    worksheet = get_or_create_worksheet(sheet, f"{feature.name}_metrics")

    # add feature name
    add_to_cell(worksheet, FEATURE_NAME_CELL, feature.name)
    format_cell_range(
        worksheet,
        FEATURE_NAME_CELL,
        CellFormat(
            textFormat=TextFormat(bold=True, fontSize=14),
        ),
    )
    
    # add feature metrics
    line = LINE_TO_INSERT_FEATURE_METRICS_TITLES
    cell = f"{COLUMNS[0]}{line}"
    data = [
        ["Complexity", "Total Clusters", "Total Cluster Invocations"],
        [
            feature.metrics.complexity,
            feature.metrics.total_clusters,
            feature.metrics.total_cluster_invocations
        ],
    ]
    add_to_cell(worksheet, cell, data)
    format_cell_range(
        worksheet,
        f"A{line}:CZ{line}",
        CellFormat(
            textFormat=TextFormat(bold=True),
            horizontalAlignment="CENTER",
        ),
    )

    # add cluster metrics
    line += 3
    cell = f"{COLUMNS[0]}{line}"
    data = [
        ["Cluster metrics"],
        ["Name", "TI", "TLI", "TRI", "TO", "TRO", "TWO", "AOPI", "ARPI", "AWPI", "TPI", "API"],
    ]
    add_to_cell(worksheet, cell, data)
    format_cell_range(
        worksheet,
        f"A{line+1}:CZ{line+1}",
        CellFormat(
            textFormat=TextFormat(bold=True),
            horizontalAlignment="CENTER",
        ),
    )
    
    line += 2
    for _, cluster in feature.clusters.items():
        cell = f"{COLUMNS[0]}{line}"
        data = [
            [
                cluster.name,
                cluster.metrics.total_invocations,
                cluster.metrics.total_lock_invocations,
                cluster.metrics.total_read_invocations,
                cluster.metrics.total_operations,
                cluster.metrics.total_read_operations,
                cluster.metrics.total_write_operations, 
                cluster.metrics.average_operations_per_invocation,
                cluster.metrics.average_reads_per_invocation,
                cluster.metrics.average_writes_per_invocation,
                cluster.metrics.total_pivot_invocations,
                cluster.metrics.average_pivot_invocations,
            ],
            [],
        ]
        add_to_cell(worksheet, cell, data)
        line += 1
    
    # add entity metrics
    line += 2
    cell = f"{COLUMNS[0]}{line}"
    data = [
        ["Entity metrics"],
        ["Name", "TO", "TRO", "TWO", "AOPCI", "TPEO", "APEO"],
    ]
    add_to_cell(worksheet, cell, data)
    format_cell_range(
        worksheet,
        f"A{line+1}:CZ{line+1}",
        CellFormat(
            textFormat=TextFormat(bold=True),
            horizontalAlignment="CENTER",
        ),
    )
    
    line += 2
    for _, cluster in feature.clusters.items():
        for _, entity in cluster.entities.items():
            cell = f"{COLUMNS[0]}{line}"
            data = [
                [
                    entity.name,
                    entity.metrics.total_operations,
                    entity.metrics.total_read_operations,
                    entity.metrics.total_write_operations,
                    entity.metrics.average_operations_per_cluster_invocation,
                    entity.metrics.total_pivot_entity_operations,
                    entity.metrics.average_pivot_entity_operations, 
                ],
                [],
            ]
            add_to_cell(worksheet, cell, data)
            line += 1

def main():
    if len(sys.argv) != 2:
        print("Trace file location missing!")
        exit()

    with open(sys.argv[1]) as json_file:
        data = json.load(json_file)

    feature = Feature(name=data["name"])
    feature.metrics.complexity = float(data["complexity"])

    for trace_invocation in data["functionalityRedesigns"][0]["redesign"]:
        invocation_id = int(trace_invocation["id"])
        if invocation_id == -1:
            continue

        feature.metrics.total_cluster_invocations += 1
        if not feature.clusters.get(trace_invocation["cluster"]):
            feature.metrics.total_clusters += 1

        cluster = feature.get_or_create_cluster(trace_invocation["cluster"])
        cluster.invocation_ids.append(invocation_id)

        calculate_invocation_metrics(cluster, trace_invocation)

    calculate_final_metrics(feature)
    report_to_sheet(feature)


if __name__ == "__main__":
    main()
