package models

import "context"

type ConfigManager interface {
	Init() error
	GetConfig() *Config
	Load() error
	Update(key string, value any) error
	Watch(ctx context.Context) (<-chan *Config, error)
}
