package itsy

import (
	"sort"
	"strings"
)

type NameList []Name

func (n NameList) Len() int {
	return len(n)
}

func (n NameList) Less(i, j int) bool {
	return n[i].Full < n[j].Full
}

func (n NameList) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n NameList) FullNames() []string {
	var result []string
	for _, name := range n {
		result = append(result, name.Full)
	}
	return result
}

func (n NameList) Lookup(fullName string) (name Name, ok bool) {
	for _, candidate := range n {
		if candidate.Full == fullName {
			return candidate, true
		}
	}
	return
}

type Name struct {
	Full  string // full name, e.g. "base.all.a.x"
	Base  string // base, e.g. "base"
	Extra string // extra name, e.g. "all.a.x"
	Topo  string // topology, e.g. "a.x"
	All   bool   // if this is an 'all' name for broadcast
}

func (n Name) String() string {
	return n.Full
}

func join(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "." + b
	}
}

func newName(base, middle, topo string, isAll bool) Name {
	var extra string
	if middle != "" {
		extra = join(extra, middle)
	}
	if topo != "" {
		extra = join(extra, topo)
	}
	full := extra
	if base != "" {
		full = join(base, extra)
	}
	return Name{
		Full:  full,
		Base:  base,
		Extra: extra,
		Topo:  topo,
		All:   isAll,
	}
}

// ExpandTopology expand the base label with all levels of the provided
// topology strings and filters our any duplicates.
// The base label MUST NOT end with a dot.
//
// Example: a base of "base" and topo ["a.b.c", "b"] with id "x123" is expanded to:
//
// - "base"
// - "base.id.x123"
// - "base.all"
// - "base.all.a"
// - "base.all.a.x"
// - "base.all.a.x.y"
// - "base.all.b"
// - "base.any"
// - "base.any.a"
// - "base.any.a.x"
// - "base.any.a.x.y"
// - "base.any.b"
func ExpandTopology(base string, topo []string, id string) (result NameList) {
	names := make(map[string]Name)
	addName := func(base, middle, topo string, isAll bool) {
		n := newName(base, middle, topo, isAll)
		names[n.Full] = n
	}

	addName(base, "", "", false)
	addName(base, "all", "", true)
	addName(base, "any", "", false)
	if id != "" {
		addName(base, "id", id, false)
	}
	for _, t := range topo {
		if t == "" {
			continue
		}
		for {
			// "foo.bar.baz", "foo.bar", "foo"
			addName(base, "all", t, true)
			addName(base, "any", t, false)
			idx := strings.LastIndexByte(t, '.')
			if idx < 0 {
				break
			}
			t = t[:idx]
		}
	}
	for _, n := range names {
		result = append(result, n)
	}
	sort.Sort(result)
	return result
}
