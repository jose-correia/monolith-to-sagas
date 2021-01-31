package main

import (
	"app/codebase"
	"app/codebase/values"
	"app/common/log"
	"app/files"
	"app/metrics"
	"fmt"
)

var (
	traceFolderPath = "/Users/josecorreia/Desktop/tese/automation/traces"
)

func main() {
	logger := log.NewLogger()

	filesHandler := files.New(logger, traceFolderPath)
	metricsHandler := metrics.New(logger)
	codebaseHandler := codebase.New(logger, filesHandler, metricsHandler)

	// find file names
	codebaseFoldersNames := filesHandler.GetCodebaseFoldersNames()

	var bestRedesign *values.Redesign
	var bestCodebaseComplexity float32
	for _, folderName := range codebaseFoldersNames {
		codebase, err := codebaseHandler.GenerateCodebase(folderName)
		if err != nil {
			logger.Log(err.Error())
			continue
		}

		for _, feature := range codebase.Features {
			bestRedesign, bestCodebaseComplexity, err = codebaseHandler.EstimateBestFeatureRedesign(feature)
			if err != nil {
				logger.Log(err.Error())
				fmt.Printf("Feature %v doesnt have more than 2 clusters! Skipping..\n\n", feature.Name)
				continue
			}

			fmt.Printf("Feature %v best redesign with orchestrator %v | Codebase complexity: %v\n\n", feature.Name, bestRedesign.FirstInvocation.Cluster.Name, bestCodebaseComplexity)
		}
	}
}
