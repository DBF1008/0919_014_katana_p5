package graph

import (
	"fmt"
	"sync"
	"testing"

	"github.com/projectdiscovery/katana/pkg/engine/headless/types"
	"github.com/stretchr/testify/require"
)

func TestCrawlGraphAddPageState(t *testing.T) {
	cg := NewCrawlGraph()

	err := cg.AddPageState(types.PageState{UniqueID: "root", URL: "https://example.com", IsRoot: true})
	require.NoError(t, err)

	// Adding the same vertex again must be a no-op, not an error.
	err = cg.AddPageState(types.PageState{UniqueID: "root", URL: "https://example.com"})
	require.NoError(t, err)

	action := &types.Action{Type: types.ActionTypeLoadURL, Input: "https://example.com/page", Depth: 1}
	err = cg.AddPageState(types.PageState{
		UniqueID:         "child",
		OriginID:         "root",
		URL:              "https://example.com/page",
		Depth:            1,
		NavigationAction: action,
	})
	require.NoError(t, err)

	state, err := cg.GetPageState("child")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/page", state.URL)

	path, err := cg.ShortestPath("root", "child")
	require.NoError(t, err)
	require.Len(t, path, 1)
	require.Equal(t, action, path[0])
}

// TestCrawlGraphConcurrentAccess exercises the graph under concurrent
// readers and writers. Run with -race to detect data races in the
// underlying dominikbraun/graph store.
func TestCrawlGraphConcurrentAccess(t *testing.T) {
	cg := NewCrawlGraph()
	require.NoError(t, cg.AddPageState(types.PageState{UniqueID: "root", URL: "https://example.com", IsRoot: true}))

	const workers = 8
	const statesPerWorker = 25

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	record := func(err error) {
		if err == nil {
			return
		}
		mu.Lock()
		errs = append(errs, err)
		mu.Unlock()
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < statesPerWorker; i++ {
				id := fmt.Sprintf("state-%d-%d", worker, i)
				action := &types.Action{Type: types.ActionTypeLoadURL, Input: "https://example.com/" + id, Depth: 1}
				record(cg.AddPageState(types.PageState{
					UniqueID:         id,
					OriginID:         "root",
					URL:              "https://example.com/" + id,
					Depth:            1,
					NavigationAction: action,
				}))

				// Duplicate adds race with other writers and must stay no-ops.
				record(cg.AddPageState(types.PageState{UniqueID: id, OriginID: "root"}))
				record(cg.AddEdge("root", id, action))

				_, err := cg.GetPageState(id)
				record(err)
				if _, err := cg.ShortestPath("root", id); err != nil {
					record(err)
				}
				_ = cg.GetVertices()
			}
		}(w)
	}
	wg.Wait()

	require.Empty(t, errs)
	require.Len(t, cg.GetVertices(), workers*statesPerWorker+1)
}
