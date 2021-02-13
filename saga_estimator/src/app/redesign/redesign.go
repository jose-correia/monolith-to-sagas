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
	defaultRedesignName            = "currentRedesign"
	addSecondBestRedesignToDataset = false
)

var (
	wg       sync.WaitGroup
	mapMutex = sync.RWMutex{}
)

type RedesignHandler interface {
	EstimateCodebaseOrchestrators(*files.Codebase, map[string]string, bool) [][]string
	EstimateBestControllerOrchestrator(*files.Decomposition, *files.Controller, *files.FunctionalityRedesign) (map[*files.FunctionalityRedesign]int, error)
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

func (svc *DefaultHandler) EstimateCodebaseOrchestrators(codebase *files.Codebase, idToEntityMap map[string]string, useExpertDecompositions bool) [][]string {
	csvData := [][]string{{"Codebase", "Feature", "Orchestrator", "Entities", "Initial System Complexity", "Final System Complexity", "Redesign Complexity"}}

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
					mapMutex.Lock()
					cluster.AddController(controller)
					mapMutex.Unlock()
				}
			}(controller)
		}
		wg.Wait()

		for _, controller := range decomposition.Controllers {
			wg.Add(1)
			go func(controller *files.Controller) {
				defer wg.Done()
				if len(controller.EntitiesPerCluster) < 3 {
					svc.logger.Log("In order to decide the best redesign the controller must have more than 2 clusters.. Skiping %s", controller.Name)
					return
				}

				initialRedesign := controller.GetFunctionalityRedesign()
				bestRedesigns, _ := svc.EstimateBestControllerOrchestrator(decomposition, controller, initialRedesign)

				for redesign, orchestratorID := range bestRedesigns {
					csvData = svc.addResultToDataset(
						csvData,
						codebase,
						controller,
						initialRedesign,
						redesign,
						orchestratorID,
						idToEntityMap,
					)

					if !addSecondBestRedesignToDataset {
						break
					}
				}
			}(controller)
		}
		wg.Wait()
	}

	return csvData
}

func (svc *DefaultHandler) addResultToDataset(
	data [][]string, codebase *files.Codebase, controller *files.Controller, initialRedesign *files.FunctionalityRedesign,
	bestRedesign *files.FunctionalityRedesign, orchestratorID int, idToEntityMap map[string]string,
) [][]string {
	clusterName := strconv.Itoa(orchestratorID)
	entityNames := []string{}
	for _, entityID := range controller.EntitiesPerCluster[clusterName] {
		entityNames = append(entityNames, idToEntityMap[strconv.Itoa(entityID)])
	}

	entityNamesCSVFormat := ""
	for _, name := range entityNames {
		entityNamesCSVFormat += name + ", "
	}

	data = append(data, []string{
		codebase.Name,
		controller.Name,
		strconv.Itoa(orchestratorID),
		entityNamesCSVFormat,
		strconv.Itoa(initialRedesign.SystemComplexity),
		strconv.Itoa(bestRedesign.SystemComplexity),
		strconv.Itoa(bestRedesign.FunctionalityComplexity),
	})

	return data
}

func (svc *DefaultHandler) EstimateBestControllerOrchestrator(decomposition *files.Decomposition, controller *files.Controller, initialRedesign *files.FunctionalityRedesign) (map[*files.FunctionalityRedesign]int, error) {
	var currentRedesign *files.FunctionalityRedesign

	bestRedesign := &files.FunctionalityRedesign{}
	var bestClusterID int
	bestCodebaseComplexity := 999999999999999999

	secondBestRedesign := &files.FunctionalityRedesign{}
	var secondBestClusterID int
	secondBestCodebaseComplexity := 999999999999999999

	first := true
	for clusterName := range controller.EntitiesPerCluster {
		cluster := decomposition.Clusters[clusterName]

		if first {
			currentRedesign = svc.RedesignControllerWithOrchestrator(controller, initialRedesign, cluster)
			first = false
		} else {
			svc.swapRedesignOrchestrator(currentRedesign, cluster)
		}

		svc.metricsHandler.CalculateDecompositionMetrics(decomposition, controller, currentRedesign)

		// metric := currentRedesign.SystemComplexity
		complexity := currentRedesign.FunctionalityComplexity

		if complexity < bestCodebaseComplexity {
			secondBestClusterID = bestClusterID
			*secondBestRedesign = *bestRedesign

			bestClusterID, _ = strconv.Atoi(clusterName)
			*bestRedesign = *currentRedesign
			bestCodebaseComplexity = complexity

			continue
		}

		if complexity < secondBestCodebaseComplexity {
			secondBestClusterID, _ = strconv.Atoi(clusterName)
			*secondBestRedesign = *currentRedesign
			secondBestCodebaseComplexity = complexity
		}
	}

	return map[*files.FunctionalityRedesign]int{
		bestRedesign:       bestClusterID,
		secondBestRedesign: secondBestClusterID,
	}, nil
}

func (svc *DefaultHandler) printRedesign(decomposition *files.Decomposition, controller *files.Controller, redesign *files.FunctionalityRedesign, orchestratorName string) {
	fmt.Printf("Orchestrator: %v\n", orchestratorName)
	fmt.Printf("System complexity: %v\n", decomposition.Complexity)
	fmt.Printf("Redesign functionality complexity: %v\n", redesign.FunctionalityComplexity)
	fmt.Printf("Redesign system complexity: %v\n\n", redesign.SystemComplexity)
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
}
