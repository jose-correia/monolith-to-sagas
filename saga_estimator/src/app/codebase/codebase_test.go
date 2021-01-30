// +build unit

package codebase_test

import (
	"app/common/log"
	"app/files"
	"testing"

	"app/codebase"
	"app/codebase/values"

	"github.com/stretchr/testify/assert"
)

func initializeHandler() codebase.CodebaseHandler {
	logger := log.NewLogger()
	return codebase.New(
		logger, files.New(logger, "traces/files"),
	)
}

func TestRedesignFeatureWithOrchestrator(t *testing.T) {
	handler := initializeHandler()

	clusterA := &values.Cluster{Name: "Cluster A"}
	clusterB := &values.Cluster{Name: "Cluster B"}
	clusterC := &values.Cluster{Name: "Cluster C"}

	entityA1 := &values.Entity{Name: "Entity 1", Cluster: clusterA}
	entityA2 := &values.Entity{Name: "Entity 2", Cluster: clusterA}
	entityB1 := &values.Entity{Name: "Entity 1", Cluster: clusterB}
	entityB2 := &values.Entity{Name: "Entity 2", Cluster: clusterB}
	entityC1 := &values.Entity{Name: "Entity 1", Cluster: clusterC}
	entityC2 := &values.Entity{Name: "Entity 2", Cluster: clusterC}

	initialRedesign := &values.Redesign{
		Name:                    "",
		Feature:                 &values.Feature{},
		FirstInvocation:         &values.Invocation{},
		InvocationsByEntity:     map[*values.Entity][]*values.Invocation{},
		InvocationsByCluster:    map[*values.Cluster][]*values.Invocation{},
		EntityAccesses:          map[*values.Entity][]*values.Access{},
		SystemComplexity:        0,
		FunctionalityComplexity: 0,
		InconsistencyComplexity: 0,
	}

	// 4
	clusterBInvocation2 := &values.Invocation{
		Cluster:         clusterB,
		Redesign:        initialRedesign,
		Type:            "",
		Accesses:        []*values.Access{{Entity: entityB1, Type: "W"}, {Entity: entityB2, Type: "R"}},
		NextInvocations: []*values.Invocation{},
	}

	// 3
	clusterCInvocation := &values.Invocation{
		Cluster:         clusterC,
		Redesign:        initialRedesign,
		Type:            "",
		Accesses:        []*values.Access{{Entity: entityC1, Type: "R"}, {Entity: entityC2, Type: "R"}},
		NextInvocations: []*values.Invocation{clusterBInvocation2},
	}

	// 2
	clusterBInvocation := &values.Invocation{
		Cluster:         clusterB,
		Redesign:        initialRedesign,
		Type:            "",
		Accesses:        []*values.Access{{Entity: entityB1, Type: "R"}, {Entity: entityB2, Type: "W"}},
		NextInvocations: []*values.Invocation{clusterCInvocation},
	}

	// 1
	clusterAInvocation := &values.Invocation{
		Cluster:         clusterA,
		Redesign:        initialRedesign,
		Type:            "",
		Accesses:        []*values.Access{{Entity: entityA1, Type: "R"}, {Entity: entityA2, Type: "R"}},
		NextInvocations: []*values.Invocation{clusterBInvocation},
	}

	feature := &values.Feature{
		Clusters: []*values.Cluster{clusterA, clusterB, clusterC},
	}

	initialRedesign.Name = "test_redesign"
	initialRedesign.FirstInvocation = clusterAInvocation
	initialRedesign.Feature = feature
	initialRedesign.InvocationsByCluster = map[*values.Cluster][]*values.Invocation{clusterA: {clusterAInvocation}, clusterB: {clusterBInvocation, clusterBInvocation2}, clusterC: {clusterCInvocation}}
	initialRedesign.EntityAccesses = map[*values.Entity][]*values.Access{
		entityA1: {clusterAInvocation.Accesses[0]},
		entityA2: {clusterAInvocation.Accesses[1]},
		entityB1: {clusterBInvocation.Accesses[0], clusterBInvocation2.Accesses[0]},
		entityB2: {clusterBInvocation.Accesses[1], clusterBInvocation2.Accesses[1]},
		entityC1: {clusterCInvocation.Accesses[0]},
		entityC2: {clusterCInvocation.Accesses[1]},
	}

	result := handler.RedesignFeatureWithOrchestrator(initialRedesign, clusterA)

	// expectedClusterBInvocation := &values.Invocation{
	// 	Cluster:         clusterB,
	// 	Redesign:        result,
	// 	Type:            "",
	// 	Accesses:        []*values.Access{{Entity: entityB1, Type: "RW"}, {Entity: entityB2, Type: "W"}},
	// 	NextInvocations: []*values.Invocation{},
	// }

	// expectedClusterCInvocation := &values.Invocation{
	// 	Cluster:         clusterC,
	// 	Redesign:        result,
	// 	Type:            "",
	// 	Accesses:        []*values.Access{{Entity: entityC1, Type: "R"}, {Entity: entityC2, Type: "R"}},
	// 	NextInvocations: []*values.Invocation{},
	// }

	// expectedClusterAInvocation := &values.Invocation{
	// 	Cluster:         clusterA,
	// 	Redesign:        result,
	// 	Type:            "",
	// 	Accesses:        []*values.Access{{Entity: entityA1, Type: "R"}, {Entity: entityA2, Type: "R"}},
	// 	NextInvocations: []*values.Invocation{expectedClusterBInvocation, expectedClusterCInvocation},
	// }

	// expectedRedesign := &values.Redesign{
	// 	Name:            "OrchestratorRedesign",
	// 	FirstInvocation: clusterAInvocation,
	// 	Feature:         feature,
	// 	InvocationsByCluster: map[*values.Cluster][]*values.Invocation{
	// 		clusterA: {expectedClusterAInvocation},
	// 		clusterB: {expectedClusterBInvocation},
	// 		clusterC: {expectedClusterCInvocation},
	// 	},
	// 	InvocationsByEntity: map[*values.Entity][]*values.Invocation{
	// 		entityA1: {expectedClusterAInvocation},
	// 		entityA2: {expectedClusterAInvocation},
	// 		entityB1: {expectedClusterBInvocation},
	// 		entityB2: {expectedClusterBInvocation},
	// 		entityC1: {expectedClusterCInvocation},
	// 		entityC2: {expectedClusterCInvocation},
	// 	},
	// 	EntityAccesses: map[*values.Entity][]*values.Access{
	// 		entityA1: {expectedClusterAInvocation.Accesses[0]},
	// 		entityA2: {expectedClusterAInvocation.Accesses[1]},
	// 		entityB1: {expectedClusterBInvocation.Accesses[0]},
	// 		entityB2: {expectedClusterBInvocation.Accesses[1]},
	// 		entityC1: {expectedClusterCInvocation.Accesses[0]},
	// 		entityC2: {expectedClusterCInvocation.Accesses[1]},
	// 	},
	// 	SystemComplexity:        0,
	// 	FunctionalityComplexity: 0,
	// 	InconsistencyComplexity: 0,
	// }

	// assert.Equal(t, expectedRedesign, result)
	assert.Equal(t, clusterA.Name, result.FirstInvocation.Cluster.Name)
	assert.Equal(t, 2, len(result.FirstInvocation.Accesses))
	assert.Equal(t, "R", result.FirstInvocation.Accesses[0].Type)
	assert.Equal(t, "R", result.FirstInvocation.Accesses[1].Type)

	assert.Equal(t, clusterB.Name, result.FirstInvocation.NextInvocations[0].Cluster.Name)
	assert.Equal(t, 2, len(result.FirstInvocation.NextInvocations[0].Accesses))
	assert.Equal(t, "RW", result.FirstInvocation.NextInvocations[0].Accesses[0].Type)
	assert.Equal(t, "W", result.FirstInvocation.NextInvocations[0].Accesses[1].Type)

	assert.Equal(t, clusterC.Name, result.FirstInvocation.NextInvocations[1].Cluster.Name)
	assert.Equal(t, 2, len(result.FirstInvocation.NextInvocations[1].Accesses))
	assert.Equal(t, "R", result.FirstInvocation.NextInvocations[1].Accesses[0].Type)
	assert.Equal(t, "R", result.FirstInvocation.NextInvocations[1].Accesses[1].Type)
}

func TestSwapOrchestrators(t *testing.T) {
	handler := initializeHandler()

	clusterB := &values.Cluster{Name: "Cluster B"}
	clusterBInvocation := &values.Invocation{
		Cluster:  clusterB,
		Accesses: []*values.Access{{Entity: &values.Entity{Name: "Entity B"}, Type: "R"}},
	}

	clusterC := &values.Cluster{Name: "Cluster C"}
	clusterCInvocation := &values.Invocation{
		Cluster:  clusterC,
		Accesses: []*values.Access{{Entity: &values.Entity{Name: "Entity C"}, Type: "R"}},
	}

	clusterA := &values.Cluster{Name: "Cluster A"}
	clusterAInvocation := &values.Invocation{
		Cluster:         clusterA,
		Accesses:        []*values.Access{{Entity: &values.Entity{Name: "Entity A"}, Type: "R"}},
		NextInvocations: []*values.Invocation{clusterBInvocation, clusterCInvocation},
	}

	redesign := &values.Redesign{
		Name:            "test_redesign",
		FirstInvocation: clusterAInvocation,
		InvocationsByCluster: map[*values.Cluster][]*values.Invocation{
			clusterA: {clusterAInvocation},
			clusterB: {clusterBInvocation},
			clusterC: {clusterCInvocation},
		},
	}

	handler.SwapRedesignOrchestrator(redesign, clusterB)

	assert.Equal(t, clusterBInvocation, redesign.FirstInvocation)
	assert.Equal(t, []*values.Invocation{
		clusterCInvocation, clusterAInvocation,
	}, redesign.FirstInvocation.NextInvocations)
	assert.Equal(t, []*values.Invocation{}, redesign.InvocationsByCluster[clusterA][0].NextInvocations)
}
