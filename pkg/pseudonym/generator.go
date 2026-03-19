package pseudonym

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Generator interface {
	Generate() string
}

type Random struct {
	mu         sync.Mutex
	rnd        *rand.Rand
	adjectives []string
	nouns      []string
}

func NewRandom(seed int64) *Random {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return &Random{
		rnd:        rand.New(rand.NewSource(seed)),
		adjectives: []string{"silent", "lunar", "amber", "gentle", "rapid", "wild", "echo", "mellow", "solar", "velvet"},
		nouns:      []string{"fox", "owl", "river", "cloud", "stone", "comet", "pine", "wave", "ember", "leaf"},
	}
}

func (r *Random) Generate() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	adj := r.adjectives[r.rnd.Intn(len(r.adjectives))]
	noun := r.nouns[r.rnd.Intn(len(r.nouns))]
	num := 100 + r.rnd.Intn(900)

	return fmt.Sprintf("%s-%s-%d", adj, noun, num)
}
