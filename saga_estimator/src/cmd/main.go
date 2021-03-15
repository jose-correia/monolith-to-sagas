package main

import (
	"app/common/log"
	"app/files"
	"app/metrics"
	"app/redesign"
	"app/training"
	"fmt"
	"time"
)

const (
	useExpertDecompositions = true
	// use if the objective of the dataset is to train the ML model
	generateTrainingDatasetFormat = true
)

func main() {
	logger := log.NewLogger()
	filesHandler := files.New(logger)
	metricsHandler := metrics.New(logger)
	trainingHandler := training.New(logger)
	redesignHandler := redesign.New(logger, metricsHandler, trainingHandler)

	codebaseNames := []string{
		"ldod-static",
	}

	csvData := getCsvDataFormat(generateTrainingDatasetFormat)

	var codebaseData [][]string
	for _, folderName := range codebaseNames {
		codebase, err := filesHandler.ReadCodebase(folderName)
		if err != nil {
			logger.Log("Failed to decode codebase %s | %s", folderName, err.Error())
			continue
		}

		idToEntityMap, err := filesHandler.ReadIDToEntityFile(folderName)
		if err != nil {
			logger.Log("Failed to decode id_to_entity map %s | %s", folderName, err.Error())
			continue
		}

		codebaseData = redesignHandler.EstimateCodebaseOrchestrators(
			codebase,
			idToEntityMap,
			useExpertDecompositions,
			generateTrainingDatasetFormat,
		)

		for _, row := range codebaseData {
			csvData = append(csvData, row)
		}
	}

	t := time.Now()
	outputFileName := fmt.Sprintf("%s.csv", t.Format("2006-01-02 15:04:05"))
	filesHandler.GenerateCSV(outputFileName, csvData)
}

func getCsvDataFormat(trainingDataset bool) [][]string {
	if trainingDataset {
		return [][]string{
			{
				"Codebase",
				"Feature",
				"Type",
				"Cluster",
				"Entities",
				"CLIP",
				"CRIP",
				"CROP",
				"CWOP",
				"CIP",
				"COP",
				"CPIF",
				"CIOF",
				"Orchestrator",
			},
		}
	}

	return [][]string{
		{
			"Codebase",
			"Feature",
			"Orchestrator",
			"Entities",
			"Type",
			"Initial System Complexity",
			"Final System Complexity",
			"System Complexity Reduction",
			"Initial Inconsistency Complexity",
			"Final Inconsistency Complexity",
			"Inconsistency Complexity Reduction",
			"Initial Functionality Complexity",
			"Final Functionality Complexity",
			"Functionality Complexity Reduction",
		},
	}
}
