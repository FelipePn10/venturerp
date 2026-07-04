package entity

import "sort"

// SelectPrimarySubstituteComponents keeps standalone components and only the
// primary component of each substitute group. Primary = lowest priority; ties
// are resolved by sequence and then child code for deterministic planning.
func SelectPrimarySubstituteComponents(children []*ItemStructure) []*ItemStructure {
	out := make([]*ItemStructure, 0, len(children))
	groupBest := make(map[int16]*ItemStructure)

	for _, child := range children {
		if child == nil {
			continue
		}
		if child.SubstituteGroup <= 0 {
			out = append(out, child)
			continue
		}
		if best, ok := groupBest[child.SubstituteGroup]; !ok || SubstitutePrecedes(child, best) {
			groupBest[child.SubstituteGroup] = child
		}
	}

	groups := make([]int, 0, len(groupBest))
	for group := range groupBest {
		groups = append(groups, int(group))
	}
	sort.Ints(groups)
	for _, group := range groups {
		out = append(out, groupBest[int16(group)])
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sequence != out[j].Sequence {
			return out[i].Sequence < out[j].Sequence
		}
		return out[i].ChildCode < out[j].ChildCode
	})
	return out
}

func SubstitutePrecedes(a, b *ItemStructure) bool {
	ap := a.SubstitutePriority
	if ap < 1 {
		ap = 1
	}
	bp := b.SubstitutePriority
	if bp < 1 {
		bp = 1
	}
	if ap != bp {
		return ap < bp
	}
	if a.Sequence != b.Sequence {
		return a.Sequence < b.Sequence
	}
	return a.ChildCode < b.ChildCode
}
