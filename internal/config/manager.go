package config

import (
	"context"
	"log"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Manager struct {
	mu      sync.RWMutex
	config  *Config
	watcher *fsnotify.Watcher
	wg      sync.WaitGroup
}

func NewManager() (*Manager, error) {
	log.Printf("Config manager: initializing configuration system...")

	config, err := Load()
	if err != nil {
		log.Printf("Config manager: failed to load initial configuration: %v", err)
		return nil, err
	}

	log.Printf("Config manager: validating initial configuration...")
	if err := config.Validate(); err != nil {
		log.Printf("Config manager: validation warning: %v", err)
	}

	m := &Manager{
		config: config,
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
				log.Printf("Config manager: file change detected: %s. Reloading config...", event.Name)
				m.reloadConfig()
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

	newConfig, err := Load()
	if err != nil {
		log.Printf("Config manager: failed to reload config: %v", err)
		return
	}

	log.Printf("Config manager: validating new configuration...")
	if err := newConfig.Validate(); err != nil {
		log.Printf("Config manager: invalid config after reload: %v", err)
		return
	}

	m.mu.Lock()
	m.config = newConfig
	m.mu.Unlock()

	log.Printf("Config manager: configuration successfully reloaded")
}
