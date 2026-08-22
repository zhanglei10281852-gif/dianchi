package trace

import "fmt"

func AddNode(c *Chain, n Node) error {
	for _, v := range c.Nodes {
		if v.ID == n.ID {
			return fmt.Errorf("duplicate node")
		}
	}
	c.Nodes = append(c.Nodes, n)
	return nil
}
func AddEdge(c *Chain, e Edge) error {
	if e.FromID == e.ToID {
		return fmt.Errorf("self edge")
	}
	hasFrom, hasTo := false, false
	for _, n := range c.Nodes {
		hasFrom = hasFrom || n.ID == e.FromID
		hasTo = hasTo || n.ID == e.ToID
	}
	if !hasFrom || !hasTo {
		return fmt.Errorf("edge endpoint missing")
	}
	c.Edges = append(c.Edges, e)
	return nil
}
func Reachable(c Chain, start string) map[string]bool {
	out := map[string]bool{start: true}
	changed := true
	for changed {
		changed = false
		for _, e := range c.Edges {
			if out[e.FromID] && !out[e.ToID] {
				out[e.ToID] = true
				changed = true
			}
		}
	}
	return out
}
