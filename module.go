package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ranakdinesh/spur-messaging/adapters/http"
	"github.com/ranakdinesh/spur-messaging/adapters/postgres"
	"github.com/ranakdinesh/spur-messaging/adapters/providers/email"
	"github.com/ranakdinesh/spur-messaging/adapters/providers/sms"
	"github.com/ranakdinesh/spur-messaging/adapters/providers/whatsapp"
	whatsappmeta "github.com/ranakdinesh/spur-messaging/adapters/providers/whatsapp/meta"
	"github.com/ranakdinesh/spur-messaging/adapters/queue"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/core/services"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
	"github.com/ranakdinesh/spur-messaging/worker"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Config holds messaging-specific configuration.
type Config struct {
	AppEnv string

	EncryptionKey    []byte // AES-256-GCM key for encrypting provider credentials
	WebhookBaseURL   string // e.g. "https://api.example.com/messaging/webhook"
	DefaultRateLimit int    // messages per second per tenant (default: 10)
	WorkerCount      int    // number of concurrent send workers (default: 5)

	// Email provider — set via MESSAGING_EMAIL_PROVIDER env var
	EmailProvider    string
	EmailAPIKey      string
	EmailFromAddress string
	EmailFromName    string
	EmailTrackOpens  bool
	EmailTrackClicks bool

	// SMS provider — set via MESSAGING_SMS_PROVIDER env var
	SMSProvider string
	SMSAPIKey   string
	SMSSenderID string

	// WhatsApp platform-level config
	WhatsAppWebhookVerifyToken     string
	WhatsAppMetaAppID              string
	WhatsAppMetaAppSecret          string
	WhatsAppGraphAPIVersion        string
	WhatsAppEmbeddedSignupConfigID string
}

// Logger interface for platform logging
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// zerologAdapter wraps *zerolog.Logger to satisfy the internal Logger interface.
// This keeps the simple log.Info("msg", "key", val) API inside services/handlers
// while accepting zerolog from the platform.
type zerologAdapter struct {
	zl *zerolog.Logger
}

func (z *zerologAdapter) Debug(msg string, args ...any) { z.event(z.zl.Debug(), msg, args...) }
func (z *zerologAdapter) Info(msg string, args ...any)  { z.event(z.zl.Info(), msg, args...) }
func (z *zerologAdapter) Warn(msg string, args ...any)  { z.event(z.zl.Warn(), msg, args...) }
func (z *zerologAdapter) Error(msg string, args ...any) { z.event(z.zl.Error(), msg, args...) }

func (z *zerologAdapter) event(e *zerolog.Event, msg string, args ...any) {
	for i := 0; i+1 < len(args); i += 2 {
		if key, ok := args[i].(string); ok {
			e = e.Interface(key, args[i+1])
		}
	}
	e.Msg(msg)
}

// Options is passed by app.go when wiring the module
type Options struct {
	DB    *pgxpool.Pool
	Log   *zerolog.Logger
	Cfg   Config
	Redis *goredis.Client
}

// Services exposes service interfaces for cross-module use
type Services struct {
	MessageService        ports.MessageService
	MessagingGateway      ports.MessagingGateway
	TemplateService       ports.TemplateService
	CampaignService       ports.CampaignService
	ContactService        ports.ContactService
	EmailSender           ports.EmailSender
	EmailTemplateService  ports.EmailTemplateService
	EmailAnalyticsService ports.EmailAnalyticsService
	UnsubscribeService    ports.UnsubscribeService
	SuppressionService    ports.SuppressionService
	ConversationService   ports.ConversationService
	TenantWebhookService  ports.TenantWebhookService
	BillingService        ports.BillingService
	WhatsAppOnboarding    ports.WhatsAppOnboardingService
}

// Module is the messaging module instance
type Module struct {
	Services       *Services
	WebhookHandler *http.WebhookHandler
	handlers       struct {
		message       *http.MessageHandler
		template      *http.TemplateHandler
		emailTemplate *http.EmailTemplateHandler
		campaign      *http.CampaignHandler
		contact       *http.ContactHandler
		conversation  *http.ConversationHandler
		webhook       *http.TenantWebhookHandler
		billing       *http.BillingHandler
		segment       *http.SegmentHandler
		provider      *http.ProviderHandler
		unsubscribe   *http.UnsubscribeHandler
		suppression   *http.SuppressionHandler
		analytics     *http.AnalyticsHandler
		waOnboarding  *http.WhatsAppOnboardingHandler
	}
	rateLimiter *http.RateLimiter
}

// New creates and wires the messaging module.
func New(ctx context.Context, opt Options) (*Module, error) {
	// 1. Run migrations
	log := &zerologAdapter{zl: opt.Log}
	log.Info("Running messaging module migrations...")

	// 2. Create repository implementations
	store := postgres.NewStore(opt.DB)

	// 3. Wrap store in adapters to satisfy generic interfaces
	msgRepo := store // store already implements MessageRepository generically
	tmplRepo := templateRepoAdapter{store}
	contactRepo := contactRepoAdapter{store}
	campaignRepo := campaignRepoAdapter{store}
	providerConfigRepo := providerConfigRepoAdapter{store}
	segmentRepo := segmentRepoAdapter{store}
	emailTemplateRepo := emailTemplateRepoAdapter{store}
	emailEventRepo := emailEventRepoAdapter{store}
	unsubscribeRepo := unsubscribeRepoAdapter{store}
	suppressionRepo := suppressionRepoAdapter{store}

	// 4. Create Redis queue
	if opt.Redis == nil {
		return nil, fmt.Errorf("messaging: Redis client is required")
	}
	msgQueue := queue.NewRedisQueueFromClient(opt.Redis)

	// 5. Create provider registry and register providers
	providerRegistry := services.NewProviderRegistry(providerConfigRepo)
	providerRegistry.RegisterWithName("meta_cloud", whatsapp.NewWhatsAppProvider())
	providerRegistry.RegisterWithName("sendgrid", email.NewSendGridProvider())
	providerRegistry.RegisterWithName("mailgun", email.NewMailgunProvider())
	providerRegistry.RegisterWithName("postmark", email.NewPostmarkProvider())
	providerRegistry.RegisterWithName("smtp", email.NewSMTPProvider())
	providerRegistry.RegisterWithName("dev_email", email.NewDevEmailProvider())
	providerRegistry.RegisterWithName("msg91", sms.NewMSG91Provider())
	providerRegistry.RegisterWithName("twilio", sms.NewTwilioProvider())
	emailProvider := resolveEmailProvider(opt.Cfg, log)
	providerRegistry.SetDefaultProvider("email", emailProvider)
	providerRegistry.SetDefaultProvider("sms", opt.Cfg.SMSProvider)
	providerRegistry.SetDefaultProvider("whatsapp", "meta_cloud")
	if cfg := defaultEmailProviderConfig(opt.Cfg); cfg != nil {
		providerRegistry.SetDefaultConfig(domain.ChannelEmail, cfg)
	}
	if cfg := defaultSMSProviderConfig(opt.Cfg); cfg != nil {
		providerRegistry.SetDefaultConfig(domain.ChannelSMS, cfg)
	}

	// 6. Create service implementations
	suppressionSvc := services.NewSuppressionService(suppressionRepo)
	unsubscribeSvc := services.NewUnsubscribeService(unsubscribeRepo)
	contactSvc := services.NewContactService(contactRepo)
	messageSvc := services.NewMessageService(msgRepo, store, contactRepo, tmplRepo, msgQueue, suppressionSvc, unsubscribeSvc, emailTemplateRepo, providerRegistry, services.Config{
		EncryptionKey:              opt.Cfg.EncryptionKey,
		WebhookBaseURL:             opt.Cfg.WebhookBaseURL,
		DefaultRateLimit:           opt.Cfg.DefaultRateLimit,
		WorkerCount:                opt.Cfg.WorkerCount,
		EmailProvider:              emailProvider,
		EmailAPIKey:                opt.Cfg.EmailAPIKey,
		EmailFromAddress:           opt.Cfg.EmailFromAddress,
		EmailFromName:              opt.Cfg.EmailFromName,
		EmailTrackOpens:            opt.Cfg.EmailTrackOpens,
		EmailTrackClicks:           opt.Cfg.EmailTrackClicks,
		SMSProvider:                opt.Cfg.SMSProvider,
		SMSAPIKey:                  opt.Cfg.SMSAPIKey,
		SMSSenderID:                opt.Cfg.SMSSenderID,
		WhatsAppWebhookVerifyToken: opt.Cfg.WhatsAppWebhookVerifyToken,
		WhatsAppMetaAppID:          opt.Cfg.WhatsAppMetaAppID,
	})
	messagingGateway := services.NewMessagingGateway(messageSvc)
	templateSvc := services.NewTemplateService(tmplRepo, campaignRepo, providerRegistry)
	campaignSvc := services.NewCampaignService(campaignRepo, tmplRepo, segmentRepo, msgQueue, suppressionSvc, unsubscribeSvc, contactRepo)
	emailTemplateSvc := services.NewEmailTemplateService(emailTemplateRepo)
	emailAnalyticsSvc := services.NewEmailAnalyticsService(emailEventRepo)
	emailSenderSvc := services.NewEmailSender(messageSvc)
	conversationSvc := services.NewConversationService(store)
	tenantWebhookSvc := services.NewTenantWebhookService(store, nil, log)
	billingSvc := services.NewBillingService(store)
	webhookSvc := services.NewWebhookService(msgRepo, store, contactSvc, emailEventRepo, suppressionSvc, unsubscribeSvc, providerRegistry, providerConfigRepo, tenantWebhookSvc, log)
	graphVersion := opt.Cfg.WhatsAppGraphAPIVersion
	if graphVersion == "" {
		graphVersion = "v23.0"
	}
	whatsAppMetaClient := whatsappmeta.NewClient(whatsappmeta.WithAPIVersion(graphVersion))
	whatsAppOnboardingSvc := services.NewWhatsAppOnboardingService(
		store,
		providerConfigRepo,
		whatsAppMetaOnboardingClient{client: whatsAppMetaClient, appID: opt.Cfg.WhatsAppMetaAppID, appSecret: opt.Cfg.WhatsAppMetaAppSecret},
		credentialCodec{key: opt.Cfg.EncryptionKey},
	)

	// 7. Create handlers
	m := &Module{
		Services: &Services{
			MessageService:        messageSvc,
			MessagingGateway:      messagingGateway,
			TemplateService:       templateSvc,
			CampaignService:       campaignSvc,
			ContactService:        contactSvc,
			EmailSender:           emailSenderSvc,
			EmailTemplateService:  emailTemplateSvc,
			EmailAnalyticsService: emailAnalyticsSvc,
			UnsubscribeService:    unsubscribeSvc,
			SuppressionService:    suppressionSvc,
			ConversationService:   conversationSvc,
			TenantWebhookService:  tenantWebhookSvc,
			BillingService:        billingSvc,
			WhatsAppOnboarding:    whatsAppOnboardingSvc,
		},
		WebhookHandler: http.NewWebhookHandler(webhookSvc, http.WebhookConfig{
			WhatsAppWebhookVerifyToken: opt.Cfg.WhatsAppWebhookVerifyToken,
		}, log),
	}

	m.handlers.message = http.NewMessageHandler(messageSvc)
	m.handlers.template = http.NewTemplateHandler(templateSvc)
	m.handlers.emailTemplate = http.NewEmailTemplateHandler(emailTemplateSvc)
	m.handlers.campaign = http.NewCampaignHandler(campaignSvc)
	m.handlers.contact = http.NewContactHandler(contactSvc)
	m.handlers.conversation = http.NewConversationHandler(conversationSvc)
	m.handlers.webhook = http.NewTenantWebhookHandler(tenantWebhookSvc)
	m.handlers.billing = http.NewBillingHandler(billingSvc)
	m.handlers.segment = http.NewSegmentHandler(segmentServiceAdapter{store})
	m.handlers.provider = http.NewProviderHandler(providerConfigRepo, messageSvc)
	m.handlers.unsubscribe = http.NewUnsubscribeHandler(unsubscribeSvc)
	m.handlers.suppression = http.NewSuppressionHandler(suppressionSvc)
	m.handlers.analytics = http.NewAnalyticsHandler(messageSvc, emailAnalyticsSvc, campaignSvc)
	m.handlers.waOnboarding = http.NewWhatsAppOnboardingHandler(whatsAppOnboardingSvc, http.WhatsAppOnboardingConfig{
		MetaAppID:       opt.Cfg.WhatsAppMetaAppID,
		ConfigID:        opt.Cfg.WhatsAppEmbeddedSignupConfigID,
		CallbackURL:     strings.TrimRight(opt.Cfg.WebhookBaseURL, "/") + "/messaging/whatsapp/onboarding/callback",
		GraphAPIVersion: graphVersion,
	})

	rateLimit := opt.Cfg.DefaultRateLimit
	if rateLimit <= 0 {
		rateLimit = 10
	}
	m.rateLimiter = http.NewRateLimiter(rateLimit, time.Second)

	// 8. Start worker goroutines
	sender := worker.NewSender(msgQueue, msgRepo, campaignRepo, providerConfigRepo, providerRegistry, billingSvc)
	go func() {
		if err := sender.Start(ctx); err != nil {
			log.Error("Sender worker stopped with error", "error", err)
		}
	}()

	campaignExecutor := worker.NewCampaignExecutor(campaignRepo, contactRepo, segmentRepo, tmplRepo, suppressionSvc, unsubscribeSvc, msgRepo, msgQueue)
	go campaignExecutor.Start(ctx)

	templateSync := worker.NewTemplateSync(tmplRepo, providerRegistry)
	go templateSync.Start(ctx)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := tenantWebhookSvc.ProcessDueDeliveries(ctx, 100); err != nil {
					log.Warn("tenant webhook retry worker failed", "error", err)
				}
			}
		}
	}()

	return m, nil
}

func defaultEmailProviderConfig(cfg Config) *domain.ProviderConfig {
	emailProvider := resolveEmailProvider(cfg, nil)
	if emailProvider == "" {
		return nil
	}
	creds := domain.EmailCredentials{APIKey: cfg.EmailAPIKey}
	if emailProvider == domain.ProviderPostmark {
		creds.ServerToken = cfg.EmailAPIKey
	}
	raw, _ := json.Marshal(creds)
	return &domain.ProviderConfig{
		Channel:     domain.ChannelEmail,
		Provider:    emailProvider,
		Credentials: raw,
		IsActive:    true,
		FromEmail:   cfg.EmailFromAddress,
		FromName:    cfg.EmailFromName,
		TrackOpens:  &cfg.EmailTrackOpens,
		TrackClicks: &cfg.EmailTrackClicks,
	}
}

func resolveEmailProvider(cfg Config, log Logger) string {
	provider := strings.TrimSpace(cfg.EmailProvider)
	if provider == "" {
		provider = domain.ProviderSendGrid
	}
	if provider == domain.ProviderDevEmail || emailProviderConfigured(provider, cfg) {
		return provider
	}
	env := strings.ToLower(strings.TrimSpace(cfg.AppEnv))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	}
	if env == "" || env == "development" || env == "dev" || env == "local" || env == "test" {
		if log != nil {
			log.Warn("Email provider credentials missing; using development email outbox", "provider", provider)
		}
		return domain.ProviderDevEmail
	}
	return provider
}

func emailProviderConfigured(provider string, cfg Config) bool {
	switch provider {
	case domain.ProviderSendGrid, domain.ProviderPostmark:
		return strings.TrimSpace(cfg.EmailAPIKey) != ""
	case domain.ProviderMailgun:
		return strings.TrimSpace(cfg.EmailAPIKey) != "" && strings.TrimSpace(os.Getenv("MAILGUN_DOMAIN")) != ""
	case domain.ProviderSMTP:
		return strings.TrimSpace(os.Getenv("MESSAGING_SMTP_HOST")) != ""
	default:
		return true
	}
}

func defaultSMSProviderConfig(cfg Config) *domain.ProviderConfig {
	if cfg.SMSProvider == "" && cfg.SMSAPIKey == "" {
		return nil
	}
	creds := domain.SMSCredentials{}
	switch cfg.SMSProvider {
	case domain.ProviderTwilio:
		creds.AccountSID = cfg.SMSAPIKey
	case domain.ProviderMSG91:
		creds.AuthKey = cfg.SMSAPIKey
		creds.SenderID = cfg.SMSSenderID
	default:
		return nil
	}
	raw, _ := json.Marshal(creds)
	return &domain.ProviderConfig{
		Channel:     domain.ChannelSMS,
		Provider:    cfg.SMSProvider,
		Credentials: raw,
		IsActive:    true,
		SMSSenderID: cfg.SMSSenderID,
	}
}

// RegisterRoutes mounts AUTHENTICATED messaging routes on the chi router.
func (m *Module) RegisterRoutes(r chi.Router) {
	// Identity middleware authenticates the token before this bridge copies
	// tenant/user/permission claims into messaging's auth context.
	r.Use(authctx.IdentityJWTBridge("spur_sso"))

	// Apply rate limiting middleware
	r.Use(m.rateLimiter.Middleware)

	http.RegisterRoutes(r,
		m.handlers.message,
		m.handlers.template,
		m.handlers.emailTemplate,
		m.handlers.campaign,
		m.handlers.contact,
		m.handlers.conversation,
		m.handlers.webhook,
		m.handlers.billing,
		m.handlers.segment,
		m.handlers.provider,
		m.handlers.waOnboarding,
		m.handlers.unsubscribe,
		m.handlers.suppression,
		m.handlers.analytics,
	)
}
