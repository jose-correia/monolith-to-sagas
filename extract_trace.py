import gspread
import sys
import json

from gspread_formatting import CellFormat, Color, TextFormat, format_cell_range
from google_sheets import (
    COLUMNS,
    open_sheet,
    get_or_create_worksheet,
    add_to_cell,
)

SHEET_NAME = "Trace extractions"

FEATUE_NAME_CELL = "A1"
ORIGINAL_FLAG_CELL = "A5"
REFACTOR_FLAG_CELL = "A10"
LINE_TO_INSERT_ORIGINAL_TRACE = 6
LINE_TO_INSERT_REFACTOR_TRACE = 11

# Refactor
# worksheet.update(f'B5', [['Text'], ['R,R,W'], ['User'], ['R']])


def add_trace_invocation_to_sheet(worksheet, column, line, invocation):
    cluster_name = invocation["cluster"]

    operations = ""
    if invocation["accessedEntities"] != "":
        transactions = json.loads(invocation["accessedEntities"])

        for index, transaction in enumerate(transactions):
            # transaction_name = transaction[0]
            transaction_type = transaction[1]

            operations += transaction_type if index == 0 else f",{transaction_type}"

    cell = f"{column}{line}"
    data = [[cluster_name], [operations]]
    add_to_cell(worksheet, cell, data)


def add_original_trace_to_sheet(worksheet, data):
    add_to_cell(worksheet, ORIGINAL_FLAG_CELL, 'Original')
    format_cell_range(
        worksheet,
        ORIGINAL_FLAG_CELL,
        CellFormat(
            textFormat=TextFormat(bold=True, fontSize=12),
        ),
    )

    sheet_column_index = 0
    for invocation in data["functionalityRedesigns"][0]["redesign"]:
        column = COLUMNS[sheet_column_index]
        add_trace_invocation_to_sheet(worksheet, column, LINE_TO_INSERT_ORIGINAL_TRACE, invocation)
        sheet_column_index += 1
    
    format_cell_range(
        worksheet,
        f"A{LINE_TO_INSERT_ORIGINAL_TRACE}:CZ{LINE_TO_INSERT_ORIGINAL_TRACE}",
        CellFormat(
            textFormat=TextFormat(bold=True),
            horizontalAlignment="CENTER",
        ),
    )

    format_cell_range(
        worksheet,
        f"A{LINE_TO_INSERT_ORIGINAL_TRACE}:CZ{LINE_TO_INSERT_ORIGINAL_TRACE+1}",
        CellFormat(
            horizontalAlignment="CENTER",
        ),
    )


def add_refactored_trace_to_sheet(worksheet, data):
    add_to_cell(worksheet, REFACTOR_FLAG_CELL, 'Refactor')
    format_cell_range(
        worksheet,
        REFACTOR_FLAG_CELL,
        CellFormat(
            textFormat=TextFormat(bold=True, fontSize=12),
        ),
    )

    # the redesign is not sorted so we sort it
    invocations_list = data["functionalityRedesigns"][1]["redesign"]
    trace = sorted(invocations_list, key=lambda k: k['id'])

    sheet_column_index = 0

    first = trace[0]
    column = COLUMNS[sheet_column_index]
    add_trace_invocation_to_sheet(worksheet, column, LINE_TO_INSERT_REFACTOR_TRACE, first)

    remote_invocations = first["remoteInvocations"]

    for invocation in remote_invocations:
        column = COLUMNS[sheet_column_index]
        line = LINE_TO_INSERT_REFACTOR_TRACE
        
        add_trace_invocation_to_sheet(worksheet, column, LINE_TO_INSERT_REFACTOR_TRACE, first)


def main():
    if len(sys.argv) != 2:
        print("Trace file location missing!")
        exit()

    data_file = sys.argv[1]

    with open(data_file) as json_file:
        data = json.load(json_file)

        feature_name = data["name"]
        # complexity = data["complexity"]

        # Open google sheet
        sheet = open_sheet(SHEET_NAME)
        worksheet = get_or_create_worksheet(sheet, f"{feature_name}_trace")

        # add feature name
        add_to_cell(worksheet, FEATUE_NAME_CELL, feature_name)
        format_cell_range(
            worksheet,
            FEATUE_NAME_CELL,
            CellFormat(
                textFormat=TextFormat(bold=True, fontSize=14),
            ),
        )

        # add original trace
        add_original_trace_to_sheet(worksheet, data)


if __name__ == "__main__":
    main()
