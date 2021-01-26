package main

import (
	"app/codebase"
	"app/common/log"
	"app/files"
)

var (
	traceFolderPath = "/Users/josecorreia/Desktop/tese/automation/traces"
)

func main() {
	logger := log.NewLogger()

	filesHandler := files.New(logger, traceFolderPath)

	codebaseHandler := codebase.New(logger, filesHandler)

	// find file names
	codebaseFoldersNames := filesHandler.GetCodebaseFoldersNames()

	for _, folderName := range codebaseFoldersNames {
		_, err := codebaseHandler.GenerateCodebase(folderName)
		if err != nil {
			logger.Log(err.Error())
			continue
		}
	}
}
