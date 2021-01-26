package metrics

import (
	"app/codebase/values"

	"github.com/go-kit/kit/log"
)

const (
	ReadAccess    = "R"
	WriteAccess   = "W"
	Compensatable = "COMPENSATABLE"
	Saga          = "SAGA"
	Query         = "QUERY"
)

type MetricsHandler interface {
	FeatureComplexityAndClusterDependencies(feature *values.Feature) (complexity float32, err error)
	ClusterComplexityAndCohesion(cluster *values.Cluster)
	ClusterCoupling(cluster *values.Cluster)
	RedesignComplexities(redesign *values.Redesign)
}

type DefaultHandler struct {
	logger log.Logger
}

func New(logger log.Logger) MetricsHandler {
	return &DefaultHandler{
		logger: log.With(logger, "module", "codebaseHandler"),
	}
}

func (svc *DefaultHandler) FeatureComplexityAndClusterDependencies(feature *values.Feature) (complexity float32, err error) {

	return
}

func (svc *DefaultHandler) ClusterComplexityAndCohesion(cluster *values.Cluster) {
	var complexity float32
	var cohesion float32
	var numberEntitiesTouched float32
	for _, feature := range cluster.Features {
		numberEntitiesTouched = 0
		for _, entity := range cluster.Entities {
			if _, found := feature.EntityAccesses[entity]; found {
				numberEntitiesTouched++
			}
		}

		cohesion += numberEntitiesTouched / float32(len(cluster.Entities))
		complexity += feature.Complexity
	}

	complexity /= float32(len(cluster.Features))
	// complexity
	cluster.Complexity = complexity

	cohesion /= float32(len(cluster.Features))
	// cohesion
	cluster.Cohesion = cohesion
	return
}

func (svc *DefaultHandler) ClusterCoupling(cluster *values.Cluster) {
	// TODO
	return
}

func (svc *DefaultHandler) RedesignComplexities(redesign *values.Redesign) {
	if redesign.Feature.Type == Query {
		svc.queryRedesignComplexity(redesign)
	} else {
		svc.sagasRedesignComplexity(redesign)
	}
}

func (svc *DefaultHandler) queryRedesignComplexity(redesign *values.Redesign) {
	entitiesRead := redesign.Feature.GetEntitiesTouchedInMode(ReadAccess)

	for _, feature := range redesign.Feature.Codebase.Features {
		if feature.Name != redesign.Feature.Name && feature.Type == Saga {
			entitiesWritten := feature.GetEntitiesTouchedInMode(WriteAccess)

			var matches []*values.Entity
			var commonClusters []*values.Cluster
			var commonClustersContains bool
			for _, entity := range entitiesWritten {
				for _, readEntity := range entitiesRead {
					if readEntity.Name == entity.Name {
						matches = append(matches, entity)

						commonClustersContains = false
						for _, commonCluster := range commonClusters {
							if commonCluster == readEntity.Cluster {
								commonClustersContains = true
							}
						}
						if !commonClustersContains {
							commonClusters = append(commonClusters, readEntity.Cluster)
						}
					}
				}
			}

			if len(commonClusters) > 1 {
				redesign.InconsistencyComplexity += len(commonClusters)
			}
		}
	}
}

func (svc *DefaultHandler) sagasRedesignComplexity(redesign *values.Redesign) {
	for invocation := redesign.FirstInvocation; invocation != nil; invocation = invocation.NextInvocation {
		for _, access := range invocation.Accesses {
			if access.Type == WriteAccess {
				if invocation.Type == Compensatable {
					redesign.FunctionalityComplexity++
					svc.systemComplexity(redesign, access.Entity)
				}
			} else if access.Type == ReadAccess {
				// svc.sostOfRead() //TODO
			}
		}
	}
}

func (svc *DefaultHandler) systemComplexity(redesign *values.Redesign, entity *values.Entity) {
	for _, feature := range entity.Features {
		if feature.Name != redesign.Feature.Name {
			for _, access := range feature.EntityAccesses[entity] {
				if access.Type == ReadAccess { // TODO this.controllerClusters.get(otherController.getName()).size() > 1) {
					redesign.SystemComplexity++
				}
			}
		}
	}
}

func (svc *DefaultHandler) sostOfRead(codebase *values.Codebase) {

	return
}
