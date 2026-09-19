package graph

import (
	"fmt"
	"sync"
	"testing"

	"github.com/projectdiscovery/katana/pkg/engine/headless/types"
	"github.com/stretchr/testify/require"
)

func TestAddPageState_DuplicateIsIdempotent(t *testing.T) {
	g := NewCrawlGraph()
	state := types.PageState{UniqueID: "a", URL: "http://example.com", Depth: 0}

	require.NoError(t, g.AddPageState(state))
	// Adding the same vertex again must not fail (ErrVertexAlreadyExists is swallowed)
	require.NoError(t, g.AddPageState(state))

	got, err := g.GetPageState("a")
	require.NoError(t, err)
	require.Equal(t, "http://example.com", got.URL)
}

func TestAddPageState_WithEdge(t *testing.T) {
	g := NewCrawlGraph()
	require.NoError(t, g.AddPageState(types.PageState{UniqueID: "root", URL: "about:blank"}))
	require.NoError(t, g.AddPageState(types.PageState{
		UniqueID:         "child",
		URL:              "http://example.com",
		OriginID:         "root",
		Depth:            1,
		NavigationAction: &types.Action{Type: types.ActionTypeLoadURL, Input: "http://example.com", Depth: 0},
	}))

	actions, err := g.ShortestPath("root", "child")
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.Equal(t, "http://example.com", actions[0].Input)
}

// TestCrawlGraph_ConcurrentAccess exercises readers and writers in parallel.
// Run with -race: before the mutex was added this reliably triggered
// "concurrent map read and map write" inside dominikbraun/graph.
func TestCrawlGraph_ConcurrentAccess(t *testing.T) {
	g := NewCrawlGraph()
	require.NoError(t, g.AddPageState(types.PageState{UniqueID: "root", URL: "about:blank"}))

	const workers = 8
	const statesPerWorker = 25

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < statesPerWorker; i++ {
				id := fmt.Sprintf("state-%d-%d", w, i)
				_ = g.AddPageState(types.PageState{
					UniqueID:         id,
					URL:              "http://example.com/" + id,
					OriginID:         "root",
					Depth:            1,
					NavigationAction: &types.Action{Type: types.ActionTypeLoadURL, Input: id, Depth: 0},
				})
				_, _ = g.GetPageState(id)
				_, _ = g.ShortestPath("root", id)
				_ = g.GetVertices()
			}
		}(w)
	}
	wg.Wait()

	require.Len(t, g.GetVertices(), workers*statesPerWorker+1)
}
