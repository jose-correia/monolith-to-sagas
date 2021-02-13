// +build unit

package redesign_test

import (
	"app/common/log"
	"app/files"
	"app/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
)

func initializeHandler() metrics.MetricsHandler {
	logger := log.NewLogger()
	return metrics.New(logger)
}

func TestCalculateControllerComplexityAndDependencies(t *testing.T) {
	metrics := initializeHandler()

	decomposition := &files.Decomposition{
		Name:          "ldod-expert",
		CodebaseName:  "ldod-static",
		DendogramName: "0a0w30r70s",
		Expert:        true,
		CutValue:      0,
		Complexity:    0,
		Cohesion:      0,
		Coupling:      0,
		Clusters: map[string]*files.Cluster{
			"0": {
				Name:                 "",
				CouplingDependencies: map[string][]int{},
				Entities:             []int{},
				Controllers:          map[string]*files.Controller{},
			},
			"1": {
				Name:                 "",
				CouplingDependencies: map[string][]int{},
				Entities:             []int{},
				Controllers:          map[string]*files.Controller{},
			},
			"2": {
				Name:                 "",
				CouplingDependencies: map[string][]int{},
				Entities:             []int{},
				Controllers:          map[string]*files.Controller{},
			},
			"3": {
				Name:                 "",
				CouplingDependencies: map[string][]int{},
				Entities:             []int{},
				Controllers:          map[string]*files.Controller{},
			},
		},
		Controllers:           map[string]*files.Controller{},
		EntityIDToClusterName: map[string]string{},
	}

	controller := &files.Controller{
		Name: "AdminController.removeTweets",
		Type: "SAGA",
		Entities: map[string]int{
			"32": 2,
			"65": 3,
			"66": 3,
			"5":  3,
			"7":  3,
			"71": 2,
			"9":  3,
			"44": 3,
			"19": 2,
			"20": 2,
			"27": 3,
			"28": 3,
			"30": 3,
		},
		FunctionalityRedesigns: []*files.FunctionalityRedesign{
			{
				Name:           "",
				UsedForMetrics: false,
				Redesign:       []*files.Invocation{},
			},
		},
		EntitiesPerCluster: map[string][]int{
			"0": {19, 20, 30},
			"1": {5, 71, 44},
			"2": {32},
			"4": {65, 66, 7, 9, 27, 28},
		},
	}

	redesign := &files.FunctionalityRedesign{
		Name:           "",
		UsedForMetrics: false,
		Redesign:       []*files.Invocation{},
	}

	metrics.CalculateControllerComplexityAndDependencies(decomposition, controller, redesign)

	assert.Equal(t, 384, controller.Complexity)
}
