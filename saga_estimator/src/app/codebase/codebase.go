package codebase

import (
	"app/codebase/values"
	"app/files"

	"github.com/go-kit/kit/log"
)

type CodebaseHandler interface {
	GenerateCodebase(codebaseName string) (codebase *values.Codebase, err error)
}

type DefaultHandler struct {
	logger       log.Logger
	traceHandler files.TraceHandler
}

func New(logger log.Logger, traceHandler files.TraceHandler) CodebaseHandler {
	return &DefaultHandler{
		logger:       log.With(logger, "module", "codebaseHandler"),
		traceHandler: traceHandler,
	}
}

func (svc *DefaultHandler) GenerateCodebase(codebaseName string) (codebase *values.Codebase, err error) {
	codebase = &values.Codebase{
		Name:     codebaseName,
		Clusters: []*values.Cluster{},
		Features: []*values.Feature{},
	}

	// find file names
	traceFileNames := svc.traceHandler.GetTraceFileNames(codebaseName)

	// read each file and decode it
	for _, traceFileName := range traceFileNames {
		trace, err := svc.traceHandler.ReadTrace(codebase.Name, traceFileName)
		if err != nil {
			svc.logger.Log("Failed to decode trace %s", traceFileName)
			continue
		}

		codebase.Features = append(codebase.Features, svc.traceHandler.DecodeTrace(codebase, trace))
	}

	return codebase, nil
}
