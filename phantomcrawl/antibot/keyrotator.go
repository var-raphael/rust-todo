package antibot

import (
	"fmt"
	"math/rand"
	"sync"
)

type KeyRotator struct {
	keys     []string
	strategy string
	index    int
	mu       sync.Mutex
}

func NewKeyRotator(keys []string, strategy string) *KeyRotator {
	return &KeyRotator{
		keys:     keys,
		strategy: strategy,
	}
}

func (k *KeyRotator) Next() (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if len(k.keys) == 0 {
		return "", fmt.Errorf("no keys available")
	}

	switch k.strategy {
	case "round_robin":
		key := k.keys[k.index%len(k.keys)]
		k.index++
		return key, nil
	case "fallback":
		return k.keys[0], nil
	default:
		// random
		return k.keys[rand.Intn(len(k.keys))], nil
	}
}

func (k *KeyRotator) HasKeys() bool {
	return len(k.keys) > 0
}