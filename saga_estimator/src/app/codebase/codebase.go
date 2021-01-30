package codebase

import (
	"app/codebase/values"
	"app/files"
	"app/metrics"
	"fmt"

	"github.com/go-kit/kit/log"
)

const (
	defaultRedesignName = "OrchestratorRedesign"
)

type CodebaseHandler interface {
	GenerateCodebase(string) (*values.Codebase, error)
	EstimateBestFeatureRedesign(*values.Feature) (*values.Redesign, error)
	RedesignFeatureWithOrchestrator(*values.Redesign, *values.Cluster) *values.Redesign
	SwapRedesignOrchestrator(*values.Redesign, *values.Cluster)
}

type DefaultHandler struct {
	logger         log.Logger
	traceHandler   files.TraceHandler
	metricsHandler metrics.MetricsHandler
}

func New(logger log.Logger, traceHandler files.TraceHandler, metricsHandler metrics.MetricsHandler) CodebaseHandler {
	return &DefaultHandler{
		logger:         log.With(logger, "module", "codebaseHandler"),
		traceHandler:   traceHandler,
		metricsHandler: metricsHandler,
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

func (svc *DefaultHandler) EstimateBestFeatureRedesign(feature *values.Feature) (*values.Redesign, error) {
	if len(feature.Clusters) < 3 {
		return nil, fmt.Errorf("In order to decide the best redesign the feature must have more than 2 clusters")
	}

	initialRedesign := feature.GetMonolithRedesign()

	var newRedesign *values.Redesign
	newRedesign = svc.RedesignFeatureWithOrchestrator(initialRedesign, feature.Clusters[0])
	// calculate metrics for first cluster

	for idx := 1; idx < len(feature.Clusters); idx++ {
		svc.SwapRedesignOrchestrator(newRedesign, feature.Clusters[idx])
		// recalculate metrics
	}

	return newRedesign, nil
}

func (svc *DefaultHandler) RedesignFeatureWithOrchestrator(initialRedesign *values.Redesign, orchestrator *values.Cluster) (redesign *values.Redesign) {
	redesign = &values.Redesign{
		Name:                 defaultRedesignName,
		Feature:              initialRedesign.Feature,
		FirstInvocation:      &values.Invocation{},
		InvocationsByEntity:  map[*values.Entity][]*values.Invocation{},
		InvocationsByCluster: map[*values.Cluster][]*values.Invocation{},
		EntityAccesses:       map[*values.Entity][]*values.Access{},
	}

	// Initialize Invocation, set dependencies and orchestrator
	dependencyInvocations := []*values.Invocation{}
	for _, cluster := range initialRedesign.Feature.Clusters {
		redesign.InvocationsByCluster[cluster] = []*values.Invocation{
			{
				Cluster:         cluster,
				Redesign:        redesign,
				Type:            "",
				Accesses:        []*values.Access{},
				NextInvocations: []*values.Invocation{},
			},
		}

		if cluster != orchestrator {
			dependencyInvocations = append(dependencyInvocations, redesign.InvocationsByCluster[cluster][0])
		}
	}
	redesign.FirstInvocation = redesign.InvocationsByCluster[orchestrator][0]
	redesign.FirstInvocation.NextInvocations = dependencyInvocations

	// Fill each invocation with all the pruned acesses from the monolith redesign
	var invocation *values.Invocation
	for entity, accesses := range initialRedesign.GetPrunedEntityAccesses() {
		invocation = redesign.InvocationsByCluster[entity.Cluster][0]
		invocation.AddEntityAccess(accesses[0].Entity.Name, accesses[0].Type)
	}

	return
}

func (svc *DefaultHandler) SwapRedesignOrchestrator(redesign *values.Redesign, newOrchestrator *values.Cluster) {
	oldOrchestratorInvocation := redesign.FirstInvocation

	redesign.FirstInvocation = redesign.InvocationsByCluster[newOrchestrator][0]
	redesign.FirstInvocation.NextInvocations = oldOrchestratorInvocation.NextInvocations
	redesign.FirstInvocation.NextInvocations = redesign.FirstInvocation.FindAndDeleteNextInvocation(*redesign.FirstInvocation)
	redesign.FirstInvocation.NextInvocations = append(redesign.FirstInvocation.NextInvocations, oldOrchestratorInvocation)

	redesign.InvocationsByCluster[oldOrchestratorInvocation.Cluster][0] = oldOrchestratorInvocation
	redesign.InvocationsByCluster[oldOrchestratorInvocation.Cluster][0].NextInvocations = []*values.Invocation{}
}
