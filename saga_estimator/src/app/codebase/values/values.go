package values

import "fmt"

type Codebase struct {
	Name     string     `json:"name,omitempty"`
	Clusters []*Cluster `json:"clusters,omitempty"`
	Features []*Feature `json:"features,omitempty"`
}

func (c *Codebase) GetClusterByName(name string) (cluster *Cluster, exists bool) {
	for _, cluster = range c.Clusters {
		if cluster.Name == name {
			exists = true
			return
		}
	}
	return
}

func (c *Codebase) GetFeatureByName(name string) (feature *Feature, exists bool) {
	for _, feature = range c.Features {
		if feature.Name == name {
			exists = true
			return
		}
	}
	return
}

type Feature struct {
	Name       string      `json:"name,omitempty"`
	Codebase   *Codebase   `json:"codebase,omitempty"`
	Type       string      `json:"type,omitempty"` // SAGA or QUERY
	Complexity float32     `json:"complexity,omitempty"`
	Clusters   []*Cluster  `json:"clusters,omitempty"`
	Redesigns  []*Redesign `json:"redesigns,omitempty"`
}

func (f *Feature) GetMonolithRedesign() *Redesign {
	return f.Redesigns[0]
}

func (f *Feature) GetClusterByName(name string) (cluster *Cluster, exists bool) {
	for _, cluster = range f.Clusters {
		if cluster.Name == name {
			exists = true
			return
		}
	}
	return
}

type Redesign struct {
	Name                    string                     `json:"name,omitempty"`
	Feature                 *Feature                   `json:"feature,omitempty"`
	FirstInvocation         *Invocation                `json:"first_invocation,omitempty"`
	InvocationsByEntity     map[*Entity][]*Invocation  `json:"invocations_by_entity,omitempty"`
	InvocationsByCluster    map[*Cluster][]*Invocation `json:"invocations_by_cluster,omitempty"`
	EntityAccesses          map[*Entity][]*Access      `json:"entity_accesses,omitempty"`
	SystemComplexity        int                        `json:"system_complexity,omitempty"`
	FunctionalityComplexity int                        `json:"functionality_complexity,omitempty"`
	InconsistencyComplexity int                        `json:"inconsistency_complexity,omitempty"`
}

func (r *Redesign) AddInvocation(clusterName string, invocationType string, accessedEntities []string) (invocation *Invocation) {
	cluster, found := r.Feature.GetClusterByName(clusterName)
	if !found {
		cluster, found = r.Feature.Codebase.GetClusterByName(clusterName)
		if !found {
			cluster = &Cluster{
				Name:         clusterName,
				Features:     []*Feature{r.Feature},
				Entities:     []*Entity{},
				Dependencies: []*Cluster{},
				Complexity:   0,
				Cohesion:     0,
			}
			r.Feature.Codebase.Clusters = append(r.Feature.Codebase.Clusters, cluster)
		}
		r.Feature.Clusters = append(r.Feature.Clusters, cluster)
	}

	invocation = &Invocation{
		Cluster:         cluster,
		Redesign:        r,
		Type:            invocationType,
		Accesses:        []*Access{},
		NextInvocations: []*Invocation{},
	}

	r.InvocationsByCluster[cluster] = append(r.InvocationsByCluster[cluster], invocation)

	for accessID, entityName := range accessedEntities {
		operation := accessedEntities[accessID+1]
		access := invocation.AddEntityAccess(entityName, operation)
		if access.Type == "W" {
			r.Feature.Type = "SAGA"
		}

		accessID++
		if accessID == len(accessedEntities)-1 {
			break
		}
	}

	return
}

func (r *Redesign) GetEntitiesTouchedInMode(mode string) (entities []*Entity) {
	for entity, accesses := range r.EntityAccesses {
		for _, access := range accesses {
			if access.Type == mode {
				entities = append(entities, entity)
				break
			}
		}
	}
	return
}

func (r *Redesign) GetPrunedEntityAccesses() map[*Entity][]*Access {
	entityAccesses := make(map[*Entity][]*Access)
	var operation string
	for entity, accesses := range r.EntityAccesses {
		for _, access := range accesses {
			if access.Type == "R" {
				if operation == "" {
					operation = "R"
				}
			} else {
				if operation == "" {
					operation = "W"
				} else if operation == "R" {
					operation = "RW"
				}
				break
			}
		}
		entityAccesses[entity] = []*Access{
			{
				Entity: entity,
				Type:   operation,
			},
		}
		operation = ""
	}
	return entityAccesses
}

type Cluster struct {
	Name         string     `json:"name,omitempty"`
	Features     []*Feature `json:"features,omitempty"`
	Entities     []*Entity  `json:"entities,omitempty"`
	Dependencies []*Cluster `json:"dependencies,omitempty"`
	Complexity   float32    `json:"complexity,omitempty"`
	Cohesion     float32    `json:"cohesion,omitempty"`
	Coupling     float32    `json:"coupling,omitempty"`
}

func (c *Cluster) GetEntityByName(name string) (entity *Entity, exists bool) {
	for _, entity = range c.Entities {
		if entity.Name == name {
			exists = true
			return
		}
	}
	return
}

func (c *Cluster) GetFeatureByName(name string) (feature *Feature, exists bool) {
	for _, feature = range c.Features {
		if feature.Name == name {
			exists = true
			return
		}
	}
	return
}

func (c *Cluster) AddDependencyIfNew(dependency *Cluster) {
	dependencyAlreadyAdded := false
	for _, cluster := range c.Dependencies {
		if cluster == dependency {
			dependencyAlreadyAdded = true
		}
	}
	if !dependencyAlreadyAdded {
		c.Dependencies = append(c.Dependencies, dependency)
	}
}

type Invocation struct {
	Cluster         *Cluster      `json:"cluster,omitempty"`
	Redesign        *Redesign     `json:"redesign,omitempty"`
	Type            string        `json:"type,omitempty"` // COMPENSATABLE or PIVOT or RETRIABLE
	Accesses        []*Access     `json:"accesses,omitempty"`
	NextInvocations []*Invocation `json:"next_invocations,omitempty"`
}

func (i *Invocation) FindAndDeleteNextInvocation(invocation Invocation) (newInvocations []*Invocation) {
	for _, i := range i.NextInvocations {
		if i.Cluster.Name != invocation.Cluster.Name {
			fmt.Printf("in %v", i.NextInvocations)
			newInvocations = append(newInvocations, i)
		}
	}
	return
}

func (i *Invocation) AddEntityAccess(entityName string, operation string) (access *Access) {
	entity, found := i.Cluster.GetEntityByName(entityName)
	if !found {
		entity = &Entity{
			Name:     entityName,
			Cluster:  i.Cluster,
			Features: []*Feature{i.Redesign.Feature},
		}
		i.Cluster.Entities = append(i.Cluster.Entities, entity)
	}

	i.Redesign.InvocationsByEntity[entity] = append(i.Redesign.InvocationsByEntity[entity], i)

	access = &Access{
		Entity: entity,
		Type:   operation,
	}

	i.Accesses = append(i.Accesses, access)
	i.Redesign.EntityAccesses[entity] = append(i.Redesign.EntityAccesses[entity], access)
	return
}

type Entity struct {
	Name     string     `json:"name,omitempty"`
	Cluster  *Cluster   `json:"cluster,omitempty"`
	Features []*Feature `json:"features,omitempty"`
}

type Access struct {
	Entity *Entity `json:"entity,omitempty"`
	Type   string  `json:"type,omitempty"` // R or W
}
