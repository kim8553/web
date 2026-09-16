package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/local/9yin-go-server/internal/role"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const facultyStoreVersion = 1

type facultyBookSnapshot struct {
	ConfigID string
	Level    int32
	Fill     int32
	Total    int32
	Power    int32
	MaxPower int32
}
type facultyProgressSnapshot struct {
	CurNeiGong     string
	Faculty        int32
	FacultyState   int32
	FacultyStyle   int32
	FacultyName    string
	FillSpeed      int32
	LastAdvanceUTC time.Time
	Books          []facultyBookSnapshot
	QGLevels       map[string]int32
}
type facultyStoreFile struct {
	Version int                                `json:"version"`
	Roles   map[string]facultyProgressSnapshot `json:"roles"`
}
type facultyProgressStore struct {
	mu    sync.Mutex
	path  string
	roles map[string]facultyProgressSnapshot
}

func openFacultyProgressStore() (*facultyProgressStore, error) {
	return openFacultyProgressStoreAt(filepath.Join(runtimeProjectRoot, "data", "faculty.json"))
}
func openFacultyProgressStoreAt(path string) (*facultyProgressStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("faculty store: empty path")
	}
	path = filepath.Clean(path)
	store := &facultyProgressStore{path: path, roles: make(map[string]facultyProgressSnapshot)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read faculty store: %w", err)
	}
	var document facultyStoreFile
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("decode faculty store: %w", err)
	}
	if document.Version != facultyStoreVersion {
		return nil, fmt.Errorf("decode faculty store: unsupported version %d", document.Version)
	}
	if document.Roles != nil {
		store.roles = document.Roles
	}
	return store, nil
}
func (store *facultyProgressStore) Load(roleID role.RoleID) (facultyProgressSnapshot, bool) {
	if store == nil || roleID == 0 {
		return facultyProgressSnapshot{}, false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	value, exists := store.roles[strconv.FormatUint(uint64(roleID), 10)]
	return cloneFacultyProgress(value), exists
}
func (store *facultyProgressStore) Save(roleID role.RoleID, value facultyProgressSnapshot) error {
	if store == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("faculty store: zero role id")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := strconv.FormatUint(uint64(roleID), 10)
	previous, existed := store.roles[key]
	store.roles[key] = cloneFacultyProgress(value)
	if err := store.persistLocked(); err != nil {
		if existed {
			store.roles[key] = previous
		} else {
			delete(store.roles, key)
		}
		return err
	}
	return nil
}
func (store *facultyProgressStore) persistLocked() error {
	document := facultyStoreFile{Version: facultyStoreVersion, Roles: store.roles}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode faculty store: %w", err)
	}
	data = append(data, '\n')
	directory := filepath.Dir(store.path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create faculty store directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".faculty-*.tmp")
	if err != nil {
		return fmt.Errorf("create faculty store temporary: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		if temporary != nil {
			_ = temporary.Close()
		}
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set faculty store temporary permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write faculty store temporary: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync faculty store temporary: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close faculty store temporary: %w", err)
	}
	if err := os.Rename(temporaryPath, store.path); err != nil {
		return fmt.Errorf("replace faculty store: %w", err)
	}
	committed = true
	return nil
}
func cloneFacultyProgress(value facultyProgressSnapshot) facultyProgressSnapshot {
	value.Books = append([]facultyBookSnapshot(nil), value.Books...)
	return value
}
