package main

import (
	"app/common/log"
	"app/files"
	"app/metrics"
	"app/redesign"
)

const (
	useExpertDecompositions = true
)

func main() {
	logger := log.NewLogger()
	filesHandler := files.New(logger)
	metricsHandler := metrics.New(logger)
	redesignHandler := redesign.New(logger, metricsHandler)

	codebaseNames := []string{
		"ldod-static",
	}

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

		redesignHandler.EstimateCodebaseOrchestrators(codebase, idToEntityMap, useExpertDecompositions)
	}
}
