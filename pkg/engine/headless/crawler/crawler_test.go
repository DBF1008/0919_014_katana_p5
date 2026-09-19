package crawler

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarkActionUnique(t *testing.T) {
	c := &Crawler{uniqueActions: make(map[string]struct{})}

	require.True(t, c.markActionUnique("hash-a"))
	require.False(t, c.markActionUnique("hash-a"))
	require.True(t, c.markActionUnique("hash-b"))
	require.Len(t, c.uniqueActions, 2)
}

// TestMarkActionUniqueConcurrent verifies that concurrent deduplication of
// discovered navigations is race-free and each hash is accepted exactly once.
// Run with -race.
func TestMarkActionUniqueConcurrent(t *testing.T) {
	c := &Crawler{uniqueActions: make(map[string]struct{})}

	const workers = 16
	const hashesPerWorker = 100

	var (
		wg       sync.WaitGroup
		accepted sync.Map
	)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < hashesPerWorker; i++ {
				// Every worker sees the same set of hashes, mimicking
				// concurrent RequestCallback-triggered discoveries.
				hash := fmt.Sprintf("hash-%d", i)
				if c.markActionUnique(hash) {
					_, alreadyAccepted := accepted.LoadOrStore(hash, struct{}{})
					require.False(t, alreadyAccepted, "hash accepted twice: %s", hash)
				}
			}
		}(w)
	}
	wg.Wait()

	require.Len(t, c.uniqueActions, hashesPerWorker)
}
