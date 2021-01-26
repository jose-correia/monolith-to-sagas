package values

type Codebase struct {
	Name     string
	Clusters []*Cluster
	Features []*Feature
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
	Name           string
	Codebase       *Codebase
	Type           string // SAGA or QUERY
	Complexity     float32
	Clusters       []*Cluster
	EntityAccesses map[*Entity][]*Access
	Redesigns      []*Redesign
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

func (f *Feature) GetEntitiesTouchedInMode(mode string) (entities []*Entity) {
	for entity, accesses := range f.EntityAccesses {
		for _, access := range accesses {
			if access.Type == mode {
				entities = append(entities, entity)
				break
			}
		}
	}
	return
}

type Redesign struct {
	Name                    string
	Feature                 *Feature
	FirstInvocation         *Invocation
	InvocationsByEntity     map[*Entity][]*Invocation
	InvocationsByCluster    map[*Cluster][]*Invocation
	SystemComplexity        int
	FunctionalityComplexity int
	InconsistencyComplexity int
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
		Cluster:        cluster,
		Redesign:       r,
		Type:           invocationType,
		Accesses:       []*Access{},
		NextInvocation: nil,
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

type Cluster struct {
	Name         string
	Features     []*Feature
	Entities     []*Entity
	Dependencies []*Cluster
	Complexity   float32
	Cohesion     float32
	Coupling     float32
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

type Invocation struct {
	Cluster        *Cluster
	Redesign       *Redesign
	Type           string // COMPENSATABLE or PIVOT or RETRIABLE
	Accesses       []*Access
	NextInvocation *Invocation
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
	i.Redesign.Feature.EntityAccesses[entity] = append(i.Redesign.Feature.EntityAccesses[entity], access)
	return
}

type Entity struct {
	Name     string
	Cluster  *Cluster
	Features []*Feature
}

type Access struct {
	Entity *Entity
	Type   string // R or W
}
