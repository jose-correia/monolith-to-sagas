package metrics

import (
	"app/files"
	"strconv"

	"github.com/go-kit/kit/log"
)

const (
	ReadMode      = 1
	WriteMode     = 2
	ReadWriteMode = 3
	Compensatable = "COMPENSATABLE"
	Saga          = "SAGA"
	Query         = "QUERY"
)

type MetricsHandler interface {
	CalculateDecompositionMetrics(*files.Decomposition, *files.Controller, *files.FunctionalityRedesign)
	CalculateControllerComplexityAndDependencies(*files.Decomposition, *files.Controller, *files.FunctionalityRedesign)
	CalculateClusterComplexityAndCohesion(*files.Cluster)
	// CalculateClusterCoupling(*values.Cluster)
	CalculateRedesignComplexities(*files.Decomposition, *files.Controller, *files.FunctionalityRedesign)
}

type DefaultHandler struct {
	logger log.Logger
}

func New(logger log.Logger) MetricsHandler {
	return &DefaultHandler{
		logger: log.With(logger, "module", "codebaseHandler"),
	}
}

func (svc *DefaultHandler) CalculateDecompositionMetrics(decomposition *files.Decomposition, controller *files.Controller, redesign *files.FunctionalityRedesign) {
	var complexity float32
	var cohesion float32
	var coupling float32

	for _, controller := range decomposition.Controllers {
		svc.CalculateControllerComplexityAndDependencies(decomposition, controller, redesign)
		svc.CalculateRedesignComplexities(decomposition, controller, redesign)
		complexity += controller.Complexity
	}

	for _, cluster := range decomposition.Clusters {
		svc.CalculateClusterComplexityAndCohesion(cluster)
		cohesion += cluster.Cohesion

		// svc.CalculateClusterCoupling(cluster)
		// coupling += cluster.Coupling
	}

	decomposition.Complexity = complexity / float32(len(decomposition.Controllers))
	decomposition.Cohesion = cohesion / float32(len(decomposition.Clusters))
	decomposition.Coupling = coupling / float32(len(decomposition.Clusters))
}

func (svc *DefaultHandler) CalculateControllerComplexityAndDependencies(decomposition *files.Decomposition, controller *files.Controller, redesign *files.FunctionalityRedesign) {
	if len(controller.EntitiesPerCluster) <= 1 {
		controller.Complexity = 0
		return
	}

	var complexity float32
	for idx, invocation := range redesign.Redesign {
		if invocation.ClusterID == -1 {
			continue
		}

		cluster := decomposition.GetClusterFromID(invocation.ClusterID)
		for i := idx; i < len(redesign.Redesign); i++ {
			cluster.AddCouplingDependency(
				redesign.GetInvocation(i).ClusterID,
				redesign.GetInvocation(i).GetAccessEntityID(0),
			)
		}

		for i := range invocation.ClusterAccesses {
			mode := files.MapAccessTypeToMode(invocation.GetAccessType(i))
			complexity += float32(svc.numberControllersThatTouchEntity(decomposition, controller, invocation.GetAccessEntityID(i), mode))
		}
	}

	controller.Complexity = float32(complexity)
	return
}

func (svc *DefaultHandler) numberControllersThatTouchEntity(decomposition *files.Decomposition, controller *files.Controller, entityID int, mode int) int {
	var numberFeaturesTouchingEntity int

	for _, otherController := range decomposition.Controllers {
		entityMode, containsEntity := otherController.GetEntityMode(entityID)
		if otherController.Name == controller.Name || len(otherController.EntitiesPerCluster) <= 1 || !containsEntity {
			continue
		}

		if entityMode != mode {
			numberFeaturesTouchingEntity++
		}
	}

	return numberFeaturesTouchingEntity
}

func (svc *DefaultHandler) CalculateClusterComplexityAndCohesion(cluster *files.Cluster) {
	var complexity float32
	var cohesion float32
	var numberEntitiesTouched float32

	for _, controller := range cluster.Controllers {
		for entityName := range controller.Entities {
			entityID, _ := strconv.Atoi(entityName)
			if cluster.ContainsEntity(entityID) {
				numberEntitiesTouched++
			}
		}

		cohesion += numberEntitiesTouched / float32(len(cluster.Entities))
		complexity += controller.Complexity
	}

	complexity /= float32(len(cluster.Controllers))
	cluster.Complexity = complexity

	cohesion /= float32(len(cluster.Controllers))
	cluster.Cohesion = cohesion
	return
}

// func (svc *DefaultHandler) CalculateClusterCoupling(cluster *files.Cluster) {
// 	var coupling float32

// 	for _, clusterFeature := range cluster.Features {
// 		for dependencyCluster, entities := range clusterFeature.RedesignUsedForMetrics.ClusterCouplingDependencies[cluster] {
// 			coupling += float32(len(entities)) / float32(len(dependencyCluster.Entities))
// 		}

// 		nrCodebaseClusters := len(clusterFeature.Codebase.Clusters)
// 		if nrCodebaseClusters > 1 {
// 			coupling = coupling / float32(nrCodebaseClusters-1)
// 		}
// 	}

// 	cluster.Coupling = coupling
// 	return
// }

func (svc *DefaultHandler) CalculateRedesignComplexities(decomposition *files.Decomposition, controller *files.Controller, redesign *files.FunctionalityRedesign) {
	if controller.Type == Query {
		svc.queryRedesignComplexity(decomposition, controller, redesign)
	} else {
		svc.sagasRedesignComplexity(decomposition, controller, redesign)
	}
}

func (svc *DefaultHandler) queryRedesignComplexity(decomposition *files.Decomposition, controller *files.Controller, redesign *files.FunctionalityRedesign) {
	entitiesRead := controller.EntitiesTouchedInMode(files.MapAccessTypeToMode("R"))
	var entitiesReadThatAreWrittenInOther []int
	var clustersInCommon []*files.Cluster

	for _, otherController := range decomposition.Controllers {
		if otherController.Name == controller.Name || len(otherController.EntitiesPerCluster) <= 1 || otherController.Type != "SAGA" {
			continue
		}

		entitiesWritten := otherController.EntitiesTouchedInMode(files.MapAccessTypeToMode("W"))
		for entity := range entitiesRead {
			_, written := entitiesWritten[entity]
			if written {
				entitiesReadThatAreWrittenInOther = append(entitiesReadThatAreWrittenInOther, entity)
			}
		}

		for entityID := range entitiesReadThatAreWrittenInOther {
			cluster := decomposition.GetEntityCluster(entityID)
			clustersInCommon = append(clustersInCommon, cluster)
		}

		if len(clustersInCommon) > 1 {
			redesign.InconsistencyComplexity += len(clustersInCommon)
		}
	}
}

func (svc *DefaultHandler) sagasRedesignComplexity(decomposition *files.Decomposition, controller *files.Controller, redesign *files.FunctionalityRedesign) {
	for _, invocation := range redesign.Redesign {
		for i := range invocation.ClusterAccesses {
			entity := invocation.GetAccessEntityID(i)
			mode := files.MapAccessTypeToMode(invocation.GetAccessType(i))

			if mode >= WriteMode { // 2 -> W, 3 -> RW
				if invocation.Type == "COMPENSATABLE" {
					redesign.FunctionalityComplexity++
					svc.systemComplexity(decomposition, controller, redesign, entity)
				}
				continue
			}

			if mode != WriteMode { // 1 -> R, 3 -> RW
				svc.costOfRead(decomposition, controller, redesign, entity)
			}
		}
	}
}

func (svc *DefaultHandler) systemComplexity(decomposition *files.Decomposition, controller *files.Controller, redesign *files.FunctionalityRedesign, entity int) {
	for _, otherController := range decomposition.Controllers {
		mode, containsEntity := otherController.GetEntityMode(entity)
		if otherController.Name == controller.Name || len(otherController.EntitiesPerCluster) <= 1 || !containsEntity || mode == WriteMode {
			continue
		}

		redesign.SystemComplexity++
	}
}

func (svc *DefaultHandler) costOfRead(decomposition *files.Decomposition, controller *files.Controller, redesign *files.FunctionalityRedesign, entity int) {
	for _, otherController := range decomposition.Controllers {
		mode, containsEntity := otherController.GetEntityMode(entity)
		if otherController.Name == controller.Name || len(otherController.EntitiesPerCluster) <= 1 || !containsEntity {
			continue
		}

		mode, exists := otherController.GetEntityMode(entity)
		if exists && mode >= WriteMode {
			redesign.FunctionalityComplexity++
		}
	}

	return
}
