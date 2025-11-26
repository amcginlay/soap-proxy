package storage

import (
    "bufio"
    "encoding/json"
    "os"
    "sync"

    "soap-proxy/internal/trace"
)

// FileTraceStore persists traces to a local file (JSONL) and keeps the last N in memory.
type FileTraceStore struct {
    mu      sync.RWMutex
    maxLen  int
    items   []trace.Entry
    idIndex map[string]int
    file    *os.File
    path    string
}

// NewFileTraceStore opens/creates the trace file and loads existing entries.
func NewFileTraceStore(path string, maxLen int) (*FileTraceStore, error) {
    f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o600)
    if err != nil {
        return nil, err
    }

    store := &FileTraceStore{
        maxLen:  maxLen,
        file:    f,
        path:    path,
        idIndex: make(map[string]int),
    }

    if err := store.loadFromFile(); err != nil {
        return nil, err
    }
    return store, nil
}

func (s *FileTraceStore) loadFromFile() error {
    f, err := os.Open(s.path)
    if err != nil {
        return err
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    buf := make([]byte, 0, 1024*1024)
    scanner.Buffer(buf, 10*1024*1024)

    var entries []trace.Entry
    for scanner.Scan() {
        line := scanner.Bytes()
        var e trace.Entry
        if err := json.Unmarshal(line, &e); err != nil {
            continue
        }
        entries = append(entries, e)
    }

    if len(entries) > s.maxLen {
        entries = entries[len(entries)-s.maxLen:]
    }

    s.items = entries
    s.idIndex = make(map[string]int, len(s.items))
    for i, e := range s.items {
        s.idIndex[e.ID] = i
    }

    return scanner.Err()
}

// Add appends a trace entry to the file and updates the in-memory ring buffer.
func (s *FileTraceStore) Add(e trace.Entry) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    b, err := json.Marshal(e)
    if err != nil {
        return err
    }
    if _, err := s.file.Write(append(b, '\n')); err != nil {
        return err
    }
    _ = s.file.Sync()

    s.items = append(s.items, e)
    if len(s.items) > s.maxLen {
        s.items = s.items[len(s.items)-s.maxLen:]
    }

    s.idIndex = make(map[string]int, len(s.items))
    for i, it := range s.items {
        s.idIndex[it.ID] = i
    }

    return nil
}

// List returns a copy of the currently buffered traces.
func (s *FileTraceStore) List() []trace.Entry {
    s.mu.RLock()
    defer s.mu.RUnlock()
    out := make([]trace.Entry, len(s.items))
    copy(out, s.items)
    return out
}

// Get finds a trace by ID in the buffered set.
func (s *FileTraceStore) Get(id string) (trace.Entry, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    idx, ok := s.idIndex[id]
    if !ok || idx < 0 || idx >= len(s.items) {
        return trace.Entry{}, false
    }
    return s.items[idx], true
}

// Close closes the underlying file.
func (s *FileTraceStore) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    if s.file != nil {
        return s.file.Close()
    }
    return nil
}
