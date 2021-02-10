package redesign

import (
	"app/files"
	"app/metrics"
	"fmt"
	"strconv"
	"sync"

	"github.com/go-kit/kit/log"
)

const (
	defaultRedesignName = "OrchestratorRedesign"
)

var wg sync.WaitGroup

type RedesignHandler interface {
	EstimateCodebaseOrchestrators(*files.Codebase, map[string]string, bool)
	EstimateBestControllerOrchestrator(*files.Decomposition, *files.Controller) (int, int, error)
	RedesignControllerWithOrchestrator(*files.Controller, *files.FunctionalityRedesign, *files.Cluster) *files.FunctionalityRedesign
}

type DefaultHandler struct {
	logger         log.Logger
	metricsHandler metrics.MetricsHandler
}

func New(logger log.Logger, metricsHandler metrics.MetricsHandler) RedesignHandler {
	return &DefaultHandler{
		logger:         log.With(logger, "module", "redesignHandler"),
		metricsHandler: metricsHandler,
	}
}

func (svc *DefaultHandler) EstimateCodebaseOrchestrators(codebase *files.Codebase, idToEntityMap map[string]string, useExpertDecompositions bool) {
	for _, dendogram := range codebase.Dendrograms {
		decomposition := dendogram.GetDecomposition(useExpertDecompositions)
		if decomposition == nil {
			svc.logger.Log("Failed to get decomposition from dendogram")
			continue
		}

		// Add to each cluster, the list of controllers that use it
		for _, controller := range decomposition.Controllers {
			wg.Add(1)
			go func(controller *files.Controller) {
				defer wg.Done()
				for clusterName := range controller.EntitiesPerCluster {
					clusterID, _ := strconv.Atoi(clusterName)
					cluster := decomposition.GetClusterFromID(clusterID)
					cluster.AddController(controller)
				}
			}(controller)
			wg.Wait()
		}

		for _, controller := range decomposition.Controllers {
			wg.Add(1)
			go func(controller *files.Controller) {
				defer wg.Done()
				fmt.Printf("---------------------------------------------------------\n")
				fmt.Printf("\nController: %v\n", controller.Name)

				bestClusterID, bestCodebaseComplexity, err := svc.EstimateBestControllerOrchestrator(decomposition, controller)
				if err != nil {
					svc.logger.Log(err.Error())
					fmt.Printf("\nController %v doesnt have more than 2 clusters! Skipping..\n\n", controller.Name)
					return
				}

				clusterName := strconv.Itoa(bestClusterID)
				entityNames := []string{}
				for _, entityID := range controller.EntitiesPerCluster[clusterName] {
					entityNames = append(entityNames, idToEntityMap[strconv.Itoa(entityID)])
				}
				fmt.Printf("\nBest orchestrator: %v\nOrchestrator entities: %v\nRedesign system complexity: %v\n\n",
					bestClusterID, entityNames, bestCodebaseComplexity,
				)
			}(controller)
			wg.Wait()
		}
	}
}

func (svc *DefaultHandler) EstimateBestControllerOrchestrator(decomposition *files.Decomposition, controller *files.Controller) (int, int, error) {
	if len(controller.EntitiesPerCluster) < 3 {
		return 0, 0, fmt.Errorf("In order to decide the best redesign the controller must have more than 2 clusters")
	}

	initialRedesign := controller.GetFunctionalityRedesign()
	svc.printRedesign(decomposition, controller, initialRedesign, "")

	first := true
	var orchestratorRedesign *files.FunctionalityRedesign
	var bestClusterID int
	bestCodebaseComplexity := 999999999999999999
	for clusterName := range controller.EntitiesPerCluster {
		cluster := decomposition.Clusters[clusterName]

		if first {
			orchestratorRedesign = svc.RedesignControllerWithOrchestrator(controller, initialRedesign, cluster)
			first = false
		} else {
			svc.swapRedesignOrchestrator(orchestratorRedesign, cluster)
		}

		svc.metricsHandler.CalculateDecompositionMetrics(decomposition, controller, orchestratorRedesign)
		svc.printRedesign(decomposition, controller, orchestratorRedesign, clusterName)

		if orchestratorRedesign.SystemComplexity < bestCodebaseComplexity {
			bestClusterID, _ = strconv.Atoi(clusterName)
			bestCodebaseComplexity = orchestratorRedesign.SystemComplexity
		}

		// if orchestratorRedesign.FunctionalityComplexity < bestCodebaseComplexity {
		// 	bestClusterID, _ = strconv.Atoi(clusterName)
		// 	bestCodebaseComplexity = orchestratorRedesign.FunctionalityComplexity
		// }
	}

	return bestClusterID, bestCodebaseComplexity, nil
}

func (svc *DefaultHandler) printRedesign(decomposition *files.Decomposition, controller *files.Controller, redesign *files.FunctionalityRedesign, orchestratorName string) {
	fmt.Printf("Orchestrator: %v\n", orchestratorName)
	fmt.Printf("System complexity: %v\n", decomposition.Complexity)
	// fmt.Printf("System cohesion: %v\n", decomposition.Cohesion)
	// fmt.Printf("System coupling: %v\n", decomposition.Coupling)
	// fmt.Printf("Controller complexity: %v\n", controller.Complexity)
	fmt.Printf("Redesign functionality complexity: %v\n", redesign.FunctionalityComplexity)
	fmt.Printf("Redesign system complexity: %v\n\n", redesign.SystemComplexity)
	// fmt.Printf("Redesign inconsistency complexity: %v\n\n", redesign.InconsistencyComplexity)
}

func (svc *DefaultHandler) RedesignControllerWithOrchestrator(controller *files.Controller, initialRedesign *files.FunctionalityRedesign, orchestrator *files.Cluster) *files.FunctionalityRedesign {
	redesign := &files.FunctionalityRedesign{
		Name:                    defaultRedesignName,
		UsedForMetrics:          true,
		Redesign:                []*files.Invocation{},
		SystemComplexity:        0,
		FunctionalityComplexity: 0,
		InconsistencyComplexity: 0,
		PivotTransaction:        0,
	}

	// Initialize Invocation, set dependencies and orchestrator
	var dependencyInvocationIDs []int
	var orchestratorInvocation *files.Invocation
	var invocationID int
	for clusterName := range controller.EntitiesPerCluster {
		clusterID, _ := strconv.Atoi(clusterName)
		invocation := &files.Invocation{
			Name:              clusterName,
			ID:                invocationID,
			ClusterID:         clusterID,
			ClusterAccesses:   [][]interface{}{},
			RemoteInvocations: []int{},
			Type:              "RETRIABLE",
		}

		if clusterName != orchestrator.Name {
			dependencyInvocationIDs = append(dependencyInvocationIDs, invocation.ID)
		} else {
			invocation.Type = "COMPENSATABLE"
			orchestratorInvocation = invocation
		}

		redesign.Redesign = append(redesign.Redesign, invocation)
		invocationID++
	}

	orchestratorInvocation.RemoteInvocations = dependencyInvocationIDs

	for _, initialInvocation := range initialRedesign.Redesign {
		for _, newInvocation := range redesign.Redesign {
			if initialInvocation.ClusterID == newInvocation.ClusterID {
				for idx, _ := range initialInvocation.ClusterAccesses {
					newInvocation.AddPrunedAccess(
						initialInvocation.GetAccessEntityID(idx),
						initialInvocation.GetAccessType(idx),
					)
				}
			}
		}
	}

	// // Add coupling dependencies of the orchestrator
	// for _, invocation := range redesign.FirstInvocation.NextInvocations {
	// 	redesign.AddCouplingDependency(redesign.FirstInvocation, invocation)
	// }

	return redesign
}

func (svc *DefaultHandler) swapRedesignOrchestrator(redesign *files.FunctionalityRedesign, newOrchestrator *files.Cluster) {
	var dependencyInvocationIDs []int
	var orchestratorInvocation *files.Invocation
	for _, invocation := range redesign.Redesign {
		clusterID, _ := strconv.Atoi(newOrchestrator.Name)
		if invocation.ClusterID == clusterID {
			invocation.Type = "COMPENSATABLE"
			orchestratorInvocation = invocation
			continue
		}

		dependencyInvocationIDs = append(dependencyInvocationIDs, invocation.ID)

		if len(invocation.ClusterAccesses) > 0 {
			invocation.Type = "RETRIABLE"
			invocation.RemoteInvocations = []int{}
		}
	}

	orchestratorInvocation.RemoteInvocations = dependencyInvocationIDs
	redesign.FunctionalityComplexity = 0
	redesign.InconsistencyComplexity = 0
	redesign.SystemComplexity = 0

	// Add coupling dependencies of the orchestrator
	// redesign.ClusterCouplingDependencies = make(map[*values.Cluster]map[*values.Cluster][]*values.Entity)
	// for _, invocation := range redesign.FirstInvocation.NextInvocations {
	// 	redesign.AddCouplingDependency(redesign.FirstInvocation, invocation)
	// }
}
