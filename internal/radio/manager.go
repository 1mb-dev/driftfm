package radio

import (
	"sync"

	"github.com/1mb-dev/driftfm/internal/inventory"
)

// Manager manages radios for all moods
type Manager struct {
	repo   *inventory.Repository
	radios map[string]*Radio
	mu     sync.RWMutex
}

// NewManager creates a new radio manager
func NewManager(repo *inventory.Repository) *Manager {
	return &Manager{
		repo:   repo,
		radios: make(map[string]*Radio),
	}
}

// GetRadio returns the radio for a mood (creates if needed)
func (m *Manager) GetRadio(mood string) *Radio {
	m.mu.RLock()
	radio, exists := m.radios[mood]
	m.mu.RUnlock()

	if exists {
		return radio
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if radio, exists = m.radios[mood]; exists {
		return radio
	}

	radio = NewRadio(m.repo, mood)
	m.radios[mood] = radio
	return radio
}

// GetPlaylist returns the playlist for a mood
func (m *Manager) GetPlaylist(mood string, instrumentalOnly bool) ([]*inventory.Track, error) {
	radio := m.GetRadio(mood)
	return radio.GetPlaylist(instrumentalOnly)
}

// RecordPlay records a play for the mood's radio
func (m *Manager) RecordPlay(mood string, trackID int64) {
	radio := m.GetRadio(mood)
	radio.RecordPlay(trackID)
}
