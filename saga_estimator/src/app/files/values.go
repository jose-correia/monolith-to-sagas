package files

import (
	"strings"
)

type Trace struct {
	Name                   string                  `json:"name,omitempty"`
	Complexity             float32                 `json:"complexity,omitempty"`
	Entities               map[string]string       `json:"entities,omitempty"`
	EntitiesSeq            string                  `json:"entitiesSeq,omitempty"`
	FunctionalityRedesigns []FunctionalityRedesign `json:"functionalityRedesigns,omitempty"`
}

func (trace *Trace) getMonolithRedesign() *FunctionalityRedesign {
	return &trace.FunctionalityRedesigns[0]
}

// outside we append feature to codebase

type Invocation struct {
	Name              string `json:"name,omitempty"`
	ID                string `json:"id,omitempty"`
	Cluster           string `json:"cluster,omitempty"`
	AccessedEntities  string `json:"accessedEntities,omitempty"`
	RemoteInvocations []int  `json:"remoteInvocations,omitempty"`
	Type              string `json:"type,omitempty"`
}

func (i *Invocation) getAcessesEntities() (acesses []string) {
	invocationArray := strings.Replace(i.AccessedEntities, "[", "", -1)
	invocationArray = strings.Replace(invocationArray, `"`, "", -1)
	invocationArray = strings.Replace(invocationArray, "]", "", -1)
	acesses = strings.Split(invocationArray, ",")
	return
}

type FunctionalityRedesign struct {
	Name                    string       `json:"name,omitempty"`
	UsedForMetrics          bool         `json:"usedForMetrics,omitempty"`
	Redesign                []Invocation `json:"redesign,omitempty"`
	SystemComplexity        int          `json:"systemComplexity,omitempty"`
	FunctionalityComplexity int          `json:"functionalityComplexity,omitempty"`
	PivotTransaction        string       `json:"pivotTransaction,omitempty"`
}
