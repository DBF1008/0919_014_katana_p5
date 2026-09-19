package crawler

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarkActionUnique(t *testing.T) {
	c := &Crawler{uniqueActions: make(map[string]struct{})}

	require.True(t, c.markActionUnique("hash-a"), "first insert should be unique")
	require.False(t, c.markActionUnique("hash-a"), "second insert of same hash should not be unique")
	require.True(t, c.markActionUnique("hash-b"))
	require.Len(t, c.uniqueActions, 2)
}

// TestMarkActionUnique_Concurrent hammers markActionUnique from many
// goroutines. Run with -race: before uniqueActionsMu existed this triggered
// "concurrent map writes" on c.uniqueActions.
func TestMarkActionUnique_Concurrent(t *testing.T) {
	c := &Crawler{uniqueActions: make(map[string]struct{})}

	const workers = 16
	const insertsPerWorker = 100

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < insertsPerWorker; i++ {
				// Shared hashes across workers to exercise the dedup path,
				// plus unique hashes to exercise the insert path.
				c.markActionUnique(fmt.Sprintf("shared-%d", i))
				c.markActionUnique(fmt.Sprintf("w%d-%d", w, i))
			}
		}(w)
	}
	wg.Wait()

	require.Len(t, c.uniqueActions, insertsPerWorker+workers*insertsPerWorker)
}
