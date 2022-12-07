package barrel

import (
	"path/filepath"
	"time"

	"github.com/mr-karan/barreldb/internal/datafile"
)

// ExamineFileSize checks for file size at a periodic interval.
// It examines the file size of the active db file and marks it as stale
// if the file size exceeds the configured size.
func (b *Barrel) ExamineFileSize(evalInterval time.Duration) {
	var (
		evalTicker = time.NewTicker(evalInterval).C
	)
	for range evalTicker {
		if err := b.rotateDF(); err != nil {
			b.lo.Error("error rotating db file", "error", err)
		}
	}
}

// RunCompaction runs cleanup process to compact the keys and cleanup
// dead/expired keys at a periodic interval. This helps to save disk space
// and merge old inactive db files in a single file. It also generates a hints file
// which helps in caching all the keys during a cold start.
func (b *Barrel) RunCompaction(evalInterval time.Duration) {
	var (
		evalTicker = time.NewTicker(evalInterval).C
	)
	for range evalTicker {
		if err := b.cleanupExpired(); err != nil {
			b.lo.Error("error removing expired keys", "error", err)
		}
		// if err := b.merge(); err != nil {
		// 	b.lo.Error("error merging old files", "error", err)
		// }
		if err := b.generateHints(); err != nil {
			b.lo.Error("error generating hints file", "error", err)
		}
	}
}

// rotateDF checks if the active file size has crossed the threshold
// of max allowed file size. If it has, it replaces the open file descriptors
// pointing to that file with a new file and adds the current file to list of
// stale files.
func (b *Barrel) rotateDF() error {
	b.Lock()
	defer b.Unlock()

	size, err := b.df.Size()
	if err != nil {
		return err
	}

	// If the file is below the threshold of max size, do no action.
	b.lo.Debug("checking if db file has exceeded max_size", "current_size", size, "max_size", b.opts.MaxActiveFileSize)
	if size < b.opts.MaxActiveFileSize {
		return nil
	}

	oldID := b.df.ID()

	// Add this datafile to list of stale files.
	b.stale[oldID] = b.df

	// Create a new datafile.
	df, err := datafile.New(b.opts.Dir, oldID+1)
	if err != nil {
		return err
	}

	// Replace with a new instance of datafile.
	b.df = df

	return nil
}

// generateHints encodes the contents of the in-memory hashtable
// as `gob` and writes the data to a hints file.
func (b *Barrel) generateHints() error {
	b.Lock()
	defer b.Unlock()

	path := filepath.Join(b.opts.Dir, HINTS_FILE)
	if err := b.keydir.Encode(path); err != nil {
		return err
	}

	return nil
}

// cleanupExpired removes the expired keys.
func (b *Barrel) cleanupExpired() error {
	b.Lock()
	defer b.Unlock()

	// Iterate over all keys and delete all keys which are expired.
	for k := range b.keydir {
		record, err := b.get(k)
		if err != nil {
			b.lo.Error("error fetching key", "key", k, "error", err)
			continue
		}
		if record.isExpired() {
			b.lo.Debug("deleting key since it's expired", "key", k)
			// Delete the key.
			if err := b.delete(k); err != nil {
				b.lo.Error("error deleting key", "key", k, "error", err)
				continue
			}
		}
	}

	return nil
}
