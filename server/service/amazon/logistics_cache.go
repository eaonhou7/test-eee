package amazon

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sync"
	"time"
)

type logisticsCache struct {
	mu      sync.RWMutex
	entries map[string]logisticsWorkbookData
}

func newLogisticsCache() *logisticsCache {
	return &logisticsCache{
		entries: map[string]logisticsWorkbookData{},
	}
}

func (c *logisticsCache) get(key string) (logisticsWorkbookData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	return entry, ok
}

func (c *logisticsCache) set(key string, value logisticsWorkbookData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = value
}

func logisticsDefaultCacheKey(provider, path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return provider + ":missing:" + path
	}
	return provider + ":path:" + path + ":" + info.ModTime().UTC().Format(time.RFC3339Nano)
}

func logisticsUploadCacheKey(provider string, data []byte) string {
	sum := sha256.Sum256(data)
	return provider + ":upload:" + hex.EncodeToString(sum[:])
}
