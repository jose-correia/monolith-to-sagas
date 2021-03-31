package redesign

import (
	"app/files"
	"app/metrics"
	"app/training"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/go-kit/kit/log"
)

const (
	defaultRedesignName = "currentRedesign"
	// add only the best redesign as a line in the CSV data
	onlyExportBestRedesign = false
	// number of accesses previous to the invocation that will be taken into account to assert dependency
	// if set to 0, there will be a dependency if the previous cluster does any R, and the next one does a W
	//previousReadDistanceThreshold = 3
	previousReadDistanceThreshold = 0
	printTraces                   = true
)

var (
	wg       sync.WaitGroup
	mapMutex = sync.RWMutex{}
)

type RedesignHandler interface {
	EstimateCodebaseOrchestrators(*files.Codebase, map[string]string, float32, bool, []string, bool) [][]string
	CreateSagaRedesigns(*files.Decomposition, *files.Controller, *files.FunctionalityRedesign) ([]*files.FunctionalityRedesign, error)
	RedesignControllerUsingRules(*files.Controller, *files.FunctionalityRedesign, *files.Cluster) *files.FunctionalityRedesign
}

type DefaultHandler struct {
	logger          log.Logger
	metricsHandler  metrics.MetricsHandler
	trainingHandler training.TrainingHandler
}

func New(logger log.Logger, metricsHandler metrics.MetricsHandler, trainingHandler training.TrainingHandler) RedesignHandler {
	return &DefaultHandler{
		logger:          log.With(logger, "module", "redesignHandler"),
		metricsHandler:  metricsHandler,
		trainingHandler: trainingHandler,
	}
}

func (svc *DefaultHandler) shouldUseController(controllerName string, controllersToUse []string) bool {
	if len(controllersToUse) != 0 {
		for _, name := range controllersToUse {
			if name == controllerName {
				return true
			}
		}
		return false
	}

	return true
}

func (svc *DefaultHandler) EstimateCodebaseOrchestrators(codebase *files.Codebase, idToEntityMap map[string]string, cutValue float32, useExpertDecompositions bool, controllersToUse []string, trainingDatasetFormat bool) [][]string {
	var data [][]string

	for _, dendogram := range codebase.Dendrograms {
		decomposition := dendogram.GetDecomposition(cutValue, useExpertDecompositions)
		if decomposition == nil {
			svc.logger.Log("Failed to get decomposition from dendogram")
			continue
		}

		// Add to each cluster, the list of controllers that use it
		for _, controller := range decomposition.Controllers {
			if !svc.shouldUseController(controller.Name, controllersToUse) {
				continue
			}

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
			if !svc.shouldUseController(controller.Name, controllersToUse) || controller.Name == "VirtualEditionController.createTopicModelling" || controller.Type == "QUERY" {
				continue
			}

			wg.Add(1)
			go func(controller *files.Controller) {
				defer wg.Done()
				if len(controller.EntitiesPerCluster) <= 2 {
					//svc.logger.Log("In order to decide the best redesign the controller must have more than 2 clusters.. Skiping %s", controller.Name)
					return
				}

				initialRedesign := controller.GetFunctionalityRedesign()

				controllerTrainingFeatures := svc.trainingHandler.CalculateControllerTrainingFeatures(initialRedesign)

				sagaRedesigns, _ := svc.CreateSagaRedesigns(decomposition, controller, initialRedesign)

				for idx, redesign := range sagaRedesigns {
					if trainingDatasetFormat {
						data = svc.trainingHandler.AddDataToTrainingDataset(data, codebase, controller, controllerTrainingFeatures, redesign.OrchestratorID, idToEntityMap)
					} else {
						data = svc.addResultToDataset(
							data,
							codebase,
							controller,
							initialRedesign,
							redesign,
							redesign.OrchestratorID,
							idToEntityMap,
						)
					}

					if idx == 0 && printTraces {
						fmt.Printf("\n\n---------- %v ----------\n\n", controller.Name)
						fmt.Printf("Initial redesign\n\n")
						svc.printRedesignTrace(initialRedesign.Redesign, idToEntityMap)

						fmt.Printf("\n\nSAGA\n")
						svc.printRedesignTrace(redesign.Redesign, idToEntityMap)
					}

					if onlyExportBestRedesign {
						break
					}
				}
			}(controller)
		}
		wg.Wait()
	}

	return data
}

func (svc *DefaultHandler) printRedesignTrace(invocations []*files.Invocation, idToEntityMap map[string]string) {
	for _, invocation := range invocations {
		fmt.Printf("\n- %v  (%v)\n", invocation.ClusterID, invocation.Type)
		for idx, _ := range invocation.ClusterAccesses {
			fmt.Printf("%v (%v) | ", idToEntityMap[strconv.Itoa(invocation.GetAccessEntityID(idx))], invocation.GetAccessType(idx))
		}
	}
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

	systemComplexityReduction := initialRedesign.SystemComplexity - bestRedesign.SystemComplexity
	functionalityComplexityReduction := initialRedesign.FunctionalityComplexity - bestRedesign.FunctionalityComplexity

	data = append(data, []string{
		codebase.Name,
		controller.Name,
		strconv.Itoa(orchestratorID),
		entityNamesCSVFormat,
		strconv.Itoa(initialRedesign.SystemComplexity),
		strconv.Itoa(bestRedesign.SystemComplexity),
		strconv.Itoa(systemComplexityReduction),
		strconv.Itoa(initialRedesign.FunctionalityComplexity),
		strconv.Itoa(bestRedesign.FunctionalityComplexity),
		strconv.Itoa(functionalityComplexityReduction),
	})

	return data
}

func (svc *DefaultHandler) CreateSagaRedesigns(decomposition *files.Decomposition, controller *files.Controller, initialRedesign *files.FunctionalityRedesign) ([]*files.FunctionalityRedesign, error) {
	sagaRedesigns := []*files.FunctionalityRedesign{}

	for clusterName := range controller.EntitiesPerCluster {
		cluster := decomposition.Clusters[clusterName]

		redesign := svc.RedesignControllerUsingRules(controller, initialRedesign, cluster)

		svc.metricsHandler.CalculateDecompositionMetrics(decomposition, controller, redesign)

		orchestratorID, _ := strconv.Atoi(clusterName)
		redesign.OrchestratorID = orchestratorID

		sagaRedesigns = append(sagaRedesigns, redesign)
	}

	// order the redesigns by ascending complexity
	sort.Slice(sagaRedesigns, func(i, j int) bool {
		// return sagaRedesigns[i].SystemComplexity < sagaRedesigns[j].SystemComplexity
		if sagaRedesigns[i].FunctionalityComplexity == sagaRedesigns[j].FunctionalityComplexity {
			return sagaRedesigns[i].SystemComplexity < sagaRedesigns[j].SystemComplexity
		}
		return sagaRedesigns[i].FunctionalityComplexity < sagaRedesigns[j].FunctionalityComplexity
	})

	return sagaRedesigns, nil
}

func (svc *DefaultHandler) RedesignControllerUsingRules(controller *files.Controller, initialRedesign *files.FunctionalityRedesign, orchestrator *files.Cluster) *files.FunctionalityRedesign {
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
	orchestratorID, _ := strconv.Atoi(orchestrator.Name)
	var invocationID int
	var prevInvocation *files.Invocation
	for _, initialInvocation := range initialRedesign.Redesign {
		if initialInvocation.ClusterID == -1 {
			continue
		}

		// if this one or the previous is not the orchestrator
		if initialInvocation.ClusterID != orchestratorID && (prevInvocation == nil || prevInvocation.ClusterID != orchestratorID) {
			// add empty orchestrator invocation
			invocation := &files.Invocation{
				Name:              fmt.Sprintf("%d: %d", invocationID, orchestratorID),
				ID:                invocationID,
				ClusterID:         orchestratorID,
				ClusterAccesses:   [][]interface{}{},
				RemoteInvocations: []int{},
				Type:              "RETRIABLE",
			}

			redesign.Redesign = append(redesign.Redesign, invocation)
			invocationID++
		}

		// add actual invocation
		invocationType := initialInvocation.GetTypeFromAccesses()
		invocation := &files.Invocation{
			Name:              fmt.Sprintf("%d: %d", invocationID, initialInvocation.ClusterID),
			ID:                invocationID,
			ClusterID:         initialInvocation.ClusterID,
			ClusterAccesses:   initialInvocation.ClusterAccesses,
			RemoteInvocations: []int{},
			Type:              invocationType,
		}

		redesign.Redesign = append(redesign.Redesign, invocation)
		prevInvocation = invocation
		invocationID++
	}

	// while any merge is done, iterate all the invocations
	var noMerges bool
	for !noMerges {
		// fmt.Printf("%v can be simplified\n", controller.Name)
		redesign.Redesign, noMerges = svc.mergeAllPossibleInvocations(redesign.Redesign)
	}

	return redesign
}

func (svc *DefaultHandler) mergeAllPossibleInvocations(invocations []*files.Invocation) ([]*files.Invocation, bool) {
	var changed bool
	var deleted int
	var isLast bool
	prevClusterInvocations := map[int][]int{}

	for idx := 0; idx < len(invocations); idx++ {
		var addToPreviousInvocations bool
		invocation := invocations[idx]

		prevInvocations, exists := prevClusterInvocations[invocation.ClusterID]

		if !exists {
			addToPreviousInvocations = true
		} else {
			prevInvocationIdx := prevInvocations[len(prevInvocations)-1]
			if idx == len(invocations)-1 {
				isLast = true
			}

			if !svc.isMergeableWithPrevious(invocations, prevInvocationIdx, idx, isLast) {
				addToPreviousInvocations = true
			} else {
				invocations, deleted = svc.mergeInvocations(invocations, prevClusterInvocations, prevInvocationIdx, idx)
				svc.pruneInvocationAccesses(invocations[prevInvocationIdx])
				changed = true

				// fix prevInvocations map after merge changes
				for cluster, prevInvocations := range prevClusterInvocations {
					for prevIdx, prevID := range prevInvocations {
						if prevID > idx-deleted {
							prevClusterInvocations[cluster][prevIdx] = prevID - deleted
						}
					}
				}

				idx -= deleted
			}
		}

		if addToPreviousInvocations {
			prevClusterInvocations[invocation.ClusterID] = append(prevClusterInvocations[invocation.ClusterID], idx)
		}
	}

	return invocations, changed
}

func (svc *DefaultHandler) isMergeableWithPrevious(
	invocations []*files.Invocation, prevInvocationIdx int, invocationIdx int, isLast bool,
) bool {
	if len(invocations[invocationIdx].ClusterAccesses) == 0 && !isLast {
		// fmt.Printf("not merge: %d\n", invocations[invocationIdx].ClusterID)
		return false
	}

	// if the invocation is just R, it can be merged
	if !invocations[invocationIdx].ContainsLock() || prevInvocationIdx == invocationIdx-1 {
		return true
	}

	prevInvocation := invocations[invocationIdx-1]
	// if the previous is an empty orchestrator call we take into consideration the one before
	if len(prevInvocation.ClusterAccesses) == 0 && invocationIdx > 1 {
		prevInvocation = invocations[invocationIdx-2]
	}

	if previousReadDistanceThreshold != 0 {
		var distance int
		var containsReadWithinThreshold bool

		for idx := len(prevInvocation.ClusterAccesses) - 1; idx >= 0; idx-- {
			if prevInvocation.GetAccessType(idx) == "R" {
				containsReadWithinThreshold = true
				break
			}

			distance++
			if distance == previousReadDistanceThreshold {
				break
			}
		}

		if !containsReadWithinThreshold {
			return true
		}
	} else if !prevInvocation.ContainsRead() {
		// fmt.Printf("merge: %d\n", invocations[invocationIdx].ClusterID)
		return true
	}

	// fmt.Printf("not merge: %d\n", invocations[invocationIdx].ClusterID)
	return false
}

func (svc *DefaultHandler) mergeInvocations(
	invocations []*files.Invocation, prevInvocations map[int][]int, prevInvocationIdx int, invocationIdx int,
) ([]*files.Invocation, int) {
	newInvocations := []*files.Invocation{}
	var invocationID int
	var deleted int
	for idx, invocation := range invocations {
		var removeFromPrevious bool

		if idx == prevInvocationIdx {
			// append the accesses to the previous invocation
			for _, access := range invocations[invocationIdx].ClusterAccesses {
				invocations[prevInvocationIdx].ClusterAccesses = append(invocations[prevInvocationIdx].ClusterAccesses, access)
			}
		}

		// if its the invocation previou to the one being merged
		if idx == invocationIdx-1 {
			// check if its empty (its the orchestrator) and can be deleted
			if len(invocation.ClusterAccesses) == 0 {
				removeFromPrevious = true
				deleted++
				continue
			}
		}

		// if its the invocation we are merging, delete
		if idx == invocationIdx {
			removeFromPrevious = true
			deleted++
			continue
		}

		if removeFromPrevious {
			var newPrevInvocations []int
			for _, prevIdx := range prevInvocations[invocation.ClusterID] {
				if prevIdx != idx {
					newPrevInvocations = append(newPrevInvocations, prevIdx)
				}
			}
		}

		invocation.ID = invocationID
		newInvocations = append(newInvocations, invocation)
		invocationID++
	}

	return newInvocations, deleted
}

func (svc *DefaultHandler) pruneInvocationAccesses(invocation *files.Invocation) {
	previousEntityAccesses := map[int]string{}
	newAccesses := [][]interface{}{}

	var containsLock bool

	for idx := range invocation.ClusterAccesses {
		entity := invocation.GetAccessEntityID(idx)
		accessType := invocation.GetAccessType(idx)

		if accessType == "W" {
			containsLock = true
		}

		previousAccess, exists := previousEntityAccesses[entity]
		if !exists {
			previousEntityAccesses[entity] = accessType
			continue
		}

		if previousAccess == "R" && accessType == "W" {
			previousEntityAccesses[entity] = "RW"
			continue
		}
	}

	for entity, accessType := range previousEntityAccesses {
		newAccesses = append(newAccesses, []interface{}{accessType, entity})
	}

	if containsLock {
		invocation.Type = "COMPENSATABLE"
	} else {
		invocation.Type = "RETRIABLE"
	}
	invocation.ClusterAccesses = newAccesses
	return
}
