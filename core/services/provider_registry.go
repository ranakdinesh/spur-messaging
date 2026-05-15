package services

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type ProviderRegistry struct {
	configRepo       ports.ProviderConfigRepository
	Providers        map[domain.Channel]map[string]ports.Provider
	defaultProviders map[domain.Channel]string
	defaultConfigs   map[domain.Channel]*domain.ProviderConfig
}

func NewProviderRegistry(configRepo ports.ProviderConfigRepository) *ProviderRegistry {
	return &ProviderRegistry{
		configRepo:       configRepo,
		Providers:        make(map[domain.Channel]map[string]ports.Provider),
		defaultProviders: make(map[domain.Channel]string),
		defaultConfigs:   make(map[domain.Channel]*domain.ProviderConfig),
	}
}

func (r *ProviderRegistry) Register(provider ports.Provider) {
	channel := provider.Channel()
	if r.Providers[channel] == nil {
		r.Providers[channel] = make(map[string]ports.Provider)
	}
	// We might need a better way to get provider name, but for now let's assume we can map it
	// Actually, the ports.Provider interface doesn't have a Name() method.
	// We'll use the channel as key for now or handle specifically.
}

// RegisterWithName registers a provider with a specific name (e.g. "meta_cloud", "sendgrid")
func (r *ProviderRegistry) RegisterWithName(name string, provider ports.Provider) {
	channel := provider.Channel()
	if r.Providers[channel] == nil {
		r.Providers[channel] = make(map[string]ports.Provider)
	}
	r.Providers[channel][name] = provider
}

func (r *ProviderRegistry) SetDefaultProvider(channel domain.Channel, name string) {
	if name == "" {
		return
	}
	r.defaultProviders[channel] = name
}

func (r *ProviderRegistry) SetDefaultConfig(channel domain.Channel, cfg *domain.ProviderConfig) {
	if cfg == nil {
		return
	}
	r.defaultConfigs[channel] = cfg
}

func (r *ProviderRegistry) GetProvider(ctx context.Context, tenantID uuid.UUID, channel domain.Channel) (ports.Provider, *domain.ProviderConfig, error) {
	// 1. Check tenant's provider_configs
	cfg, err := r.configRepo.GetByChannel(ctx, tenantID, channel)
	if err == nil && cfg != nil && cfg.IsActive {
		p, ok := r.Providers[channel][cfg.Provider]
		if ok {
			return p, cfg, nil
		}
	}

	// 2. Fallback to platform default from env
	var defaultProviderName string
	if r.defaultProviders != nil {
		defaultProviderName = r.defaultProviders[channel]
	}
	switch channel {
	case domain.ChannelEmail:
		if defaultProviderName == "" {
			defaultProviderName = os.Getenv("MESSAGING_EMAIL_PROVIDER")
		}
	case domain.ChannelSMS:
		if defaultProviderName == "" {
			defaultProviderName = os.Getenv("MESSAGING_SMS_PROVIDER")
		}
	case domain.ChannelWhatsApp:
		if defaultProviderName == "" {
			defaultProviderName = "meta_cloud" // WhatsApp usually only has one in this setup
		}
	}

	if defaultProviderName == "" {
		return nil, nil, fmt.Errorf("no provider configured for channel %s", channel)
	}

	p, ok := r.Providers[channel][defaultProviderName]
	if !ok {
		return nil, nil, fmt.Errorf("provider %s not registered for channel %s", defaultProviderName, channel)
	}

	// For platform default, we create a dummy config or one loaded from env
	// The provider implementation should know how to use env if cfg is "minimal"
	// But AGENTS.md says we should resolve tenant config -> fallback to platform env.

	// Create a platform config
	platformCfg := &domain.ProviderConfig{
		TenantID: tenantID,
		Channel:  channel,
		Provider: defaultProviderName,
		IsActive: true,
	}
	if r.defaultConfigs != nil && r.defaultConfigs[channel] != nil {
		cfgCopy := *r.defaultConfigs[channel]
		cfgCopy.ID = uuid.Nil
		cfgCopy.TenantID = tenantID
		cfgCopy.Channel = channel
		cfgCopy.Provider = defaultProviderName
		cfgCopy.IsActive = true
		platformCfg = &cfgCopy
	}

	return p, platformCfg, nil
}
