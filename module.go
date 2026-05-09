package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ranakdinesh/spur-messaging/adapters/http"
	"github.com/ranakdinesh/spur-messaging/adapters/postgres"
	"github.com/ranakdinesh/spur-messaging/adapters/providers/email"
	"github.com/ranakdinesh/spur-messaging/adapters/providers/sms"
	"github.com/ranakdinesh/spur-messaging/adapters/providers/whatsapp"
	"github.com/ranakdinesh/spur-messaging/adapters/queue"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/core/services"
	"github.com/ranakdinesh/spur-messaging/worker"
)

// Config holds messaging-specific configuration.
type Config struct {
	EncryptionKey    []byte // AES-256-GCM key for encrypting provider credentials
	WebhookBaseURL   string // e.g. "https://api.example.com/messaging/webhook"
	DefaultRateLimit int    // messages per second per tenant (default: 10)
	RedisURL         string // Redis connection for message queue
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
	WhatsAppWebhookVerifyToken string
	WhatsAppMetaAppID          string
}

// Logger interface for platform logging
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// Options is passed by app.go when wiring the module
type Options struct {
	DB  *pgxpool.Pool
	Log Logger
	Cfg Config
}

// Services exposes service interfaces for cross-module use
type Services struct {
	MessageService        ports.MessageService
	TemplateService       ports.TemplateService
	CampaignService       ports.CampaignService
	ContactService        ports.ContactService
	EmailSender           ports.EmailSender
	EmailTemplateService  ports.EmailTemplateService
	EmailAnalyticsService ports.EmailAnalyticsService
	UnsubscribeService    ports.UnsubscribeService
	SuppressionService    ports.SuppressionService
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
		segment       *http.SegmentHandler
		provider      *http.ProviderHandler
		unsubscribe   *http.UnsubscribeHandler
		suppression   *http.SuppressionHandler
		analytics     *http.AnalyticsHandler
	}
	rateLimiter *http.RateLimiter
}

// New creates and wires the messaging module.
func New(ctx context.Context, opt Options) (*Module, error) {
	// 1. Run migrations
	opt.Log.Info("Running messaging module migrations...")

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
	msgQueue, err := queue.NewRedisQueue(opt.Cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis queue: %w", err)
	}

	// 5. Create provider registry and register providers
	providerRegistry := services.NewProviderRegistry(providerConfigRepo)
	providerRegistry.RegisterWithName("meta_cloud", whatsapp.NewWhatsAppProvider())
	providerRegistry.RegisterWithName("sendgrid", email.NewSendGridProvider())
	providerRegistry.RegisterWithName("mailgun", email.NewMailgunProvider())
	providerRegistry.RegisterWithName("postmark", email.NewPostmarkProvider())
	providerRegistry.RegisterWithName("msg91", sms.NewMSG91Provider())
	providerRegistry.RegisterWithName("twilio", sms.NewTwilioProvider())

	// 6. Create service implementations
	suppressionSvc := services.NewSuppressionService(suppressionRepo)
	unsubscribeSvc := services.NewUnsubscribeService(unsubscribeRepo)
	contactSvc := services.NewContactService(contactRepo)
	messageSvc := services.NewMessageService(msgRepo, contactRepo, tmplRepo, msgQueue, suppressionSvc, unsubscribeSvc, emailTemplateRepo, providerRegistry, services.Config(opt.Cfg))
	templateSvc := services.NewTemplateService(tmplRepo, campaignRepo, providerRegistry)
	campaignSvc := services.NewCampaignService(campaignRepo, tmplRepo, segmentRepo, msgQueue, suppressionSvc, unsubscribeSvc, contactRepo)
	emailTemplateSvc := services.NewEmailTemplateService(emailTemplateRepo)
	emailAnalyticsSvc := services.NewEmailAnalyticsService(emailEventRepo)
	emailSenderSvc := services.NewEmailSender(messageSvc)
	webhookSvc := services.NewWebhookService(msgRepo, emailEventRepo, suppressionSvc, unsubscribeSvc, providerRegistry, providerConfigRepo, opt.Log)

	// 7. Create handlers
	m := &Module{
		Services: &Services{
			MessageService:        messageSvc,
			TemplateService:       templateSvc,
			CampaignService:       campaignSvc,
			ContactService:        contactSvc,
			EmailSender:           emailSenderSvc,
			EmailTemplateService:  emailTemplateSvc,
			EmailAnalyticsService: emailAnalyticsSvc,
			UnsubscribeService:    unsubscribeSvc,
			SuppressionService:    suppressionSvc,
		},
		WebhookHandler: http.NewWebhookHandler(webhookSvc, http.WebhookConfig{
			WhatsAppWebhookVerifyToken: opt.Cfg.WhatsAppWebhookVerifyToken,
		}, opt.Log),
	}

	m.handlers.message = http.NewMessageHandler(messageSvc)
	m.handlers.template = http.NewTemplateHandler(templateSvc)
	m.handlers.emailTemplate = http.NewEmailTemplateHandler(emailTemplateSvc)
	m.handlers.campaign = http.NewCampaignHandler(campaignSvc)
	m.handlers.contact = http.NewContactHandler(contactSvc)
	m.handlers.segment = http.NewSegmentHandler(segmentServiceAdapter{store})
	m.handlers.provider = http.NewProviderHandler(providerConfigRepo, messageSvc)
	m.handlers.unsubscribe = http.NewUnsubscribeHandler(unsubscribeSvc)
	m.handlers.suppression = http.NewSuppressionHandler(suppressionSvc)
	m.handlers.analytics = http.NewAnalyticsHandler(messageSvc, emailAnalyticsSvc, campaignSvc)

	rateLimit := opt.Cfg.DefaultRateLimit
	if rateLimit <= 0 {
		rateLimit = 10
	}
	m.rateLimiter = http.NewRateLimiter(rateLimit, time.Second)

	// 8. Start worker goroutines
	sender := worker.NewSender(msgQueue, msgRepo, campaignRepo, providerConfigRepo, providerRegistry)
	go func() {
		if err := sender.Start(ctx); err != nil {
			opt.Log.Error("Sender worker stopped with error", "error", err)
		}
	}()

	campaignExecutor := worker.NewCampaignExecutor(campaignRepo, contactRepo, segmentRepo, tmplRepo, suppressionSvc, unsubscribeSvc, msgRepo, msgQueue)
	go campaignExecutor.Start(ctx)

	templateSync := worker.NewTemplateSync(tmplRepo, providerRegistry)
	go templateSync.Start(ctx)

	return m, nil
}

// RegisterRoutes mounts AUTHENTICATED messaging routes on the chi router.
func (m *Module) RegisterRoutes(r chi.Router) {
	// Apply rate limiting middleware
	r.Use(m.rateLimiter.Middleware)

	http.RegisterRoutes(r,
		m.handlers.message,
		m.handlers.template,
		m.handlers.emailTemplate,
		m.handlers.campaign,
		m.handlers.contact,
		m.handlers.segment,
		m.handlers.provider,
		m.handlers.unsubscribe,
		m.handlers.suppression,
		m.handlers.analytics,
	)
}
