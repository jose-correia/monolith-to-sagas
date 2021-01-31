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
	CalculateCodebaseMetrics(*values.Codebase)
	CalculateFeatureComplexity(*values.Feature)
	CalculateClusterComplexityAndCohesion(*values.Cluster)
	CalculateClusterCoupling(*values.Cluster)
	CalculateRedesignComplexities(*values.Redesign)
}

type DefaultHandler struct {
	logger log.Logger
}

func New(logger log.Logger) MetricsHandler {
	return &DefaultHandler{
		logger: log.With(logger, "module", "codebaseHandler"),
	}
}

func (svc *DefaultHandler) CalculateCodebaseMetrics(codebase *values.Codebase) {
	var complexity float32
	var cohesion float32
	var coupling float32

	for _, feature := range codebase.Features {
		svc.CalculateFeatureComplexity(feature)
		svc.CalculateRedesignComplexities(feature.RedesignUsedForMetrics)
		complexity += feature.Complexity
	}

	for _, cluster := range codebase.Clusters {
		svc.CalculateClusterComplexityAndCohesion(cluster)
		cohesion += cluster.Cohesion

		svc.CalculateClusterCoupling(cluster)
		coupling += cluster.Coupling
	}

	codebase.Complexity = complexity / float32(len(codebase.Features))
	codebase.Cohesion = cohesion / float32(len(codebase.Clusters))
	codebase.Coupling = coupling / float32(len(codebase.Clusters))
}

func (svc *DefaultHandler) CalculateFeatureComplexity(feature *values.Feature) {
	if len(feature.Clusters) == 1 {
		feature.Complexity = 0
		return
	}

	var complexity float32
	for _, accesses := range feature.RedesignUsedForMetrics.EntityAccesses {
		for _, access := range accesses {
			complexity += float32(svc.numberFeaturesThatTouchEntity(access))
		}
	}

	feature.Complexity = float32(complexity)
	return
}

func (svc *DefaultHandler) numberFeaturesThatTouchEntity(access *values.Access) int {
	var numberFeaturesTouchingEntity int
	var containsEntity bool
	var entityAccesses []*values.Access

	for _, otherFeature := range access.Invocation.Redesign.Feature.Codebase.Features {
		entityAccesses, containsEntity = otherFeature.RedesignUsedForMetrics.EntityAccesses[access.Entity]

		if otherFeature == access.Invocation.Redesign.Feature || !containsEntity || len(otherFeature.Clusters) == 1 {
			continue
		}

		for _, entityAccess := range entityAccesses {
			if entityAccess.Type == access.GetOpositeAccessType() {
				numberFeaturesTouchingEntity++
				break
			}
		}
	}
	return numberFeaturesTouchingEntity
}

func (svc *DefaultHandler) CalculateClusterComplexityAndCohesion(cluster *values.Cluster) {
	var complexity float32
	var cohesion float32
	var numberEntitiesTouched float32
	for _, feature := range cluster.Features {
		numberEntitiesTouched = 0
		for _, entity := range cluster.Entities {
			if _, found := feature.RedesignUsedForMetrics.EntityAccesses[entity]; found {
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

func (svc *DefaultHandler) CalculateClusterCoupling(cluster *values.Cluster) {
	var coupling float32

	for _, clusterFeature := range cluster.Features {
		for dependencyCluster, entities := range clusterFeature.RedesignUsedForMetrics.ClusterCouplingDependencies[cluster] {
			coupling += float32(len(entities) / len(dependencyCluster.Entities))
		}

		nrCodebaseClusters := len(clusterFeature.Codebase.Clusters)
		if nrCodebaseClusters > 1 {
			coupling = coupling / float32(nrCodebaseClusters-1)
		}
	}

	cluster.Coupling = coupling
	return
}

func (svc *DefaultHandler) CalculateRedesignComplexities(redesign *values.Redesign) {
	if redesign.Feature.Type == Query {
		svc.queryRedesignComplexity(redesign)
	} else {
		svc.sagasRedesignComplexity(redesign)
	}
}

func (svc *DefaultHandler) queryRedesignComplexity(redesign *values.Redesign) {
	entitiesRead := redesign.GetEntitiesTouchedInMode(ReadAccess)

	for _, feature := range redesign.Feature.Codebase.Features {
		if feature.Name != redesign.Feature.Name && feature.Type == Saga {
			entitiesWritten := feature.RedesignUsedForMetrics.GetEntitiesTouchedInMode(WriteAccess)

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
	for entity, accesses := range redesign.EntityAccesses {
		for _, access := range accesses {
			if access.Type == WriteAccess {
				if access.Invocation.Type == Compensatable {
					redesign.FunctionalityComplexity++
					svc.systemComplexity(redesign, entity)
				}
			} else if access.Type == ReadAccess {
				svc.costOfRead(redesign, entity)
			}
		}
	}
}

func (svc *DefaultHandler) systemComplexity(redesign *values.Redesign, entity *values.Entity) {
	for _, feature := range entity.Features {
		if feature.Name != redesign.Feature.Name && len(feature.Clusters) > 1 {
			for _, access := range feature.RedesignUsedForMetrics.EntityAccesses[entity] {
				if access.Type == ReadAccess {
					redesign.SystemComplexity++
				}
			}
		}
	}
}

func (svc *DefaultHandler) costOfRead(redesign *values.Redesign, entity *values.Entity) {
	var containsEntity bool
	var accesses []*values.Access
	var entityIsWritten bool

	for _, feature := range redesign.Feature.Codebase.Features {
		accesses, containsEntity = feature.RedesignUsedForMetrics.EntityAccesses[entity]
		if feature == redesign.Feature || !containsEntity || len(feature.Clusters) == 1 {
			continue
		}

		entityIsWritten = false
		for _, access := range accesses {
			if access.Type == WriteAccess {
				entityIsWritten = true
			}
		}

		if !entityIsWritten {
			continue
		}

		redesign.FunctionalityComplexity++
	}
	return
}
