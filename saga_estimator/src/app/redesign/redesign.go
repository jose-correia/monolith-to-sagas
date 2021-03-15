package redesign

import (
	"app/files"
	"app/metrics"
	"app/training"
	"fmt"
	"strconv"
	"sync"

	"github.com/go-kit/kit/log"
)

const (
	defaultRedesignName = "currentRedesign"
	// add the second best orchestrator as a line in the CSV data
	addSecondBestRedesignToDataset = false
	// use the new redesign rules in order to redesign. If false, will use the simple redesign,
	// with one invocation per cluster
	useRedesignRules = true
	// number of accesses previous to the invocation that will be taken into account to assert dependency
	// if set to 0, there will be a dependency if the previous cluster does any R, and the next one does a W
	previousReadDistanceThreshold = 3
)

var (
	wg       sync.WaitGroup
	mapMutex = sync.RWMutex{}
)

type RedesignHandler interface {
	EstimateCodebaseOrchestrators(*files.Codebase, map[string]string, bool, bool) [][]string
	EstimateBestControllerOrchestrator(*files.Decomposition, *files.Controller, *files.FunctionalityRedesign) (map[*files.FunctionalityRedesign]int, error)
	RedesignControllerUsingSimpleStrategy(*files.Controller, *files.FunctionalityRedesign, *files.Cluster) *files.FunctionalityRedesign
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

func (svc *DefaultHandler) EstimateCodebaseOrchestrators(codebase *files.Codebase, idToEntityMap map[string]string, useExpertDecompositions bool, trainingDatasetFormat bool) [][]string {
	var data [][]string

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
			if controller.Name == "VirtualEditionController.createTopicModelling" {
				continue
			}
			wg.Add(1)
			go func(controller *files.Controller) {
				defer wg.Done()
				if len(controller.EntitiesPerCluster) < 3 {
					//svc.logger.Log("In order to decide the best redesign the controller must have more than 2 clusters.. Skiping %s", controller.Name)
					return
				}

				initialRedesign := controller.GetFunctionalityRedesign()

				controllerTrainingFeatures := svc.trainingHandler.CalculateControllerTrainingFeatures(initialRedesign)

				bestRedesigns, _ := svc.EstimateBestControllerOrchestrator(decomposition, controller, initialRedesign)

				for redesign, orchestratorID := range bestRedesigns {
					if trainingDatasetFormat {
						data = svc.trainingHandler.AddDataToTrainingDataset(data, codebase, controller, controllerTrainingFeatures, orchestratorID, idToEntityMap)
					} else {
						data = svc.addResultToDataset(
							data,
							codebase,
							controller,
							initialRedesign,
							redesign,
							orchestratorID,
							idToEntityMap,
						)
					}

					if !addSecondBestRedesignToDataset {
						break
					}
				}
			}(controller)
		}
		wg.Wait()
	}

	return data
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
	inconsistencyComplexityReduction := initialRedesign.InconsistencyComplexity - bestRedesign.InconsistencyComplexity
	functionalityComplexityReduction := initialRedesign.FunctionalityComplexity - bestRedesign.FunctionalityComplexity

	data = append(data, []string{
		codebase.Name,
		controller.Name,
		strconv.Itoa(orchestratorID),
		entityNamesCSVFormat,
		controller.Type,
		strconv.Itoa(initialRedesign.SystemComplexity),
		strconv.Itoa(bestRedesign.SystemComplexity),
		strconv.Itoa(systemComplexityReduction),
		strconv.Itoa(initialRedesign.InconsistencyComplexity),
		strconv.Itoa(bestRedesign.InconsistencyComplexity),
		strconv.Itoa(inconsistencyComplexityReduction),
		strconv.Itoa(initialRedesign.FunctionalityComplexity),
		strconv.Itoa(bestRedesign.FunctionalityComplexity),
		strconv.Itoa(functionalityComplexityReduction),
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

		if useRedesignRules {
			currentRedesign = svc.RedesignControllerUsingRules(controller, initialRedesign, cluster)
		} else {
			if first {
				currentRedesign = svc.RedesignControllerUsingSimpleStrategy(controller, initialRedesign, cluster)
				first = false
			} else {
				svc.swapRedesignOrchestrator(currentRedesign, cluster)
			}
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

func (svc *DefaultHandler) RedesignControllerUsingSimpleStrategy(controller *files.Controller, initialRedesign *files.FunctionalityRedesign, orchestrator *files.Cluster) *files.FunctionalityRedesign {
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

					if initialInvocation.GetAccessType(idx) == "W" && newInvocation.Type != "COMPENSATABLE" {
						newInvocation.Type = "COMPENSATABLE"
					}
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
		fmt.Printf("%v can be simplified\n", controller.Name)
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

	if previousReadDistanceThreshold != 0 {
		var distance int
		var containsReadWithinThreshold bool
		prevInvocationAccesses := invocations[invocationIdx-1].ClusterAccesses
		for idx := len(prevInvocationAccesses) - 1; idx >= 0; idx-- {
			if invocations[invocationIdx-1].GetAccessType(idx) == "R" {
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
	} else if !invocations[invocationIdx-1].ContainsRead() {
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
