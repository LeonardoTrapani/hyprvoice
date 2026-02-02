package config

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Manager struct {
	mu      sync.RWMutex
	config  *Config
	watcher *fsnotify.Watcher
	wg      sync.WaitGroup

	onConfigReload func()

	// Debouncer for config reloads
	debounceTimer *time.Timer
	debounceMutex sync.Mutex
	debounceDelay time.Duration

	// legacy tracks if config is in legacy format (needs onboarding)
	legacy bool
}

func NewManager() (*Manager, error) {
	log.Printf("Config manager: initializing configuration system...")

	config, legacy, err := LoadOrLegacy()
	if err != nil {
		log.Printf("Config manager: failed to load initial configuration: %v", err)
		return nil, err
	}

	if legacy {
		log.Printf("Config manager: legacy config detected, daemon will prompt for onboarding")
	} else {
		log.Printf("Config manager: validating initial configuration...")
		if err := config.Validate(); err != nil {
			log.Printf("Config manager: validation warning: %v", err)
		}
	}

	m := &Manager{
		config:        config,
		debounceDelay: 500 * time.Millisecond, // 500ms debounce delay
		legacy:        legacy,
	}

	log.Printf("Config manager: initialization completed successfully")
	return m, nil
}

func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	configCopy := *m.config
	return &configCopy
}

// IsLegacy returns true if the config is in legacy format and needs onboarding
func (m *Manager) IsLegacy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.legacy
}

func (m *Manager) StartWatching(ctx context.Context) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	m.watcher = watcher

	configDir := filepath.Dir(configPath)
	err = watcher.Add(configDir)
	if err != nil {
		watcher.Close()
		return err
	}

	m.wg.Add(1)
	go m.watchLoop(ctx, configPath)

	log.Printf("Config manager: watching %s for changes", configPath)
	return nil
}

func (m *Manager) Stop() {
	if m.watcher != nil {
		m.watcher.Close()
	}

	// Clean up debounce timer
	m.debounceMutex.Lock()
	if m.debounceTimer != nil {
		m.debounceTimer.Stop()
	}
	m.debounceMutex.Unlock()

	m.wg.Wait()
}

func (m *Manager) watchLoop(ctx context.Context, configPath string) {
	defer m.wg.Done()
	configFileName := filepath.Base(configPath)

	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			// Filter for our config file only
			eventFileName := filepath.Base(event.Name)
			if eventFileName != configFileName {
				continue
			}

			// Only react to Write and Create events (ignore Chmod, Remove, etc.)
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("Config manager: file change detected: %s. Debouncing reload...", event.Name)
				m.debounceReloadConfig()
			}

		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Config watcher error: %v", err)

		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) reloadConfig() {
	log.Printf("Config manager: starting configuration reload...")

	newConfig, legacy, err := LoadOrLegacy()
	if err != nil {
		log.Printf("Config manager: failed to reload config: %v", err)
		return
	}

	if legacy {
		log.Printf("Config manager: config still in legacy format, skipping reload")
		return
	}

	log.Printf("Config manager: validating new configuration...")
	if err := newConfig.Validate(); err != nil {
		log.Printf("Config manager: invalid config after reload: %v", err)
		return
	}

	m.mu.Lock()
	m.config = newConfig
	m.legacy = false // clear legacy flag on successful reload
	onConfigReload := m.onConfigReload
	m.mu.Unlock()

	if onConfigReload != nil {
		onConfigReload()
	}

	log.Printf("Config manager: configuration successfully reloaded")
}

func (m *Manager) SetOnConfigReload(onConfigReload func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onConfigReload = onConfigReload
}

// debounceReloadConfig implements debouncing to prevent duplicate reloads
func (m *Manager) debounceReloadConfig() {
	m.debounceMutex.Lock()
	defer m.debounceMutex.Unlock()

	// Cancel existing timer if it exists
	if m.debounceTimer != nil {
		m.debounceTimer.Stop()
	}

	// Create new timer with debounce delay
	m.debounceTimer = time.AfterFunc(m.debounceDelay, func() {
		log.Printf("Config manager: debounce period expired, reloading config...")
		m.reloadConfig()
	})
}
