package main

import (
	"app/codebase"
	"app/codebase/values"
	"app/common/log"
	"app/files"
	"app/metrics"
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
	for _, folderName := range codebaseFoldersNames {
		codebase, err := codebaseHandler.GenerateCodebase(folderName)
		if err != nil {
			logger.Log(err.Error())
			continue
		}

		for _, feature := range codebase.Features {
			bestRedesign, err = codebaseHandler.EstimateBestFeatureRedesign(feature)
			if err != nil {
				logger.Log(err.Error())
				continue
			}
		}
	}
}
