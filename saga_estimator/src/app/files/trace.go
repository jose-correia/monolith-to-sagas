package files

import (
	cb "app/codebase/values"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
)

type TraceHandler interface {
	GetCodebaseFoldersNames() []string
	GetTraceFileNames(codebaseName string) []string
	ReadTrace(codebase string, filename string) (*Trace, error)
	DecodeTrace(codebase *cb.Codebase, trace *Trace) *cb.Feature
}

type DefaultHandler struct {
	logger          log.Logger
	traceFolderPath string
}

func New(logger log.Logger, traceFolderPath string) TraceHandler {
	return &DefaultHandler{
		logger:          log.With(logger, "module", "traceHandler"),
		traceFolderPath: traceFolderPath,
	}
}

func (svc *DefaultHandler) GetCodebaseFoldersNames() (codebaseFoldersNames []string) {
	var idx int
	filepath.Walk(svc.traceFolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			svc.logger.Log(err.Error())
			return err
		}

		if info.IsDir() && idx != 0 {
			codebaseFoldersNames = append(codebaseFoldersNames, info.Name())
		}
		idx++
		return nil
	})
	return
}

func (svc *DefaultHandler) GetTraceFileNames(codebaseName string) (traceFilesNames []string) {
	filepath.Walk(svc.traceFolderPath+"/"+codebaseName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			svc.logger.Log(err.Error())
			return err
		}

		if !info.IsDir() {
			traceFilesNames = append(traceFilesNames, info.Name())
		}
		return nil
	})
	return
}

func (svc *DefaultHandler) ReadTrace(codebase string, filename string) (*Trace, error) {
	path := svc.traceFolderPath + "/" + codebase + "/" + filename
	jsonFile, err := os.Open(path)
	if err != nil {
		svc.logger.Log(err)
		return nil, err
	}

	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		svc.logger.Log(err)
		return nil, err
	}

	var trace Trace
	err = json.Unmarshal(byteValue, &trace)
	if err != nil {
		svc.logger.Log(err)
		return nil, err
	}

	return &trace, nil
}

func (svc *DefaultHandler) DecodeTrace(codebase *cb.Codebase, trace *Trace) (feature *cb.Feature) {
	feature = &cb.Feature{
		Name:           trace.Name,
		Codebase:       codebase,
		Type:           "QUERY",
		Complexity:     trace.Complexity,
		Clusters:       []*cb.Cluster{},
		EntityAccesses: map[*cb.Entity][]*cb.Access{},
		Redesigns:      []*cb.Redesign{},
	}
	feature.Redesigns = append(
		feature.Redesigns,
		svc.decodeRedesign(feature, trace.getMonolithRedesign()),
	)
	return
}

func (svc *DefaultHandler) decodeRedesign(feature *cb.Feature, traceRedesign *FunctionalityRedesign) (redesign *cb.Redesign) {
	redesign = &cb.Redesign{
		Name:                    traceRedesign.Name,
		Feature:                 feature,
		FirstInvocation:         &cb.Invocation{},
		InvocationsByEntity:     map[*cb.Entity][]*cb.Invocation{},
		InvocationsByCluster:    map[*cb.Cluster][]*cb.Invocation{},
		SystemComplexity:        traceRedesign.SystemComplexity,
		FunctionalityComplexity: traceRedesign.FunctionalityComplexity,
	}

	// for each invocation in the trace we add its operations and add it to a linked list
	var prevInvocation *cb.Invocation
	for idx, clusterInvocation := range traceRedesign.Redesign {
		if clusterInvocation.ID == "-1" {
			continue
		}

		invocation := redesign.AddInvocation(
			clusterInvocation.Cluster,
			clusterInvocation.Type,
			clusterInvocation.getAcessesEntities(),
		)

		if idx == 1 {
			redesign.FirstInvocation = invocation
		} else {
			prevInvocation.NextInvocation = invocation
		}

		prevInvocation = invocation
	}

	return
}
