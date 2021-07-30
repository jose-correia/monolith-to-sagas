package main

import (
	"automation/app/common/log"
	"automation/app/configuration"
	"automation/app/files"
	"automation/app/metrics"
	"automation/app/redesign"
	"automation/app/training"
	"fmt"
	"time"
)

const (
	ONLY_LAST_INVOCATION     = 0
	ALL_PREVIOUS_INVOCATIONS = -1
)

func main() {
	execution := configuration.Execution{
		Configuration: &configuration.Configuration{
			LdodOnly:                              false,
			OnlyJoaoControllers:                   true,
			GenerateComplexitiesCSV:               false,
			GenerateMetricsCSV:                    false,
			Executions:                            1,
			MinimizeSumBothComplexities:           false,
			DataDependenceThreshold:               1,
			ExcludeLowDistanceRedesigns:           false,
			AcceptableComplexityDistanceThreshold: 0,
			OnlyExportBestRedesign:                false,
			PrintTraces:                           false,
			PrintSpecificFunctionality:            "",
		},
	}
	execution.Configuration.GenerateDefaultCodebaseConfiguration()

	logger := log.NewLogger()
	filesHandler := files.New(logger)
	redesignHandler := redesign.New(
		logger,
		metrics.New(logger),
		training.New(logger),
		execution,
	)

	fmt.Printf("Estimating the best saga redesign for %v codebases...\n\n", len(execution.Configuration.Codebases))

	for i := 0; i < execution.Configuration.Executions; i++ {
		start := time.Now()

		results := execution.GenerateResults()

		for _, codebaseConfig := range execution.Configuration.Codebases {
			codebase, err := filesHandler.ReadCodebase(codebaseConfig.Name)
			if err != nil {
				logger.Log("Failed to decode codebase %s | %s", codebaseConfig.Name, err.Error())
				continue
			}

			idToEntityMap, err := filesHandler.ReadIDToEntityFile(codebaseConfig.Name)
			if err != nil {
				logger.Log("Failed to decode id_to_entity map %s | %s", codebaseConfig.Name, err.Error())
				continue
			}

			datasets := redesignHandler.EstimateCodebaseOrchestrators(
				codebase,
				idToEntityMap,
				codebaseConfig,
				&results,
			)

			if execution.Configuration.GenerateComplexitiesCSV {
				for _, row := range datasets.ComplexitiesDataset {
					results.Datasets.ComplexitiesDataset = append(results.Datasets.ComplexitiesDataset, row)
				}
			}

			if execution.Configuration.GenerateMetricsCSV {
				for _, row := range datasets.MetricsDataset {
					results.Datasets.MetricsDataset = append(results.Datasets.MetricsDataset, row)
				}
			}

			fmt.Printf("Finished estimation for codebase %v\n", codebase.Name)
		}

		results.ExecutionTime = time.Since(start)
		execution.ResultsBatches = append(execution.ResultsBatches, results)
		generateCSVFiles(execution, results, filesHandler)
	}

	performanceEvaluation(execution)

	fmt.Printf("\nDone!\n")
}

func performanceEvaluation(execution configuration.Execution) {
	fmt.Println(execution.ResultsBatches[1].FunctionalityExecutionTimes)

	fmt.Printf("\nAverage refactorization duration for everything: %s\n", execution.GetAverageExecutionTimes())

	fmt.Printf("\nAverage refactorization duration for each codebase: %s\n", configuration.GetAverageDuration(execution.ResultsBatches[1].CodebaseExecutionTimes))
	highest, lowest := configuration.GetLowestAndHighestDuration(execution.ResultsBatches[1].CodebaseExecutionTimes)
	fmt.Printf("\nLowest: %s   |    Highest: %s\n", lowest, highest)

	fmt.Printf("\nAverage refactorization time for each functionalities: %s\n", configuration.GetAverageDuration(execution.ResultsBatches[1].FunctionalityExecutionTimes))
	highest, lowest = configuration.GetLowestAndHighestDuration(execution.ResultsBatches[1].FunctionalityExecutionTimes)
	fmt.Printf("\nLowest: %s   |    Highest: %s\n", lowest, highest)
}

func generateCSVFiles(
	execution configuration.Execution, result configuration.Results, filesHandler files.FilesHandler,
) {
	identifier := "all"
	if execution.Configuration.LdodOnly {
		identifier = "ldod"
	}

	if execution.Configuration.GenerateComplexitiesCSV {
		t := time.Now()
		outputFileName := fmt.Sprintf("%s-complexities-%s.csv", identifier, t.Format("2006-01-02-15-04-05"))
		fmt.Printf("\nGenerating complexities .csv: %v\n", outputFileName)
		filesHandler.GenerateCSV(outputFileName, result.Datasets.ComplexitiesDataset)
	}

	if execution.Configuration.GenerateMetricsCSV {
		t := time.Now()
		outputFileName := fmt.Sprintf("%s-metrics-%s.csv", identifier, t.Format("2006-01-02-15-04-05"))
		fmt.Printf("\nGenerating metrics .csv: %v\n", outputFileName)
		filesHandler.GenerateCSV(outputFileName, result.Datasets.MetricsDataset)
	}
}
