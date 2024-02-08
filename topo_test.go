package itsy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandTopology_NameList(t *testing.T) {
	names := ExpandTopology("base", []string{"a.b"}, "id1")
	n, ok := names.Lookup("base.all.a")
	assert.True(t, ok)
	assert.Equal(t, "base.all.a", n.Full)
	assert.Equal(t, "base", n.Base)
	assert.Equal(t, "all.a", n.Extra)
	assert.Equal(t, "a", n.Topo)
	assert.True(t, n.All)
}

func TestExpandTopology(t *testing.T) {
	tests := []struct {
		name       string
		base       string
		topo       []string
		id         string
		wantResult []string
	}{
		{
			name: "empty-list",
			base: "base",
			topo: nil,
			id:   "",
			wantResult: []string{
				"base",
				"base.all",
				"base.any",
			},
		},
		{
			name: "with-single-level",
			base: "base",
			topo: []string{"a", "b"},
			id:   "",
			wantResult: []string{
				"base",
				"base.all",
				"base.all.a",
				"base.all.b",
				"base.any",
				"base.any.a",
				"base.any.b",
			},
		},
		{
			name: "with-multi-level",
			base: "base",
			topo: []string{"a.x.y", "b"},
			id:   "",
			wantResult: []string{
				"base",
				"base.all",
				"base.all.a",
				"base.all.a.x",
				"base.all.a.x.y",
				"base.all.b",
				"base.any",
				"base.any.a",
				"base.any.a.x",
				"base.any.a.x.y",
				"base.any.b",
			},
		},
		{
			name: "with-multi-level-id",
			base: "base",
			topo: []string{"a.x.y", "b"},
			id:   "x123",
			wantResult: []string{
				"base",
				"base.all",
				"base.all.a",
				"base.all.a.x",
				"base.all.a.x.y",
				"base.all.b",
				"base.any",
				"base.any.a",
				"base.any.a.x",
				"base.any.a.x.y",
				"base.any.b",
				"base.id.x123",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult := ExpandTopology(tt.base, tt.topo, tt.id)
			assert.Equal(t, tt.wantResult, gotResult.FullNames())
		})
	}
}
