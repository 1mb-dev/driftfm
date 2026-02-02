package radio

import (
	"math/rand"
	"sync"
	"time"

	"github.com/1mb-dev/driftfm/internal/inventory"
)

// DefaultMaxRecent is the number of recently played tracks to remember
// for avoiding repetition in playlist generation
const DefaultMaxRecent = 3

// Radio manages playlist generation for a mood
type Radio struct {
	repo           *inventory.Repository
	mood           string
	recentlyPlayed []int64
	maxRecent      int
	mu             sync.Mutex
	rng            *rand.Rand
}

// NewRadio creates a new radio for a mood
func NewRadio(repo *inventory.Repository, mood string) *Radio {
	return &Radio{
		repo:           repo,
		mood:           mood,
		recentlyPlayed: make([]int64, 0),
		maxRecent:      DefaultMaxRecent,
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetPlaylist returns a shuffled playlist for the mood.
// Recently played tracks are pushed to the end of the playlist.
func (r *Radio) GetPlaylist(instrumentalOnly bool) ([]*inventory.Track, error) {
	tracks, err := r.repo.GetByMood(r.mood, instrumentalOnly)
	if err != nil {
		return nil, err
	}

	if len(tracks) == 0 {
		return tracks, nil
	}

	// Make a copy to avoid modifying the original
	shuffled := make([]*inventory.Track, len(tracks))
	copy(shuffled, tracks)

	r.mu.Lock()
	r.shuffleWithRecencyLocked(shuffled)
	r.mu.Unlock()

	return shuffled, nil
}

// shuffleWithRecencyLocked shuffles tracks, pushing recently played to the end.
// Caller must hold r.mu.
func (r *Radio) shuffleWithRecencyLocked(tracks []*inventory.Track) {
	recentSet := make(map[int64]bool)
	for _, id := range r.recentlyPlayed {
		recentSet[id] = true
	}

	// Partition: non-recent first, recent last
	nonRecent := make([]*inventory.Track, 0, len(tracks))
	recent := make([]*inventory.Track, 0)

	for _, track := range tracks {
		if recentSet[track.ID] {
			recent = append(recent, track)
		} else {
			nonRecent = append(nonRecent, track)
		}
	}

	// Fisher-Yates shuffle for non-recent tracks
	for i := len(nonRecent) - 1; i > 0; i-- {
		j := r.rng.Intn(i + 1)
		nonRecent[i], nonRecent[j] = nonRecent[j], nonRecent[i]
	}

	// Rebuild tracks slice: non-recent first, recent last
	idx := 0
	for _, track := range nonRecent {
		tracks[idx] = track
		idx++
	}
	for _, track := range recent {
		tracks[idx] = track
		idx++
	}
}

// RecordPlay records that a track was played
func (r *Radio) RecordPlay(trackID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already in recent list
	for _, id := range r.recentlyPlayed {
		if id == trackID {
			return
		}
	}

	r.recentlyPlayed = append(r.recentlyPlayed, trackID)

	// Trim to max size
	if len(r.recentlyPlayed) > r.maxRecent {
		r.recentlyPlayed = r.recentlyPlayed[1:]
	}
}
