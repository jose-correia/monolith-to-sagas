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
        # counts
        self.read_operations = 0                          # Total number of read operations
        self.write_operations = 0                         # Total number of write operations
        self.operations = 0                               # Total number of times that an entity is accessed
        self.pivot_operations = 0                  # Total number of other entities called between this entity operations
        # averages
        self.average_operations = 0.0    # Average number of times that an entity is accessed per cluster invocation
        self.average_pivot_operations = 0.0              # Average number of other entities accessed between entity operations


class Entity():
    
    def __init__(self, cluster_name, name):
        self.cluster_name = cluster_name
        self.name = name
        self.metrics = EntityMetrics()
        self.operation_ids = list()
        self.invocation_ids = list()


class ClusterMetrics():
    
    def __init__(self):
        # counts
        self.lock_invocations = 0                       # Total number of invocations that have one or more write operations
        self.read_invocations = 0                       # Total number of invocations with only read operations, and no semantic lock
        self.invocations = 0                            # Total number of times that a cluster appears in the trace
        self.read_operations = 0                        # Total number of read operations in any given entity
        self.write_operations = 0                       # Total number of write operations in any given entity
        self.operations = 0                             # Total number of opperations in the entities of the cluster
        self.pivot_invocations = 0                      # Total number of external clusters accessed between all cluster invocations
        # averages
        self.average_invocation_operations = 0.0        # Average number of total entity operations per cluster invocations
        self.average_invocation_read_operations = 0.0   # Average number of read entity operations per cluster invocations
        self.average_invocation_write_operations = 0.0  # Average number of write entity operations per cluster invocations
        self.average_pivot_invocations = 0.0            # Average number of other clusters called between each cluster invocation
        # cluster probabilities
        self.lock_invocation_probability = 0.0
        self.read_invocation_probability = 0.0
        self.read_operation_probability = 0.0
        self.write_operation_probability = 0.0
        # feature-cluster probabilities
        self.invocation_probability = 0.0
        self.operation_probability = 0.0
        # factors
        self.pivot_invocations_factor = 0.0
        self.invocation_operations_factor = 0.0


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
        # counts
        self.clusters = 0
        self.lock_invocations = 0
        self.read_invocations = 0
        self.invocations = 0
        self.read_operations = 0
        self.write_operations = 0
        self.operations = 0
        # averages
        self.average_invocation_operations = 0.0
        self.average_invocation_read_operations = 0.0
        self.average_invocation_write_operations = 0.0
        self.average_pivot_invocations = 0.0


class Feature():

    def __init__(self, name):
        self.name = name
        self.metrics = FeatureMetrics()
        self.clusters = dict()

    def get_or_create_cluster(self, cluster_name: str) -> Cluster:
        if not self.clusters.get(cluster_name):
            self.clusters[cluster_name] = Cluster(name=cluster_name)
        
        return self.clusters[cluster_name]


def calculate_invocation_metrics(feature: Feature, cluster: Cluster, trace_invocation): 
    global entity_operation_id
    has_semantick_lock = False

    feature.metrics.invocations += 1
    cluster.metrics.invocations += 1

    entity_operations = json.loads(trace_invocation["accessedEntities"])
    for entity_operation in entity_operations:
        entity_name = entity_operation[0]
        operation = entity_operation[1]

        entity = cluster.get_or_create_entity(entity_name)

        entity.invocation_ids.append(cluster.invocation_ids[len(cluster.invocation_ids) - 1])
        entity.operation_ids.append(entity_operation_id)
        entity_operation_id += 1

        feature.metrics.operations += 1
        cluster.metrics.operations += 1
        entity.metrics.operations += 1

        if operation == "R":
            feature.metrics.read_operations += 1
            cluster.metrics.read_operations += 1
            entity.metrics.read_operations += 1

        elif operation == "W":
            feature.metrics.write_operations += 1
            cluster.metrics.write_operations += 1
            entity.metrics.write_operations += 1
            has_semantick_lock = True

    if has_semantick_lock:
        feature.metrics.lock_invocations += 1
        cluster.metrics.lock_invocations += 1
    else:
        feature.metrics.read_invocations += 1
        cluster.metrics.read_invocations += 1


def calculate_cluster_averages(feature: Feature, cluster: Cluster):
    cluster.metrics.average_invocation_operations = float(cluster.metrics.operations / cluster.metrics.invocations)
    cluster.metrics.average_invocation_read_operations = float(cluster.metrics.read_operations / cluster.metrics.invocations)
    cluster.metrics.average_invocation_write_operations = float(cluster.metrics.write_operations / cluster.metrics.invocations)

    if cluster.metrics.invocations > 1:
        cluster.metrics.average_pivot_invocations = float(cluster.metrics.pivot_invocations / (cluster.metrics.invocations - 1))


def calculate_cluster_probabilities(feature: Feature, cluster: Cluster):
    cluster.metrics.lock_invocation_probability = float(cluster.metrics.lock_invocations / cluster.metrics.invocations)
    cluster.metrics.read_invocation_probability = float(cluster.metrics.read_invocations / cluster.metrics.invocations)
    cluster.metrics.read_operation_probability = float(cluster.metrics.read_operations / cluster.metrics.operations)
    cluster.metrics.write_operation_probability = float(cluster.metrics.write_operations / cluster.metrics.operations)
    cluster.metrics.invocation_probability = float(cluster.metrics.invocations / feature.metrics.invocations)
    cluster.metrics.operation_probability = float(cluster.metrics.operations / feature.metrics.operations)


def update_feature_averages(feature: Feature, cluster: Cluster):
    feature.metrics.average_invocation_operations += float(cluster.metrics.average_invocation_operations / feature.metrics.clusters)
    feature.metrics.average_invocation_read_operations += float(cluster.metrics.average_invocation_read_operations / feature.metrics.clusters)
    feature.metrics.average_invocation_write_operations += float(cluster.metrics.average_invocation_write_operations / feature.metrics.clusters)
    feature.metrics.average_pivot_invocations += float(cluster.metrics.average_pivot_invocations / feature.metrics.clusters)


def calculate_entity_averages(cluster: Cluster, entity: Entity):
    entity.metrics.average_operations = float(entity.metrics.operations / cluster.metrics.invocations)

    # entity.metrics.average_pivot_entity_operations


def calculate_cluster_factors(feature: Feature, cluster: Cluster):
    cluster.metrics.pivot_invocations_factor = float(cluster.metrics.average_pivot_invocations / feature.metrics.average_pivot_invocations)
    cluster.metrics.invocation_operations_factor = float(cluster.metrics.average_invocation_operations / feature.metrics.average_invocation_operations)


def calculate_final_metrics(feature: Feature):
    for _, cluster in feature.clusters.items():
        cluster.metrics.pivot_invocations = feature.metrics.invocations - cluster.metrics.invocations - cluster.invocation_ids[0] - (feature.metrics.invocations - cluster.invocation_ids[len(cluster.invocation_ids) - 1] - 1) # WE HAVE TO REMOVE THE NUMBER OF INVOCATIONS BEFORE THE FIRST ONE AND AFTER THE LAST ONE OF THIS CLUSTER

        calculate_cluster_averages(feature, cluster)
        calculate_cluster_probabilities(feature, cluster)

        update_feature_averages(feature, cluster)
        
        for _, entity in cluster.entities.items():
            calculate_entity_averages(cluster, entity)

    for _, cluster in feature.clusters.items():
        calculate_cluster_factors(feature, cluster)


def report_feature_metrics(worksheet, line, feature: Feature):
    cell = f"{COLUMNS[0]}{line}"
    data = [
        ["Complexity", "TC", "TI", "TLI", "TRI", "TO", "TRO", "TWO", "AIO", "AIRO", "AIWO", "API"],
        [
            feature.metrics.complexity,
            feature.metrics.clusters,
            feature.metrics.invocations,
            feature.metrics.lock_invocations,
            feature.metrics.read_invocations,
            feature.metrics.operations,
            feature.metrics.read_operations,
            feature.metrics.write_operations,
            feature.metrics.average_invocation_operations,
            feature.metrics.average_invocation_read_operations,
            feature.metrics.average_invocation_write_operations,
            feature.metrics.average_pivot_invocations,
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


def report_cluster_metrics(worksheet, line, feature: Feature):
    cell = f"{COLUMNS[0]}{line}"
    data = [
        ["Cluster metrics"],
        ["Name", "CI", "CLI", "CRI", "CO", "CRO", "CWO", "CPI", "", "Name", "ACIO", "ACIRO", "ACIWO", "ACPI", "", "Name", "CLIP", "CRIP", "CROP", "CWOP", "", "Name", "CIP", "COP", "", "Name", "CPIF", "CIOF"],
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
                cluster.metrics.invocations,
                cluster.metrics.lock_invocations,
                cluster.metrics.read_invocations,
                cluster.metrics.operations,
                cluster.metrics.read_operations,
                cluster.metrics.write_operations, 
                cluster.metrics.pivot_invocations,
                "",
                cluster.name,
                cluster.metrics.average_invocation_operations,
                cluster.metrics.average_invocation_read_operations,
                cluster.metrics.average_invocation_write_operations,
                cluster.metrics.average_pivot_invocations,
                "",
                cluster.name,
                cluster.metrics.lock_invocation_probability,
                cluster.metrics.read_invocation_probability,
                cluster.metrics.read_operation_probability,
                cluster.metrics.write_operation_probability,
                "",
                cluster.name,
                cluster.metrics.invocation_probability,
                cluster.metrics.operation_probability,
                "",
                cluster.name,
                cluster.metrics.pivot_invocations_factor,
                cluster.metrics.invocation_operations_factor,
            ],
            [],
        ]
        add_to_cell(worksheet, cell, data)
        line += 1
    return line


def report_entity_metrics(worksheet, line, feature):
    cell = f"{COLUMNS[0]}{line}"
    data = [
        ["Entity metrics"],
        ["Cluster", "Name", "EO", "ERO", "EWO", "EPO", "", "Name", "AEO", "AEPO"],
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
                    entity.cluster_name,
                    entity.name,
                    entity.metrics.operations,
                    entity.metrics.read_operations,
                    entity.metrics.write_operations,
                    entity.metrics.pivot_operations,
                    "",
                    entity.name,
                    entity.metrics.average_operations,
                    entity.metrics.average_pivot_operations, 
                ],
                [],
            ]
            add_to_cell(worksheet, cell, data)
            line += 1
    return line


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
    report_feature_metrics(worksheet, line, feature)

    # add cluster metrics
    line += 3
    line = report_cluster_metrics(worksheet, line, feature)
    
    # add entity metrics
    line += 2
    _ = report_entity_metrics(worksheet, line, feature)
    

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

        if not feature.clusters.get(trace_invocation["cluster"]):
            feature.metrics.clusters += 1

        cluster = feature.get_or_create_cluster(trace_invocation["cluster"])
        cluster.invocation_ids.append(invocation_id)

        calculate_invocation_metrics(feature, cluster, trace_invocation)

    calculate_final_metrics(feature)
    report_to_sheet(feature)


if __name__ == "__main__":
    main()
