// Package offline provides a persistent local store for OData responses,
// enabling read operations when the network is unavailable.
package offline

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ErrNotFound is returned by Get when the path has no cached entry.
var ErrNotFound = errors.New("offline: entry not found")

const indexFile = "_index.json"

// Store persists OData entity set responses to the local filesystem.
// Each cached path is stored as a JSON file named by the sha256 of the path.
// An index file tracks the mapping from hash to original path.
type Store struct {
	dir string
	mu  sync.Mutex
}

// NewStore creates a Store that persists data to the given directory.
// The directory is created if it does not exist.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("offline: create store dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Set stores the serialised JSON of an OData response for the given entity set path.
// The path is used as the cache key (e.g. "/Customers" or "/Customers(1)").
func (s *Store) Set(path string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := hashPath(path)
	file := filepath.Join(s.dir, hash+".json")
	if err := writeFileAtomic(file, data, 0o600); err != nil {
		return fmt.Errorf("offline: write entry: %w", err)
	}
	return s.updateIndex(hash, path)
}

// Get retrieves the cached response for the given entity set path.
// Returns ErrNotFound if no cached entry exists.
func (s *Store) Get(path string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := hashPath(path)
	file := filepath.Join(s.dir, hash+".json")
	data, err := os.ReadFile(file) //nolint:gosec // file path is sha256-derived, no traversal risk
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("offline: read entry: %w", err)
	}
	return data, nil
}

// Delete removes a cached entry.
func (s *Store) Delete(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := hashPath(path)
	file := filepath.Join(s.dir, hash+".json")
	if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("offline: delete entry: %w", err)
	}
	return s.removeFromIndex(hash)
}

// Keys returns all cached paths.
func (s *Store) Keys() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, err := s.readIndex()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(idx))
	for _, path := range idx {
		keys = append(keys, path)
	}
	return keys, nil
}

// Clear removes all cached entries.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, err := s.readIndex()
	if err != nil {
		return err
	}
	for hash := range idx {
		file := filepath.Join(s.dir, hash+".json")
		if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("offline: clear entry: %w", err)
		}
	}
	return s.writeIndex(map[string]string{})
}

// hashPath returns the hex-encoded sha256 of path.
func hashPath(path string) string {
	sum := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", sum)
}

// readIndex reads the index file from disk. Returns an empty map if the file
// does not exist yet.
func (s *Store) readIndex() (map[string]string, error) {
	file := filepath.Join(s.dir, indexFile)
	data, err := os.ReadFile(file) //nolint:gosec // path is constructed from a fixed constant
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("offline: read index: %w", err)
	}
	var idx map[string]string
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("offline: parse index: %w", err)
	}
	return idx, nil
}

// writeIndex persists the index map to disk atomically using a temp-file rename.
func (s *Store) writeIndex(idx map[string]string) error {
	data, err := json.Marshal(idx)
	if err != nil {
		return fmt.Errorf("offline: marshal index: %w", err)
	}
	file := filepath.Join(s.dir, indexFile)
	return writeFileAtomic(file, data, 0o600)
}

// writeFileAtomic writes data to path atomically by writing to a sibling temp
// file first and then renaming it. On POSIX systems rename(2) is atomic within
// the same filesystem, so readers never see a partially-written file.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// updateIndex adds or updates the hash→path mapping in the index.
func (s *Store) updateIndex(hash, path string) error {
	idx, err := s.readIndex()
	if err != nil {
		return err
	}
	idx[hash] = path
	return s.writeIndex(idx)
}

// removeFromIndex removes the hash entry from the index.
func (s *Store) removeFromIndex(hash string) error {
	idx, err := s.readIndex()
	if err != nil {
		return err
	}
	delete(idx, hash)
	return s.writeIndex(idx)
}
