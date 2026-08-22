package trace

import "time"

type Node struct {
	ID, TenantID, LotID, ParentID, Kind, Label string
	CreatedAt                                  time.Time
}
type Edge struct {
	ID, TenantID, FromID, ToID, Relation string
	CreatedAt                            time.Time
}
type Chain struct {
	Nodes []Node
	Edges []Edge
}

func (c Chain) Roots() []Node {
	parents := map[string]bool{}
	for _, e := range c.Edges {
		parents[e.ToID] = true
	}
	out := make([]Node, 0)
	for _, n := range c.Nodes {
		if !parents[n.ID] {
			out = append(out, n)
		}
	}
	return out
}
func (c Chain) Valid() bool {
	for _, e := range c.Edges {
		if e.FromID == e.ToID {
			return false
		}
	}
	return len(c.Nodes) > 0
}
