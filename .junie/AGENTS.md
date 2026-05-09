# AGENTS.md — spur-messaging

> **This is the single source of truth for AI coding agents working on this module.**
> Read this file COMPLETELY before writing any code. Do not skip sections.
> Do not invent patterns — follow what is documented here exactly.

---

## 0. MODULE IDENTITY

| Field | Value |
|---|---|
| Repo | `github.com/ranakdinesh/spur-messaging` |
| Go version | 1.25+ |
| Module type | Spur backend module (pure Go + OpenAPI) |
| Schema | `messaging` (NOT public) |
| Dev port | Backend 9090 (shared with other modules via chi router) |
| DB port | PostgreSQL 5433 |
| Dependencies | pgxpool, chi, sqlc, crypto, net/http |
| No frontend | This repo contains ZERO frontend code. No TSX, no JS, no CSS. |

---

## 0A. SPUR.JSON MANIFEST

This file lives at the repo root as `spur.json`. The Spur CLI uses it to:
- Wire the module into `app.go` with correct imports and init code
- Register required/optional env vars in `.env`
- Generate the `App` struct field and value

Create this file exactly:

```json
{
  "name": "messaging",
  "version": "1.0.0",
  "description": "WhatsApp, Email, SMS — campaigns, templates, contacts, analytics",
  "go_package": "github.com/ranakdinesh/spur-messaging",
  "private": true,
  "status": "alpha",
  "required_env": [
    "MESSAGING_ENCRYPTION_KEY",
    "MESSAGING_WEBHOOK_BASE_URL",
    "MESSAGING_REDIS_URL"
  ],
  "optional_env": [
    "MESSAGING_DEFAULT_RATE_LIMIT",
    "MESSAGING_WORKER_COUNT",
    "WHATSAPP_WEBHOOK_VERIFY_TOKEN",
    "WHATSAPP_META_APP_ID",
    "MESSAGING_EMAIL_PROVIDER",
    "MESSAGING_EMAIL_FROM_ADDRESS",
    "MESSAGING_EMAIL_FROM_NAME",
    "MESSAGING_EMAIL_TRACK_OPENS",
    "MESSAGING_EMAIL_TRACK_CLICKS",
    "MESSAGING_SMS_PROVIDER",
    "MESSAGING_SMS_SENDER_ID",
    "SENDGRID_API_KEY",
    "MAILGUN_API_KEY",
    "MAILGUN_DOMAIN",
    "POSTMARK_SERVER_TOKEN",
    "MSG91_AUTH_KEY",
    "TWILIO_ACCOUNT_SID",
    "TWILIO_AUTH_TOKEN",
    "TWILIO_FROM_NUMBER"
  ],
  "has_sqlc": true,
  "config_struct": "MessagingConfig",
  "config_env_prefix": "MESSAGING_",
  "app_field": "Messaging *messaging.Module",
  "app_value": "Messaging: messagingModule,",
  "wire_code": {
    "import": "\"encoding/hex\"\n\t\"strconv\"\n\tmessaging \"github.com/ranakdinesh/spur-messaging\"",
    "init": "encKey, err := hex.DecodeString(cfg.MessagingEncryptionKey)\nif err != nil { return nil, fmt.Errorf(\"MESSAGING_ENCRYPTION_KEY must be 64-char hex: %w\", err) }\nif len(encKey) != 32 { return nil, fmt.Errorf(\"MESSAGING_ENCRYPTION_KEY must be 32 bytes (64 hex chars)\") }\nrateLimit := 10\nif cfg.MessagingDefaultRateLimit != \"\" { rateLimit, _ = strconv.Atoi(cfg.MessagingDefaultRateLimit) }\nworkerCount := 5\nif cfg.MessagingWorkerCount != \"\" { workerCount, _ = strconv.Atoi(cfg.MessagingWorkerCount) }\nemailProvider := cfg.MessagingEmailProvider\nif emailProvider == \"\" { emailProvider = \"sendgrid\" }\nvar emailAPIKey string\nswitch emailProvider {\ncase \"sendgrid\": emailAPIKey = cfg.SendGridAPIKey\ncase \"mailgun\": emailAPIKey = cfg.MailgunAPIKey\ncase \"postmark\": emailAPIKey = cfg.PostmarkServerToken\n}\nsmsProvider := cfg.MessagingSMSProvider\nif smsProvider == \"\" { smsProvider = \"msg91\" }\nvar smsAPIKey string\nswitch smsProvider {\ncase \"msg91\": smsAPIKey = cfg.MSG91AuthKey\ncase \"twilio\": smsAPIKey = cfg.TwilioAccountSID\n}\nemailFrom := cfg.MessagingEmailFromAddr\nif emailFrom == \"\" { emailFrom = \"noreply@example.com\" }\nemailName := cfg.MessagingEmailFromName\nif emailName == \"\" { emailName = \"Spur\" }\nsmsSender := cfg.MessagingSMSSenderID\nif smsSender == \"\" { smsSender = \"SPUR\" }\nmessagingCfg := messaging.Config{EncryptionKey: encKey, WebhookBaseURL: cfg.MessagingWebhookBaseURL, DefaultRateLimit: rateLimit, RedisURL: cfg.MessagingRedisURL, WorkerCount: workerCount, EmailProvider: emailProvider, EmailAPIKey: emailAPIKey, EmailFromAddress: emailFrom, EmailFromName: emailName, EmailTrackOpens: cfg.MessagingEmailTrackOpens != \"false\", EmailTrackClicks: cfg.MessagingEmailTrackClicks != \"false\", SMSProvider: smsProvider, SMSAPIKey: smsAPIKey, SMSSenderID: smsSender, WhatsAppWebhookVerifyToken: cfg.WhatsAppWebhookVerifyToken, WhatsAppMetaAppID: cfg.WhatsAppMetaAppID}\nmessagingLog := infra.Log.Logger()\nmessagingModule, err := messaging.New(ctx, messaging.Options{DB: infra.DB, Log: &messagingLog, Cfg: messagingCfg})\nif err != nil { return nil, fmt.Errorf(\"messaging: %%w\", err) }",
    "routes": "r.Route(\"/messaging\", func(r chi.Router) {\n\tr.Use(identityModule.AuthGuard.Middleware)\n\tr.Use(identityModule.AuthGuard.TenantIsolation)\n\tmessagingModule.RegisterRoutes(r)\n})\nr.Get(\"/messaging/webhook/whatsapp\", messagingModule.WebhookHandler.Verify)\nr.Post(\"/messaging/webhook/whatsapp\", messagingModule.WebhookHandler.Handle)\nr.Post(\"/messaging/webhook/email/sendgrid\", messagingModule.WebhookHandler.HandleSendGrid)\nr.Post(\"/messaging/webhook/email/mailgun\", messagingModule.WebhookHandler.HandleMailgun)\nr.Post(\"/messaging/webhook/email/postmark\", messagingModule.WebhookHandler.HandlePostmark)\nr.Post(\"/messaging/webhook/sms\", messagingModule.WebhookHandler.HandleSMS)\nr.Get(\"/messaging/unsubscribe/{token}\", messagingModule.WebhookHandler.HandleUnsubscribeLink)"
  }
}
```

### Key notes on spur.json:

1. **`wire_code.routes`** — this is the critical part. It shows the CLI exactly how
   to mount routes in `app.go`. Notice:
   - Authenticated routes go inside `r.Route("/messaging", ...)` with identity middleware
   - Webhook routes are mounted OUTSIDE the auth group (no middleware)
   - Unsubscribe link handler is also outside auth (public endpoint)
   - Identity module's `AuthGuard.Middleware` and `TenantIsolation` are applied

2. **`wire_code.init`** — decodes the encryption key from hex, resolves the email/SMS
   provider API keys based on which provider is selected, and creates the Config.

3. **`required_env`** — only 3 vars are truly required. Everything else has defaults.
   - `MESSAGING_ENCRYPTION_KEY`: 64-char hex string (32 bytes for AES-256-GCM)
   - `MESSAGING_WEBHOOK_BASE_URL`: public URL for webhooks (e.g. https://api.citual.com)
   - `MESSAGING_REDIS_URL`: Redis connection string (e.g. redis://localhost:6379/1)

4. **`optional_env`** — provider API keys are optional at platform level because
   tenants can bring their own via `provider_configs` table. But at least one
   email provider key should be set for transactional emails from other modules.

### .env example

```env
# Required
MESSAGING_ENCRYPTION_KEY=a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2
MESSAGING_WEBHOOK_BASE_URL=https://api.citual.com
MESSAGING_REDIS_URL=redis://localhost:6379/1

# WhatsApp (per-tenant credentials go in provider_configs table via API, not here)
# This verify token is platform-level — used during Meta webhook verification challenge
WHATSAPP_WEBHOOK_VERIFY_TOKEN=my_random_verify_token_change_this
WHATSAPP_META_APP_ID=123456789012345

# Email (pick one provider)
MESSAGING_EMAIL_PROVIDER=sendgrid
SENDGRID_API_KEY=SG.xxxx
MESSAGING_EMAIL_FROM_ADDRESS=noreply@citual.com
MESSAGING_EMAIL_FROM_NAME=Citual
MESSAGING_EMAIL_TRACK_OPENS=true
MESSAGING_EMAIL_TRACK_CLICKS=true

# SMS (pick one provider)
MESSAGING_SMS_PROVIDER=msg91
MSG91_AUTH_KEY=xxxx
MESSAGING_SMS_SENDER_ID=CITUAL

# Optional tuning
MESSAGING_DEFAULT_RATE_LIMIT=10
MESSAGING_WORKER_COUNT=5
```

### WhatsApp setup — why it's different from Email/SMS

**Email and SMS** have platform-level API keys in `.env` because you (the platform operator)
have one SendGrid/MSG91 account that all tenants share. Small tenants use your account,
big tenants bring their own keys via `provider_configs`.

**WhatsApp CANNOT work this way.** Each WhatsApp Business number is tied to ONE business.
You cannot share a number across tenants. So WhatsApp is 100% per-tenant — there are NO
platform-level WhatsApp credentials.

The ONLY platform-level WhatsApp config is:
- `WHATSAPP_WEBHOOK_VERIFY_TOKEN` — a random string YOU choose. When Meta verifies your
  webhook endpoint, it sends this token back and your endpoint must echo it. All tenants'
  Meta Apps use the same webhook URL and verify token.
- `WHATSAPP_META_APP_ID` — your Meta App ID for reference/logging.

### WhatsApp tenant onboarding flow (API-driven)

This is the complete flow for getting a tenant's WhatsApp working on your platform.

**Step 1: Tenant creates their Meta setup (they do this, not you)**

```
1. Go to business.facebook.com → create Meta Business Account
2. Go to developers.facebook.com → create a new App (type: Business)
3. Add "WhatsApp" product to the app
4. Go to WhatsApp > Getting Started:
   - Note the "Phone number ID" (auto-generated test number)
   - Note the "WhatsApp Business Account ID" (WABA ID)
5. Go to Business Settings > System Users:
   - Create a system user (Admin role)
   - Generate a permanent access token with these permissions:
     whatsapp_business_management, whatsapp_business_messaging
   - Note the "Access Token"
6. Go to App Settings > Basic:
   - Note the "App Secret" (used for webhook signature verification)
```

**Step 2: Tenant configures webhook in Meta (points to YOUR platform)**

```
1. Go to WhatsApp > Configuration > Webhook
2. Set Callback URL: {MESSAGING_WEBHOOK_BASE_URL}/messaging/webhook/whatsapp
   Example: https://api.citual.com/messaging/webhook/whatsapp
3. Set Verify Token: {WHATSAPP_WEBHOOK_VERIFY_TOKEN} (same value from your .env)
4. Click "Verify and Save" — Meta sends GET to your endpoint with the token
5. Subscribe to these webhook fields:
   - messages (incoming messages)
   - message_template_status_update (template approval changes)
```

**Step 3: Tenant enters credentials into YOUR platform via API**

```bash
# The tenant (or your admin) calls your API to save their WhatsApp credentials
curl -X POST https://api.citual.com/messaging/providers \
  -H "Authorization: Bearer {jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "whatsapp",
    "provider": "meta_cloud",
    "credentials": {
      "access_token": "EAAxxxxxx...",
      "app_secret": "abc123def456..."
    },
    "phone_number_id": "100000000000000",
    "waba_id": "200000000000000",
    "business_id": "300000000000000",
    "display_phone": "+919810914244"
  }'
```

Your platform encrypts the credentials with AES-256-GCM and stores them in `provider_configs`.

**Step 4: Test the connection**

```bash
curl -X POST https://api.citual.com/messaging/providers/{provider_id}/test \
  -H "Authorization: Bearer {jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "+919810914244",
    "message": "Hello from Citual! Your WhatsApp is connected."
  }'
```

**Step 5: Tenant can now send messages**

```bash
# Send a template message
curl -X POST https://api.citual.com/messaging/send \
  -H "Authorization: Bearer {jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "whatsapp",
    "recipient": "+919876543210",
    "message_type": "template",
    "template_name": "hello_world",
    "template_language": "en_US"
  }'
```

### How webhook routing works (one URL, many tenants)

All tenants configure the SAME webhook URL: `https://api.citual.com/messaging/webhook/whatsapp`

When Meta sends a webhook, the payload includes the WABA ID:

```json
{
  "entry": [{
    "id": "WABA_ID_HERE",
    "changes": [{ ... }]
  }]
}
```

Your webhook handler:
1. Extracts `entry[0].id` (the WABA ID)
2. Looks up `provider_configs WHERE waba_id = ? AND channel = 'whatsapp'`
3. Finds the tenant that owns this WABA
4. Loads that tenant's `app_secret` from encrypted credentials
5. Verifies the webhook signature using that app_secret
6. Processes the event (status update, incoming message) for that tenant

This is why `waba_id` has an index and a lookup method in the repository.

---

## 1. MODULE CONTRACT

This module composes into the spur-template binary via chi router.
It MUST expose the following in `module.go`:

```go
package messaging

import (
    "context"
    "github.com/go-chi/chi/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    // internal imports only
)

// Config holds messaging-specific configuration.
// Email and SMS provider selection is done via env vars — each tenant can
// override at the provider_configs level, but the platform default is set here.
type Config struct {
    EncryptionKey       []byte // AES-256-GCM key for encrypting provider credentials
    WebhookBaseURL      string // e.g. "https://api.example.com/messaging/webhook"
    DefaultRateLimit    int    // messages per second per tenant (default: 10)
    RedisURL            string // Redis connection for message queue
    WorkerCount         int    // number of concurrent send workers (default: 5)

    // Email provider — set via MESSAGING_EMAIL_PROVIDER env var
    // Valid values: "sendgrid", "mailgun", "postmark"
    // The corresponding API key is set via:
    //   SENDGRID_API_KEY, MAILGUN_API_KEY, POSTMARK_API_KEY
    EmailProvider       string // "sendgrid" | "mailgun" | "postmark"
    EmailAPIKey         string // API key for the selected email provider
    EmailFromAddress    string // Default FROM address (e.g. "noreply@citual.com")
    EmailFromName       string // Default FROM name (e.g. "Citual")
    EmailTrackOpens     bool   // Enable open tracking pixel (default: true)
    EmailTrackClicks    bool   // Enable click tracking via redirect (default: true)

    // SMS provider — set via MESSAGING_SMS_PROVIDER env var
    // Valid values: "msg91", "twilio"
    SMSProvider         string // "msg91" | "twilio"
    SMSAPIKey           string // API key for the selected SMS provider
    SMSSenderID         string // Sender ID for SMS (e.g. "CITUAL")

    // WhatsApp platform-level config
    // NOTE: WhatsApp credentials (access_token, phone_number_id, waba_id, app_secret)
    // are PER-TENANT — stored encrypted in provider_configs table, NOT in env vars.
    // Only the webhook verify token is platform-level.
    WhatsAppWebhookVerifyToken string // Random string for Meta webhook verification challenge
    WhatsAppMetaAppID          string // Your Meta App ID (for documentation/reference)
}

// Options is passed by app.go when wiring the module
type Options struct {
    DB  *pgxpool.Pool
    Log Logger          // use the platform's logger interface
    Cfg Config
    // NOTE: Auth middleware (AuthGuard, RBAC, GUC) is NOT passed here.
    // It is applied EXTERNALLY by app.go on the chi router group.
    // This module only reads auth context via the authctx helper package.
}

// Services exposes service interfaces for cross-module use
type Services struct {
    MessageService  ports.MessageService
    TemplateService ports.TemplateService
    CampaignService ports.CampaignService
    ContactService  ports.ContactService
}

// Module is the messaging module instance
type Module struct {
    Services       *Services
    WebhookHandler *handlers.WebhookHandler // exposed so app.go can mount it outside auth
}

// New creates and wires the messaging module. Returns error, NEVER calls log.Fatal or os.Exit.
func New(ctx context.Context, opt Options) (*Module, error) {
    // 1. Run migrations
    // 2. Create repository implementations (postgres adapter)
    // 3. Create service implementations (inject repos via ports)
    // 4. Create handlers (inject services via port interfaces)
    // 5. Return &Module{Services: ..., WebhookHandler: ...}
}

// RegisterRoutes mounts AUTHENTICATED messaging routes on the chi router.
// All routes are under /messaging/* prefix.
// Auth middleware is applied EXTERNALLY by app.go — NOT inside this function.
// Webhook routes are NOT mounted here — they are mounted separately by app.go
// outside the auth middleware group.
func (m *Module) RegisterRoutes(r chi.Router) {
    // Mount routes — see Section 7 for full route map
    // DOES NOT mount /messaging/webhook/* — those go outside auth
}
```

### Rules for module.go:
- `New()` MUST return `(*Module, error)` — never panic, never os.Exit
- All dependencies injected via `Options` — no global state
- Services field uses PORT INTERFACES, not concrete structs
- RegisterRoutes receives the parent chi.Router and mounts a subrouter
- WebhookHandler is exposed on Module so app.go can mount it outside auth middleware

---

## 1A. AUTH INTEGRATION WITH IDENTITY MODULE

**The messaging module does NOT implement authentication, authorization, RBAC, or
tenant isolation.** All of that is handled by the identity module's middleware,
applied externally by `app.go`.

This module only READS auth context from `context.Context` using a lightweight
helper package.

### 1A.1 How app.go wires auth to messaging routes

```go
// app.go (in spur-template) — THIS IS NOT IN THE MESSAGING REPO
// Shown here so the agent understands the wiring context.

identityModule, _ := identity.New(ctx, identityOpts)
messagingModule, _ := messaging.New(ctx, messagingOpts)

r := chi.NewRouter()

// Identity mounts its own routes (/auth/*, /users/*, /tenants/*)
identityModule.RegisterRoutes(r)

// Messaging routes: wrapped in identity's auth middleware stack
r.Route("/messaging", func(r chi.Router) {
    // Middleware 1: Validates JWT token OR API key from request headers
    //   JWT:     Authorization: Bearer eyJhbG...
    //   API Key: X-API-Key: sk_live_abc123
    // Populates ctx with: tenant_id, user_id, roles, permissions
    r.Use(identityModule.AuthGuard.Middleware)

    // Middleware 2: Sets PostgreSQL GUC variable for RLS enforcement
    //   Executes: SET LOCAL app.tenant_id = '<uuid>'
    //   This makes RLS policies on messaging.* tables work automatically
    r.Use(identityModule.AuthGuard.TenantIsolation)

    // Now mount messaging's authenticated routes
    messagingModule.RegisterRoutes(r)
})

// Webhook routes: OUTSIDE auth middleware — verified by provider HMAC signature
r.Get("/messaging/webhook/whatsapp", messagingModule.WebhookHandler.Verify)
r.Post("/messaging/webhook/whatsapp", messagingModule.WebhookHandler.Handle)
r.Post("/messaging/webhook/sms", messagingModule.WebhookHandler.HandleSMS)
r.Post("/messaging/webhook/email", messagingModule.WebhookHandler.HandleEmail)
```

### 1A.2 Auth context helper package

Create a lightweight `authctx` package that reads values from context.
This is the ONLY auth-related code in the messaging module.

```go
// pkg/authctx/authctx.go
package authctx

import (
    "context"
    "github.com/google/uuid"
)

// Context keys — must match what the identity module sets.
// These are the standard Spur context keys used across all modules.
type ctxKey string

const (
    keyTenantID    ctxKey = "tenant_id"
    keyUserID      ctxKey = "user_id"
    keyRoles       ctxKey = "roles"
    keyPermissions ctxKey = "permissions"
    keyAuthMethod  ctxKey = "auth_method" // "jwt" or "api_key"
)

// TenantID extracts the tenant UUID from context. Panics if missing.
// This is safe because the identity middleware guarantees it is set
// for all authenticated routes. If it panics, it means auth middleware
// was not applied — which is a wiring bug in app.go, not a runtime error.
func TenantID(ctx context.Context) uuid.UUID {
    v, ok := ctx.Value(keyTenantID).(uuid.UUID)
    if !ok {
        panic("authctx: tenant_id missing from context — auth middleware not applied")
    }
    return v
}

// UserID extracts the user UUID from context. Returns uuid.Nil for API key auth
// (API keys are not tied to a specific user, only to a tenant).
func UserID(ctx context.Context) uuid.UUID {
    v, _ := ctx.Value(keyUserID).(uuid.UUID)
    return v
}

// HasPermission checks if the authenticated user/key has a specific permission.
func HasPermission(ctx context.Context, permission string) bool {
    perms, ok := ctx.Value(keyPermissions).([]string)
    if !ok {
        return false
    }
    for _, p := range perms {
        if p == permission || p == "*" { // wildcard = superadmin
            return true
        }
    }
    return false
}

// HasRole checks if the authenticated user has a specific role.
func HasRole(ctx context.Context, role string) bool {
    roles, ok := ctx.Value(keyRoles).([]string)
    if !ok {
        return false
    }
    for _, r := range roles {
        if r == role {
            return true
        }
    }
    return false
}

// AuthMethod returns "jwt" or "api_key" depending on how the request was authenticated.
func AuthMethod(ctx context.Context) string {
    v, _ := ctx.Value(keyAuthMethod).(string)
    return v
}
```

### 1A.3 RBAC permissions for messaging

These permissions are registered in the identity module's RBAC system.
The messaging module checks them in handlers using `authctx.HasPermission()`.

```
PERMISSION STRING                    DESCRIPTION
─────────────────────────────────────────────────────────────────
messaging:providers:read             View provider configurations
messaging:providers:write            Add/update/delete provider configs (WhatsApp creds etc.)
messaging:providers:test             Send a test message via provider

messaging:templates:read             View message templates
messaging:templates:write            Create/update/delete templates
messaging:templates:submit           Submit template for Meta approval

messaging:messages:send              Send individual messages
messaging:messages:send_bulk         Send bulk messages
messaging:messages:read              View message history and status

messaging:contacts:read              View contacts
messaging:contacts:write             Create/update/delete contacts
messaging:contacts:import            Bulk import contacts
messaging:contacts:manage_consent    Change opt-in/opt-out status

messaging:segments:read              View segments
messaging:segments:write             Create/update/delete segments

messaging:campaigns:read             View campaigns
messaging:campaigns:write            Create/update/delete campaigns
messaging:campaigns:execute          Start/pause/resume a campaign

messaging:analytics:read             View analytics and reports
```

### 1A.4 Pre-built roles (created via identity module seed/migration)

| Role | Permissions | Typical User |
|---|---|---|
| `messaging_admin` | All `messaging:*` | Tenant owner, CTO |
| `messaging_marketer` | templates:*, campaigns:*, contacts:read, segments:*, analytics:read | Marketing team |
| `messaging_agent` | messages:send, messages:read, contacts:read | Customer support agent |
| `messaging_viewer` | *:read only (all read permissions) | Reporting, finance, observer |
| `messaging_developer` | providers:*, templates:*, messages:*, contacts:read | Developer integrating via API key |

### 1A.5 Handler pattern with auth (MANDATORY for every handler)

Every handler in the messaging module MUST follow this exact pattern:

```go
// adapters/http/handlers/campaign_handler.go

type CampaignHandler struct {
    campaignService ports.CampaignService  // PORT interface, not concrete struct
}

func NewCampaignHandler(svc ports.CampaignService) *CampaignHandler {
    return &CampaignHandler{campaignService: svc}
}

func (h *CampaignHandler) ExecuteCampaign(w http.ResponseWriter, r *http.Request) {
    // Step 1: Read tenant context (set by identity middleware)
    tenantID := authctx.TenantID(r.Context())

    // Step 2: Check permission (REQUIRED for mutating operations)
    if !authctx.HasPermission(r.Context(), "messaging:campaigns:execute") {
        response.RespondError(w, domain.ErrForbidden)
        return
    }

    // Step 3: Parse and validate request input
    campaignID, err := uuid.Parse(chi.URLParam(r, "id"))
    if err != nil {
        response.RespondValidationError(w, "id", "invalid campaign ID")
        return
    }

    // Step 4: Call service with tenantID (service never sees auth details)
    err = h.campaignService.Execute(r.Context(), tenantID, campaignID)
    if err != nil {
        response.RespondError(w, err)
        return
    }

    // Step 5: Respond
    response.RespondOK(w, map[string]string{"status": "started"})
}
```

**Rules for auth in handlers:**
- READ operations: check `messaging:<resource>:read` permission
- WRITE operations (create/update/delete): check `messaging:<resource>:write` permission
- SPECIAL operations (execute campaign, submit template, import contacts): check their specific permission
- ALWAYS extract tenantID via `authctx.TenantID(ctx)` — never from request body or URL params
- NEVER pass auth details (user_id, roles) into service layer — services only receive tenantID
- Exception: audit logging — you MAY pass userID to services if you need to record who performed an action

### 1A.6 API key access for external tenant systems

Some tenants will consume the messaging API directly from their own systems
(e.g., a tenant's Node.js backend triggering an OTP message on user signup).

They use API keys created through the identity module:

```
POST /auth/api-keys
{
  "name": "Production Backend",
  "permissions": ["messaging:messages:send", "messaging:contacts:read"],
  "expires_at": "2027-01-01T00:00:00Z"
}
→ { "api_key": "sk_live_abc123xyz..." }
```

The tenant uses this key in their system:

```bash
curl -X POST https://api.citual.com/messaging/send \
  -H "X-API-Key: sk_live_abc123xyz..." \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "whatsapp",
    "recipient": "+919876543210",
    "message_type": "template",
    "template_name": "order_confirmation",
    "template_params": { "1": "Rahul", "2": "ORD-9876" }
  }'
```

The identity module's AuthGuard middleware:
1. Sees `X-API-Key` header (no JWT)
2. Looks up the API key → finds tenant_id + permissions
3. Sets the same context values as JWT auth
4. The messaging handler cannot distinguish between JWT and API key access

**This means the messaging module does NOT need any API key handling code.**
Everything is transparent via `authctx.TenantID(ctx)` and `authctx.HasPermission(ctx, ...)`.

### 1A.7 Webhook routes — NO auth, HMAC signature only

Webhook endpoints (`/messaging/webhook/*`) are called by Meta, Twilio, etc.
They are NOT authenticated via identity middleware. Instead:

```go
// adapters/http/handlers/webhook_handler.go

func (h *WebhookHandler) HandleWhatsApp(w http.ResponseWriter, r *http.Request) {
    // Step 1: Read raw body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // Step 2: Verify HMAC signature (Meta signs with app secret)
    // The webhook service finds the right tenant by WABA ID in the payload,
    // loads that tenant's provider config, and verifies the signature.
    err = h.webhookService.HandleWhatsAppWebhook(r.Context(), r.Header, body)
    if err != nil {
        // Log error but ALWAYS return 200 to Meta (or they retry aggressively)
        w.WriteHeader(http.StatusOK)
        return
    }

    w.WriteHeader(http.StatusOK)
}
```

**Tenant routing in webhooks:** The webhook payload contains the WABA ID.
The webhook service looks up `provider_configs` by `waba_id` to find which
tenant this webhook belongs to, then processes accordingly. No auth middleware needed.

---

## 2. DIRECTORY STRUCTURE

```
spur-messaging/
├── module.go                  ← Options, Config, Services, New(), RegisterRoutes()
├── openapi.json               ← OpenAPI 3.1 spec (THE contract for any frontend/client)
├── spur.json                  ← CLI manifest for platform integration (Section 0A)
├── AGENTS.md                  ← This file
├── .env.example               ← Example env vars (copy from Section 0A)
├── go.mod
├── go.sum
│
├── pkg/
│   └── authctx/               ← Auth context reader (reads values set by identity middleware)
│       └── authctx.go         ← TenantID(), UserID(), HasPermission(), HasRole()
│
├── core/
│   ├── domain/                ← Entity structs — NO db tags, NO json tags
│   │   ├── channel.go
│   │   ├── message.go
│   │   ├── template.go
│   │   ├── campaign.go
│   │   ├── contact.go
│   │   ├── conversation.go
│   │   ├── provider_config.go
│   │   ├── segment.go
│   │   └── errors.go          ← Domain error types
│   │
│   ├── ports/                 ← Interfaces ONLY — never concrete types
│   │   ├── repositories.go    ← All repository interfaces in one file
│   │   ├── services.go        ← All service interfaces in one file
│   │   ├── provider.go        ← Provider interface (channel adapters)
│   │   └── queue.go           ← MessageQueue interface
│   │
│   └── services/              ← Business logic — depends ONLY on ports
│       ├── message_service.go
│       ├── template_service.go
│       ├── campaign_service.go
│       ├── contact_service.go
│       ├── webhook_service.go
│       └── provider_registry.go  ← Maps channel → provider implementation
│
├── adapters/
│   ├── postgres/
│   │   ├── store.go           ← Wraps sqlc Querier, implements repository ports
│   │   ├── gen/               ← sqlc generated code (DO NOT EDIT)
│   │   │   ├── db.go
│   │   │   ├── models.go
│   │   │   └── querier.go
│   │   └── mappers.go         ← Convert between sqlc models ↔ domain entities
│   │
│   ├── http/
│   │   ├── routes.go          ← Route registration (called by module.RegisterRoutes)
│   │   ├── middleware.go      ← Rate limiting (auth middleware is EXTERNAL from identity)
│   │   ├── response.go        ← Standard JSON response envelope
│   │   └── handlers/          ← Each handler: authctx.TenantID() → permission check → validate → service → respond
│   │       ├── message_handler.go
│   │       ├── template_handler.go
│   │       ├── campaign_handler.go
│   │       ├── contact_handler.go
│   │       ├── provider_handler.go
│   │       ├── webhook_handler.go     ← NO auth — HMAC signature verification only
│   │       └── analytics_handler.go
│   │
│   ├── providers/
│   │   ├── whatsapp/
│   │   │   ├── client.go      ← Meta Cloud API HTTP client
│   │   │   ├── webhook.go     ← Verify signature + parse events
│   │   │   ├── templates.go   ← Template CRUD on Meta
│   │   │   ├── media.go       ← Upload/download media
│   │   │   └── types.go       ← Meta API request/response structs
│   │   ├── sms/
│   │   │   ├── msg91.go       ← MSG91 adapter (India primary)
│   │   │   ├── twilio.go      ← Twilio adapter (international)
│   │   │   └── webhook.go     ← Delivery receipt parsing for both providers
│   │   └── email/
│   │       ├── sendgrid.go    ← SendGrid v3 API adapter
│   │       ├── mailgun.go     ← Mailgun API adapter
│   │       ├── postmark.go    ← Postmark API adapter
│   │       ├── webhook.go     ← Webhook parsing for all 3 providers (bounce, delivery, open, click, complaint)
│   │       ├── renderer.go    ← HTML email template rendering with variable substitution
│   │       └── types.go       ← Shared email request/response types
│   │
│   ├── queue/
│   │   └── redis_streams.go   ← Redis Streams consumer group
│   │
│   └── crypto/
│       └── encrypt.go         ← AES-256-GCM encrypt/decrypt for provider credentials
│
├── worker/
│   ├── sender.go              ← Main send worker: dequeue → provider.Send → update status
│   ├── campaign_executor.go   ← Fan-out campaign to individual messages
│   └── template_sync.go       ← Periodic template status sync with Meta
│
├── sql/
│   ├── migrations/
│   │   ├── 001_messaging_schema.up.sql
│   │   ├── 001_messaging_schema.down.sql
│   │   ├── 002_campaigns.up.sql
│   │   ├── 002_campaigns.down.sql
│   │   ├── 003_conversations.up.sql
│   │   ├── 003_conversations.down.sql
│   │   ├── 004_email_system.up.sql          ← Email templates, events, unsubscribes, suppression
│   │   └── 004_email_system.down.sql
│   └── queries/
│       ├── messages.sql
│       ├── templates.sql
│       ├── campaigns.sql
│       ├── contacts.sql
│       ├── provider_configs.sql
│       ├── conversations.sql
│       ├── analytics.sql
│       ├── email_templates.sql              ← HTML email template CRUD
│       ├── email_events.sql                 ← Open, click, bounce, complaint tracking
│       └── unsubscribes.sql                 ← Unsubscribe + suppression list management
│
└── sqlc.yaml
```

---

## 3. ARCHITECTURE RULES

### 3.1 Hexagonal Architecture (strictly enforced)

```
    HTTP Request
        │
        ▼
    adapters/http/handlers/     ← Validate input, call service, format response
        │
        ▼
    core/services/              ← Business logic, orchestration
        │                         Depends ONLY on ports (interfaces)
        ▼
    core/ports/                 ← Interfaces for repos, providers, queue
        │
        ▼
    adapters/postgres/          ← Implements repository ports
    adapters/providers/         ← Implements provider port
    adapters/queue/             ← Implements queue port
```

### 3.2 Import Rules

| Package | Can Import | CANNOT Import |
|---|---|---|
| `core/domain` | stdlib only | anything else |
| `core/ports` | `core/domain`, stdlib | services, adapters |
| `core/services` | `core/domain`, `core/ports`, stdlib | adapters, handlers |
| `adapters/*` | `core/domain`, `core/ports`, third-party libs | `core/services`, other adapters |
| `handlers` | `core/domain`, `core/ports`, `adapters/http/response` | `core/services` directly — use port interfaces |
| `module.go` | everything (it's the wiring root) | — |

### 3.3 Error Handling

```go
// core/domain/errors.go
package domain

import "errors"

var (
    ErrNotFound          = errors.New("not found")
    ErrAlreadyExists     = errors.New("already exists")
    ErrInvalidInput      = errors.New("invalid input")
    ErrUnauthorized      = errors.New("unauthorized")
    ErrForbidden         = errors.New("forbidden")
    ErrProviderError     = errors.New("provider error")
    ErrRateLimitExceeded = errors.New("rate limit exceeded")
    ErrTemplateNotApproved = errors.New("template not approved")
    ErrOptInRequired     = errors.New("contact has not opted in")
    ErrSessionWindowClosed = errors.New("24-hour session window closed")
)

// DomainError wraps a sentinel error with context
type DomainError struct {
    Err     error
    Message string
    Field   string // optional: which field caused the error
}

func (e *DomainError) Error() string { return e.Message }
func (e *DomainError) Unwrap() error { return e.Err }
```

Handlers map domain errors to HTTP status codes:

```go
// adapters/http/response.go
func mapDomainError(err error) int {
    switch {
    case errors.Is(err, domain.ErrNotFound):
        return http.StatusNotFound
    case errors.Is(err, domain.ErrAlreadyExists):
        return http.StatusConflict
    case errors.Is(err, domain.ErrInvalidInput):
        return http.StatusBadRequest // 400
    case errors.Is(err, domain.ErrUnauthorized):
        return http.StatusUnauthorized
    case errors.Is(err, domain.ErrForbidden):
        return http.StatusForbidden
    case errors.Is(err, domain.ErrRateLimitExceeded):
        return http.StatusTooManyRequests
    case errors.Is(err, domain.ErrProviderError):
        return http.StatusBadGateway
    default:
        return http.StatusInternalServerError
    }
}
```

### 3.4 JSON Response Envelope

ALL API responses use this envelope. No exceptions.

```go
// adapters/http/response.go

// Success response
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Meta    *Pagination `json:"meta,omitempty"`
}

// Error response
type APIError struct {
    Success bool   `json:"success"`
    Error   ErrorDetail `json:"error"`
}

type ErrorDetail struct {
    Code    string `json:"code"`    // machine-readable: "TEMPLATE_NOT_FOUND"
    Message string `json:"message"` // human-readable
    Field   string `json:"field,omitempty"` // which input field, if applicable
}

type Pagination struct {
    Page       int `json:"page"`
    PerPage    int `json:"per_page"`
    Total      int `json:"total"`
    TotalPages int `json:"total_pages"`
}

// Helper functions
func RespondOK(w http.ResponseWriter, data interface{}) { ... }
func RespondCreated(w http.ResponseWriter, data interface{}) { ... }
func RespondList(w http.ResponseWriter, data interface{}, meta Pagination) { ... }
func RespondError(w http.ResponseWriter, err error) { ... }
func RespondValidationError(w http.ResponseWriter, field, message string) { ... }
```

---

## 4. DOMAIN ENTITIES

### 4.1 Core Entities

Domain entities have NO struct tags (no `json`, no `db`). They are pure Go structs.
JSON serialization happens in handlers. DB mapping happens in adapters/postgres/mappers.go.

```go
// core/domain/channel.go
package domain

type Channel string

const (
    ChannelWhatsApp Channel = "whatsapp"
    ChannelSMS      Channel = "sms"
    ChannelEmail    Channel = "email"
)

type ProviderConfig struct {
    ID               uuid.UUID
    TenantID         uuid.UUID
    Channel          Channel
    Provider         string    // "meta_cloud", "msg91", "twilio", "sendgrid", "mailgun", "postmark"
    Credentials      []byte    // AES-256-GCM encrypted JSON
    WebhookSecret    string
    IsActive         bool
    PhoneNumberID    string    // WhatsApp-specific: Meta phone number ID
    WABAID           string    // WhatsApp-specific: WhatsApp Business Account ID
    BusinessID       string    // Meta Business ID
    DisplayPhone     string    // The actual phone number (for display only)
    FromEmail        string    // Email-specific: verified sender address
    FromName         string    // Email-specific: sender display name
    ReplyToEmail     string    // Email-specific: reply-to address
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

// WhatsAppCredentials is the decrypted form of Credentials for WhatsApp
type WhatsAppCredentials struct {
    AccessToken string `json:"access_token"`
    AppSecret   string `json:"app_secret"` // for webhook signature verification
}

// SMSCredentials for MSG91 or Twilio
type SMSCredentials struct {
    AuthKey    string `json:"auth_key"`     // MSG91
    SenderID   string `json:"sender_id"`    // MSG91
    AccountSID string `json:"account_sid"`  // Twilio
    AuthToken  string `json:"auth_token"`   // Twilio
    FromNumber string `json:"from_number"`  // Twilio
}

// EmailCredentials — provider-specific API keys
// The platform default provider is set via MESSAGING_EMAIL_PROVIDER env var.
// Tenants can override by creating their own provider_config with their own keys.
type EmailCredentials struct {
    APIKey       string `json:"api_key"`                  // All providers
    Domain       string `json:"domain,omitempty"`         // Mailgun: sending domain
    ServerToken  string `json:"server_token,omitempty"`   // Postmark: server token
    WebhookToken string `json:"webhook_token,omitempty"`  // For webhook signature verification
}
```

```go
// core/domain/message.go
package domain

type MessageStatus string

const (
    MessageStatusQueued    MessageStatus = "queued"
    MessageStatusSent      MessageStatus = "sent"
    MessageStatusDelivered MessageStatus = "delivered"
    MessageStatusRead      MessageStatus = "read"
    MessageStatusFailed    MessageStatus = "failed"
)

type MessageType string

const (
    MessageTypeTemplate    MessageType = "template"
    MessageTypeText        MessageType = "text"
    MessageTypeMedia       MessageType = "media"
    MessageTypeInteractive MessageType = "interactive"
    MessageTypeLocation    MessageType = "location"
)

type Message struct {
    ID                uuid.UUID
    TenantID          uuid.UUID
    CampaignID        *uuid.UUID
    ConversationID    *uuid.UUID
    Channel           Channel
    Direction         string // "outbound" or "inbound"
    Recipient         string // E.164 phone or email
    Sender            string // platform phone number or email
    MessageType       MessageType
    TemplateID        *uuid.UUID
    TemplateName      *string
    TemplateParams    map[string]string
    TextBody          *string
    MediaURL          *string
    MediaType         *string // image, video, document, audio
    ProviderMessageID string  // Meta's wamid, Twilio SID, etc.
    Status            MessageStatus
    ErrorCode         *string
    ErrorMessage      *string
    Cost              *float64
    SentAt            *time.Time
    DeliveredAt       *time.Time
    ReadAt            *time.Time
    FailedAt          *time.Time
    CreatedAt         time.Time
    Metadata          map[string]string // custom tracking key-value pairs
}
```

```go
// core/domain/template.go
package domain

type TemplateStatus string

const (
    TemplateStatusDraft    TemplateStatus = "draft"
    TemplateStatusPending  TemplateStatus = "pending"
    TemplateStatusApproved TemplateStatus = "approved"
    TemplateStatusRejected TemplateStatus = "rejected"
)

type TemplateCategory string

const (
    TemplateCategoryMarketing      TemplateCategory = "marketing"
    TemplateCategoryUtility        TemplateCategory = "utility"
    TemplateCategoryAuthentication TemplateCategory = "authentication"
)

type Template struct {
    ID                 uuid.UUID
    TenantID           uuid.UUID
    Channel            Channel
    Name               string // alphanumeric + underscore, lowercase
    Language           string // BCP 47: en, en_US, hi, ar
    Category           TemplateCategory
    Components         []TemplateComponent // header, body, footer, buttons
    Status             TemplateStatus
    ProviderTemplateID *string // Meta's template ID after submission
    RejectionReason    *string
    CreatedAt          time.Time
    UpdatedAt          time.Time
}

type TemplateComponentType string

const (
    ComponentHeader  TemplateComponentType = "HEADER"
    ComponentBody    TemplateComponentType = "BODY"
    ComponentFooter  TemplateComponentType = "FOOTER"
    ComponentButtons TemplateComponentType = "BUTTONS"
)

type TemplateComponent struct {
    Type       TemplateComponentType `json:"type"`
    Format     *string               `json:"format,omitempty"` // TEXT, IMAGE, VIDEO, DOCUMENT (header only)
    Text       *string               `json:"text,omitempty"`
    Example    *TemplateExample      `json:"example,omitempty"`
    Buttons    []TemplateButton      `json:"buttons,omitempty"`
}

type TemplateButton struct {
    Type        string  `json:"type"` // QUICK_REPLY, URL, PHONE_NUMBER
    Text        string  `json:"text"`
    URL         *string `json:"url,omitempty"`
    PhoneNumber *string `json:"phone_number,omitempty"`
}

type TemplateExample struct {
    HeaderHandle []string   `json:"header_handle,omitempty"`
    BodyText     [][]string `json:"body_text,omitempty"`
}
```

```go
// core/domain/contact.go
package domain

type OptInStatus string

const (
    OptInStatusPending  OptInStatus = "pending"
    OptInStatusOptedIn  OptInStatus = "opted_in"
    OptInStatusOptedOut OptInStatus = "opted_out"
)

type Contact struct {
    ID              uuid.UUID
    TenantID        uuid.UUID
    Phone           *string // E.164 format
    Email           *string
    Name            *string
    Attributes      map[string]string // custom fields
    Tags            []string
    OptInWhatsApp   OptInStatus
    OptInSMS        OptInStatus
    OptInEmail      OptInStatus
    OptedInAt       *time.Time
    OptedOutAt      *time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

```go
// core/domain/campaign.go
package domain

type CampaignStatus string

const (
    CampaignStatusDraft     CampaignStatus = "draft"
    CampaignStatusScheduled CampaignStatus = "scheduled"
    CampaignStatusRunning   CampaignStatus = "running"
    CampaignStatusPaused    CampaignStatus = "paused"
    CampaignStatusCompleted CampaignStatus = "completed"
    CampaignStatusFailed    CampaignStatus = "failed"
)

type Campaign struct {
    ID             uuid.UUID
    TenantID       uuid.UUID
    Name           string
    Channel        Channel
    TemplateID     uuid.UUID
    TemplateParams map[string]string // static params; per-contact params come from contact attributes
    SegmentID      *uuid.UUID        // target segment
    ContactIDs     []uuid.UUID       // OR explicit contact list
    ScheduledAt    *time.Time
    StartedAt      *time.Time
    CompletedAt    *time.Time
    Status         CampaignStatus
    Stats          CampaignStats
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type CampaignStats struct {
    Total     int `json:"total"`
    Queued    int `json:"queued"`
    Sent      int `json:"sent"`
    Delivered int `json:"delivered"`
    Read      int `json:"read"`
    Failed    int `json:"failed"`
}
```

```go
// core/domain/segment.go
package domain

type Segment struct {
    ID        uuid.UUID
    TenantID  uuid.UUID
    Name      string
    IsDynamic bool
    Rules     []SegmentRule // for dynamic segments
    CreatedAt time.Time
    UpdatedAt time.Time
}

type SegmentRule struct {
    Field    string `json:"field"`    // "tags", "opt_in_whatsapp", "attributes.city"
    Operator string `json:"operator"` // "eq", "neq", "contains", "gt", "lt", "in"
    Value    string `json:"value"`
}
```

---

## 5. PORT INTERFACES

```go
// core/ports/repositories.go
package ports

import (
    "context"
    "github.com/google/uuid"
    "github.com/ranakdinesh/spur-messaging/core/domain"
)

type MessageRepository interface {
    Create(ctx context.Context, msg *domain.Message) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Message, error)
    List(ctx context.Context, tenantID uuid.UUID, filter MessageFilter) ([]domain.Message, int, error)
    UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.MessageStatus, providerMsgID string) error
    UpdateStatusByProviderID(ctx context.Context, providerMsgID string, status domain.MessageStatus, timestamp time.Time) error
    GetByCampaignID(ctx context.Context, tenantID, campaignID uuid.UUID, page, perPage int) ([]domain.Message, int, error)
}

type MessageFilter struct {
    Channel    *domain.Channel
    Status     *domain.MessageStatus
    Recipient  *string
    CampaignID *uuid.UUID
    DateFrom   *time.Time
    DateTo     *time.Time
    Page       int
    PerPage    int
}

type TemplateRepository interface {
    Create(ctx context.Context, tmpl *domain.Template) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error)
    GetByName(ctx context.Context, tenantID uuid.UUID, name, language string) (*domain.Template, error)
    List(ctx context.Context, tenantID uuid.UUID, channel *domain.Channel, status *domain.TemplateStatus, page, perPage int) ([]domain.Template, int, error)
    Update(ctx context.Context, tmpl *domain.Template) error
    UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.TemplateStatus, providerID *string, rejectionReason *string) error
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type ContactRepository interface {
    Create(ctx context.Context, contact *domain.Contact) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Contact, error)
    GetByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*domain.Contact, error)
    GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.Contact, error)
    List(ctx context.Context, tenantID uuid.UUID, filter ContactFilter) ([]domain.Contact, int, error)
    Update(ctx context.Context, contact *domain.Contact) error
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
    BulkCreate(ctx context.Context, contacts []domain.Contact) (int, error) // returns count created
    UpdateOptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel, status domain.OptInStatus) error
    GetBySegment(ctx context.Context, tenantID, segmentID uuid.UUID, page, perPage int) ([]domain.Contact, int, error)
}

type ContactFilter struct {
    Phone     *string
    Email     *string
    Tag       *string
    OptedIn   *domain.Channel // filter contacts opted in to this channel
    Page      int
    PerPage   int
}

type CampaignRepository interface {
    Create(ctx context.Context, campaign *domain.Campaign) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Campaign, error)
    List(ctx context.Context, tenantID uuid.UUID, status *domain.CampaignStatus, page, perPage int) ([]domain.Campaign, int, error)
    Update(ctx context.Context, campaign *domain.Campaign) error
    UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.CampaignStatus) error
    UpdateStats(ctx context.Context, tenantID, id uuid.UUID, stats domain.CampaignStats) error
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
    GetScheduledCampaigns(ctx context.Context, before time.Time) ([]domain.Campaign, error)
}

type ProviderConfigRepository interface {
    Create(ctx context.Context, cfg *domain.ProviderConfig) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ProviderConfig, error)
    GetByChannel(ctx context.Context, tenantID uuid.UUID, channel domain.Channel) (*domain.ProviderConfig, error)
    GetByWABAID(ctx context.Context, wabaID string) (*domain.ProviderConfig, error) // for webhook routing
    List(ctx context.Context, tenantID uuid.UUID) ([]domain.ProviderConfig, error)
    Update(ctx context.Context, cfg *domain.ProviderConfig) error
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type SegmentRepository interface {
    Create(ctx context.Context, segment *domain.Segment) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Segment, error)
    List(ctx context.Context, tenantID uuid.UUID) ([]domain.Segment, error)
    Update(ctx context.Context, segment *domain.Segment) error
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
    ResolveContacts(ctx context.Context, tenantID, segmentID uuid.UUID, page, perPage int) ([]domain.Contact, int, error)
}

type AnalyticsRepository interface {
    GetMessageStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time, channel *domain.Channel) (*domain.MessageAnalytics, error)
    GetCampaignStats(ctx context.Context, tenantID, campaignID uuid.UUID) (*domain.CampaignStats, error)
}
```

```go
// core/ports/services.go
package ports

type MessageService interface {
    Send(ctx context.Context, tenantID uuid.UUID, req SendMessageRequest) (*domain.Message, error)
    SendBulk(ctx context.Context, tenantID uuid.UUID, reqs []SendMessageRequest) ([]domain.Message, error)
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Message, error)
    List(ctx context.Context, tenantID uuid.UUID, filter MessageFilter) ([]domain.Message, int, error)
    Retry(ctx context.Context, tenantID, id uuid.UUID) (*domain.Message, error)
}

type SendMessageRequest struct {
    Channel        domain.Channel
    Recipient      string
    MessageType    domain.MessageType
    TemplateName   *string
    TemplateLanguage *string
    TemplateParams map[string]string
    Text           *string
    MediaURL       *string
    MediaType      *string
    Metadata       map[string]string
}

type TemplateService interface {
    Create(ctx context.Context, tenantID uuid.UUID, req CreateTemplateRequest) (*domain.Template, error)
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error)
    List(ctx context.Context, tenantID uuid.UUID, channel *domain.Channel, status *domain.TemplateStatus, page, perPage int) ([]domain.Template, int, error)
    Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateTemplateRequest) (*domain.Template, error)
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
    SubmitForApproval(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error)
    SyncStatus(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error)
}

type CampaignService interface {
    Create(ctx context.Context, tenantID uuid.UUID, req CreateCampaignRequest) (*domain.Campaign, error)
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Campaign, error)
    List(ctx context.Context, tenantID uuid.UUID, status *domain.CampaignStatus, page, perPage int) ([]domain.Campaign, int, error)
    Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateCampaignRequest) (*domain.Campaign, error)
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
    Execute(ctx context.Context, tenantID, id uuid.UUID) error
    Pause(ctx context.Context, tenantID, id uuid.UUID) error
    Resume(ctx context.Context, tenantID, id uuid.UUID) error
    GetStats(ctx context.Context, tenantID, id uuid.UUID) (*domain.CampaignStats, error)
}

type ContactService interface {
    Create(ctx context.Context, tenantID uuid.UUID, req CreateContactRequest) (*domain.Contact, error)
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Contact, error)
    List(ctx context.Context, tenantID uuid.UUID, filter ContactFilter) ([]domain.Contact, int, error)
    Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateContactRequest) (*domain.Contact, error)
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
    BulkImport(ctx context.Context, tenantID uuid.UUID, contacts []CreateContactRequest) (int, error)
    OptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel) error
    OptOut(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel) error
}

type WebhookService interface {
    HandleWhatsAppWebhook(ctx context.Context, headers http.Header, body []byte) error
    VerifyWhatsAppWebhook(ctx context.Context, mode, token, challenge string) (string, error)
}
```

```go
// core/ports/provider.go
package ports

// Provider is implemented by each channel adapter (whatsapp, sms, email)
type Provider interface {
    Channel() domain.Channel
    Send(ctx context.Context, cfg *domain.ProviderConfig, req ProviderSendRequest) (*ProviderSendResult, error)
    SubmitTemplate(ctx context.Context, cfg *domain.ProviderConfig, tmpl domain.Template) (string, error)
    GetTemplateStatus(ctx context.Context, cfg *domain.ProviderConfig, providerTmplID string) (domain.TemplateStatus, *string, error)
    ParseWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]WebhookEvent, error)
    ValidateWebhookSignature(cfg *domain.ProviderConfig, headers http.Header, body []byte) bool
}

type ProviderSendRequest struct {
    Recipient      string
    MessageType    domain.MessageType
    TemplateName   *string
    TemplateLanguage *string
    TemplateParams map[string]string
    TemplateComponents []domain.TemplateComponent
    Text           *string
    MediaURL       *string
    MediaType      *string
    ReplyToMsgID   *string
}

type ProviderSendResult struct {
    ProviderMessageID string
    Status            domain.MessageStatus
    Cost              *float64
    Timestamp         time.Time
}

type WebhookEventType string

const (
    WebhookEventStatusUpdate WebhookEventType = "status_update"  // delivery receipt
    WebhookEventIncoming     WebhookEventType = "incoming"        // inbound message
)

type WebhookEvent struct {
    Type              WebhookEventType
    ProviderMessageID string
    Status            *domain.MessageStatus // for status updates
    Timestamp         time.Time
    From              *string               // for incoming messages
    Text              *string
    MediaURL          *string
    WABAID            string                // to route to correct tenant
}
```

```go
// core/ports/queue.go
package ports

type MessageQueue interface {
    Enqueue(ctx context.Context, msg QueueMessage) error
    EnqueueBulk(ctx context.Context, msgs []QueueMessage) error
    // Dequeue is called by the worker — implemented as Redis Streams consumer
    StartConsumer(ctx context.Context, handler func(ctx context.Context, msg QueueMessage) error) error
    Stop()
}

type QueueMessage struct {
    MessageID uuid.UUID       `json:"message_id"`
    TenantID  uuid.UUID       `json:"tenant_id"`
    Channel   domain.Channel  `json:"channel"`
    Priority  int             `json:"priority"` // 0 = normal, 1 = high (OTP/auth)
}
```

---

## 6. DATABASE SCHEMA

### 6.1 Migration 001: Core messaging tables

```sql
-- sql/migrations/001_messaging_schema.up.sql

CREATE SCHEMA IF NOT EXISTS messaging;

-- Provider configurations (tenant's own WhatsApp/SMS/Email credentials)
CREATE TABLE messaging.provider_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    channel         TEXT NOT NULL CHECK (channel IN ('whatsapp', 'sms', 'email')),
    provider        TEXT NOT NULL CHECK (provider IN ('meta_cloud', 'msg91', 'twilio', 'sendgrid', 'mailgun', 'postmark')),
    credentials     BYTEA NOT NULL,         -- AES-256-GCM encrypted
    webhook_secret  TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    phone_number_id TEXT,                    -- WhatsApp: Meta phone number ID
    waba_id         TEXT,                    -- WhatsApp: Business Account ID
    business_id     TEXT,                    -- Meta Business ID
    display_phone   TEXT,                    -- Display phone number
    from_email      TEXT,                    -- Email: verified sender address
    from_name       TEXT,                    -- Email: sender display name
    reply_to_email  TEXT,                    -- Email: reply-to address
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, channel, provider)
);

-- Message templates
CREATE TABLE messaging.templates (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    channel              TEXT NOT NULL DEFAULT 'whatsapp',
    name                 TEXT NOT NULL,
    language             TEXT NOT NULL DEFAULT 'en',
    category             TEXT NOT NULL CHECK (category IN ('marketing', 'utility', 'authentication')),
    components           JSONB NOT NULL DEFAULT '[]',
    status               TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending', 'approved', 'rejected')),
    provider_template_id TEXT,
    rejection_reason     TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name, language)
);

-- Contacts
CREATE TABLE messaging.contacts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    phone           TEXT,                    -- E.164 format
    email           TEXT,
    name            TEXT,
    attributes      JSONB NOT NULL DEFAULT '{}',
    tags            TEXT[] NOT NULL DEFAULT '{}',
    opt_in_whatsapp TEXT NOT NULL DEFAULT 'pending' CHECK (opt_in_whatsapp IN ('pending', 'opted_in', 'opted_out')),
    opt_in_sms      TEXT NOT NULL DEFAULT 'pending' CHECK (opt_in_sms IN ('pending', 'opted_in', 'opted_out')),
    opt_in_email    TEXT NOT NULL DEFAULT 'pending' CHECK (opt_in_email IN ('pending', 'opted_in', 'opted_out')),
    opted_in_at     TIMESTAMPTZ,
    opted_out_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_contacts_tenant_phone ON messaging.contacts (tenant_id, phone) WHERE phone IS NOT NULL;
CREATE UNIQUE INDEX idx_contacts_tenant_email ON messaging.contacts (tenant_id, email) WHERE email IS NOT NULL;

-- Messages (outbound and inbound)
CREATE TABLE messaging.messages (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    campaign_id         UUID,
    conversation_id     UUID,
    channel             TEXT NOT NULL,
    direction           TEXT NOT NULL DEFAULT 'outbound' CHECK (direction IN ('outbound', 'inbound')),
    recipient           TEXT NOT NULL,
    sender              TEXT,
    message_type        TEXT NOT NULL CHECK (message_type IN ('template', 'text', 'media', 'interactive', 'location')),
    template_id         UUID REFERENCES messaging.templates(id),
    template_name       TEXT,
    template_params     JSONB,
    text_body           TEXT,
    media_url           TEXT,
    media_type          TEXT,
    provider_message_id TEXT,
    status              TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'sent', 'delivered', 'read', 'failed')),
    error_code          TEXT,
    error_message       TEXT,
    cost                DECIMAL(10, 6),
    sent_at             TIMESTAMPTZ,
    delivered_at        TIMESTAMPTZ,
    read_at             TIMESTAMPTZ,
    failed_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata            JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_messages_tenant_status ON messaging.messages (tenant_id, status);
CREATE INDEX idx_messages_tenant_recipient ON messaging.messages (tenant_id, recipient);
CREATE INDEX idx_messages_tenant_campaign ON messaging.messages (tenant_id, campaign_id) WHERE campaign_id IS NOT NULL;
CREATE INDEX idx_messages_provider_id ON messaging.messages (provider_message_id) WHERE provider_message_id IS NOT NULL;
CREATE INDEX idx_messages_tenant_created ON messaging.messages (tenant_id, created_at DESC);

-- Segments
CREATE TABLE messaging.segments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    name        TEXT NOT NULL,
    is_dynamic  BOOLEAN NOT NULL DEFAULT false,
    rules       JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

-- Static segment membership (for non-dynamic segments)
CREATE TABLE messaging.segment_contacts (
    segment_id UUID NOT NULL REFERENCES messaging.segments(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES messaging.contacts(id) ON DELETE CASCADE,
    added_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (segment_id, contact_id)
);

-- Campaigns
CREATE TABLE messaging.campaigns (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            TEXT NOT NULL,
    channel         TEXT NOT NULL,
    template_id     UUID NOT NULL REFERENCES messaging.templates(id),
    template_params JSONB NOT NULL DEFAULT '{}',
    segment_id      UUID REFERENCES messaging.segments(id),
    contact_ids     UUID[],                  -- explicit contact list (alternative to segment)
    scheduled_at    TIMESTAMPTZ,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'scheduled', 'running', 'paused', 'completed', 'failed')),
    stats           JSONB NOT NULL DEFAULT '{"total":0,"queued":0,"sent":0,"delivered":0,"read":0,"failed":0}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- RLS policies
ALTER TABLE messaging.provider_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.contacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.segments ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.segment_contacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.campaigns ENABLE ROW LEVEL SECURITY;

-- RLS policy: tenant can only see their own data
-- The GUC variable app.tenant_id is set by the identity module's middleware
CREATE POLICY tenant_isolation_provider_configs ON messaging.provider_configs
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_templates ON messaging.templates
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_contacts ON messaging.contacts
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_messages ON messaging.messages
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_segments ON messaging.segments
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_campaigns ON messaging.campaigns
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### 6.2 sqlc.yaml

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries/"
    schema: "sql/migrations/"
    gen:
      go:
        package: "gen"
        out: "adapters/postgres/gen"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
        overrides:
          - db_type: "uuid"
            go_type:
              import: "github.com/google/uuid"
              type: "UUID"
          - db_type: "timestamptz"
            go_type: "time.Time"
          - db_type: "jsonb"
            go_type: "json.RawMessage"
          - db_type: "bytea"
            go_type: "[]byte"
```

---

## 7. HTTP ROUTE MAP

All routes under `/messaging` prefix. Identity module auth middleware is applied
to all routes EXCEPT webhook endpoints.

Webhook endpoints are PUBLIC — verified via provider-specific signature validation.

```
# All routes below are AUTHENTICATED (identity middleware applied by app.go)
# Permission annotations show which authctx.HasPermission() check is required

# Provider Configuration (onboarding)                     PERMISSION
POST   /messaging/providers                     → CreateProvider          messaging:providers:write
GET    /messaging/providers                     → ListProviders           messaging:providers:read
GET    /messaging/providers/{id}                → GetProvider             messaging:providers:read
PUT    /messaging/providers/{id}                → UpdateProvider          messaging:providers:write
DELETE /messaging/providers/{id}                → DeleteProvider          messaging:providers:write
POST   /messaging/providers/{id}/test           → TestProvider            messaging:providers:test

# Templates                                               PERMISSION
POST   /messaging/templates                     → CreateTemplate          messaging:templates:write
GET    /messaging/templates                     → ListTemplates           messaging:templates:read
GET    /messaging/templates/{id}                → GetTemplate             messaging:templates:read
PUT    /messaging/templates/{id}                → UpdateTemplate          messaging:templates:write
DELETE /messaging/templates/{id}                → DeleteTemplate          messaging:templates:write
POST   /messaging/templates/{id}/submit         → SubmitForApproval       messaging:templates:submit
POST   /messaging/templates/{id}/sync           → SyncStatus              messaging:templates:read

# Messages                                                PERMISSION
POST   /messaging/send                          → SendMessage             messaging:messages:send
POST   /messaging/send-bulk                     → SendBulkMessages        messaging:messages:send_bulk
GET    /messaging/messages                      → ListMessages            messaging:messages:read
GET    /messaging/messages/{id}                 → GetMessage              messaging:messages:read

# Contacts                                                PERMISSION
POST   /messaging/contacts                      → CreateContact           messaging:contacts:write
GET    /messaging/contacts                      → ListContacts            messaging:contacts:read
GET    /messaging/contacts/{id}                 → GetContact              messaging:contacts:read
PUT    /messaging/contacts/{id}                 → UpdateContact           messaging:contacts:write
DELETE /messaging/contacts/{id}                 → DeleteContact           messaging:contacts:write
POST   /messaging/contacts/import               → BulkImport              messaging:contacts:import
POST   /messaging/contacts/{id}/opt-in          → OptIn                   messaging:contacts:manage_consent
POST   /messaging/contacts/{id}/opt-out         → OptOut                  messaging:contacts:manage_consent

# Segments                                                PERMISSION
POST   /messaging/segments                      → CreateSegment           messaging:segments:write
GET    /messaging/segments                      → ListSegments            messaging:segments:read
GET    /messaging/segments/{id}                 → GetSegment              messaging:segments:read
PUT    /messaging/segments/{id}                 → UpdateSegment           messaging:segments:write
DELETE /messaging/segments/{id}                 → DeleteSegment           messaging:segments:write
GET    /messaging/segments/{id}/contacts        → ResolveContacts         messaging:segments:read

# Campaigns                                               PERMISSION
POST   /messaging/campaigns                     → CreateCampaign          messaging:campaigns:write
GET    /messaging/campaigns                     → ListCampaigns           messaging:campaigns:read
GET    /messaging/campaigns/{id}                → GetCampaign             messaging:campaigns:read
PUT    /messaging/campaigns/{id}                → UpdateCampaign          messaging:campaigns:write
DELETE /messaging/campaigns/{id}                → DeleteCampaign          messaging:campaigns:write
POST   /messaging/campaigns/{id}/execute        → ExecuteCampaign         messaging:campaigns:execute
POST   /messaging/campaigns/{id}/pause          → PauseCampaign           messaging:campaigns:execute
POST   /messaging/campaigns/{id}/resume         → ResumeCampaign          messaging:campaigns:execute
GET    /messaging/campaigns/{id}/stats          → GetCampaignStats        messaging:campaigns:read

# Analytics                                               PERMISSION
GET    /messaging/analytics/messages            → MessageAnalytics        messaging:analytics:read
GET    /messaging/analytics/overview            → DashboardOverview       messaging:analytics:read

# Webhooks (MOUNTED BY app.go OUTSIDE auth middleware group)
# These are NOT in RegisterRoutes() — they are on module.WebhookHandler
# Verified by provider HMAC signature, not by identity middleware
GET    /messaging/webhook/whatsapp              → WebhookHandler.Verify (Meta verification challenge)
POST   /messaging/webhook/whatsapp              → WebhookHandler.Handle (delivery receipts + inbound)
POST   /messaging/webhook/sms                   → WebhookHandler.HandleSMS
POST   /messaging/webhook/email                 → WebhookHandler.HandleEmail
```

---

## 8. OPENAPI SPEC CONVENTIONS

The `openapi.json` file at repo root is the SINGLE source of truth for any frontend
or API consumer. Generate it AFTER implementing the handlers, ensuring every route
is documented.

### Conventions:
- OpenAPI version: 3.1.0
- All request/response bodies use the standard envelope (Section 3.4)
- All list endpoints support `page` and `per_page` query params
- All IDs are UUID format
- All dates are ISO 8601 / RFC 3339
- Authentication: Bearer token (JWT from identity module) in Authorization header
- Tenant context: extracted from JWT claims (no separate header needed)
- Error responses use the `APIError` schema consistently
- Tag each endpoint group: `templates`, `messages`, `contacts`, `campaigns`, `segments`, `providers`, `analytics`, `webhooks`

---

## 9. WHATSAPP CLOUD API REFERENCE

### 9.1 Base URL
```
https://graph.facebook.com/v20.0
```

### 9.2 Send Template Message
```
POST /{phone_number_id}/messages
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "messaging_product": "whatsapp",
  "to": "919810914244",
  "type": "template",
  "template": {
    "name": "hello_world",
    "language": { "code": "en_US" },
    "components": [
      {
        "type": "body",
        "parameters": [
          { "type": "text", "text": "Dinesh" }
        ]
      }
    ]
  }
}
```

### 9.3 Send Text Message (within 24hr window only)
```
POST /{phone_number_id}/messages
{
  "messaging_product": "whatsapp",
  "to": "919810914244",
  "type": "text",
  "text": { "body": "Hello, how can we help?" }
}
```

### 9.4 Webhook Payload (Status Update)
```json
{
  "entry": [{
    "id": "WABA_ID",
    "changes": [{
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {
          "display_phone_number": "919810914244",
          "phone_number_id": "PHONE_NUMBER_ID"
        },
        "statuses": [{
          "id": "wamid.xxx",
          "status": "delivered",
          "timestamp": "1234567890",
          "recipient_id": "919876543210"
        }]
      },
      "field": "messages"
    }]
  }]
}
```

### 9.5 Webhook Signature Verification
Verify using HMAC-SHA256 with the app secret:
```go
mac := hmac.New(sha256.New, []byte(appSecret))
mac.Write(body)
expected := hex.EncodeToString(mac.Sum(nil))
// Compare with X-Hub-Signature-256 header (strip "sha256=" prefix)
```

### 9.6 Template Submission
```
POST /{waba_id}/message_templates
{
  "name": "order_confirmation",
  "language": "en_US",
  "category": "UTILITY",
  "components": [
    {
      "type": "BODY",
      "text": "Hi {{1}}, your order {{2}} has been confirmed.",
      "example": { "body_text": [["Dinesh", "ORD-12345"]] }
    }
  ]
}
```

---

## 10. IMPLEMENTATION PHASES

### Phase 1 — Foundation (implement in this order)

**Step 1: Scaffold**
1. `go mod init github.com/ranakdinesh/spur-messaging`
2. Create directory structure exactly as Section 2 (including `pkg/authctx/`)
3. Create `spur.json` — copy exactly from Section 0A
4. Create `.env.example` — copy from Section 0A
5. Create `pkg/authctx/authctx.go` — the auth context helper from Section 1A.2
6. Create `core/domain/*.go` — all entity structs from Section 4
7. Create `core/domain/errors.go` — all error types from Section 10A.5 (NOT Section 3.3)
8. Create `core/ports/*.go` — all interfaces from Section 5
9. Run `go build ./...` — must compile with zero errors

**Step 2: Database**
1. Create `sql/migrations/001_messaging_schema.up.sql` from Section 6.1
2. Create `sqlc.yaml` from Section 6.2
3. Write SQLC queries in `sql/queries/*.sql` for ALL repository methods
4. Run `sqlc generate` — must produce clean generated code
5. Create `adapters/postgres/store.go` implementing all repository ports
6. Create `adapters/postgres/mappers.go` for sqlc model ↔ domain entity conversion

**Step 3: WhatsApp Provider**
1. Create `adapters/providers/whatsapp/types.go` — Meta API structs
2. Create `adapters/providers/whatsapp/client.go` — HTTP client for graph.facebook.com
3. Create `adapters/providers/whatsapp/webhook.go` — signature verification + event parsing
4. Create `adapters/providers/whatsapp/templates.go` — template CRUD on Meta
5. Implement the `ports.Provider` interface

**Step 4: Crypto**
1. Create `adapters/crypto/encrypt.go` — AES-256-GCM encrypt/decrypt
2. Used by provider config handlers to encrypt credentials before storage

**Step 5: Services**
1. Implement all services in `core/services/`
2. Each service depends ONLY on port interfaces (injected via constructor)
3. `message_service.go`: validates input, checks opt-in, enqueues to queue
4. `template_service.go`: CRUD + calls provider.SubmitTemplate
5. `campaign_service.go`: validates template is approved, resolves contacts, fans out to queue
6. `contact_service.go`: CRUD + opt-in/out management
7. `webhook_service.go`: routes webhook events to correct tenant, updates message status

**Step 6: HTTP Handlers**
1. Create `adapters/http/response.go` — response envelope from Section 3.4
2. Create `adapters/http/middleware.go` — rate limiting middleware (auth middleware is EXTERNAL)
3. Create all handlers in `adapters/http/handlers/`
4. Each handler MUST follow the pattern in Section 1A.5: `authctx.TenantID()` → permission check → validate → service → respond
5. Create `adapters/http/routes.go` — route registration matching Section 7
6. Webhook handler does NOT use authctx — it uses HMAC verification (Section 1A.7)

**Step 7: Queue + Worker**
1. Create `adapters/queue/redis_streams.go` implementing `ports.MessageQueue`
2. Create `worker/sender.go` — dequeue message, load provider config, call provider.Send, update status
3. Worker handles retries with exponential backoff (max 3 retries)

**Step 8: Module Wiring**
1. Create `module.go` — wire everything together in `New()`
2. Create `RegisterRoutes()` — mount the chi subrouter
3. Run `go build ./...` — must compile

**Step 9: OpenAPI**
1. Generate `openapi.json` from the implemented routes
2. Every endpoint documented with request/response schemas
3. Place at repo root

**Validation after Phase 1:**
```bash
go build ./...         # must pass
go vet ./...           # must pass
sqlc compile           # must pass
# The following must exist and be non-empty:
# - module.go with New() and RegisterRoutes()
# - openapi.json at repo root
# - All files in the directory structure from Section 2
```

---

## 10A. INPUT VALIDATION RULES

Every handler MUST validate input before calling the service layer.
Every service MUST validate business rules before calling repositories or providers.
These rules are NON-NEGOTIABLE — do not skip any.

### 10A.1 Field-level validation (handler layer)

Handlers validate format and presence. Return 400 with field name on failure.

```
FIELD                   VALIDATION RULE                                          ERROR MESSAGE
──────────────────────────────────────────────────────────────────────────────────────────────────
phone                   Must be E.164 format: ^\+[1-9]\d{6,14}$                  "phone must be E.164 format (e.g. +919810914244)"
                        Strip spaces, dashes, parentheses before validating
                        If user sends "9810914244", do NOT auto-prepend +91

email                   Must match RFC 5322 basic format                         "invalid email address"
                        Max 254 characters
                        Lowercase before storage

template.name           Lowercase alphanumeric + underscore only: ^[a-z0-9_]+$   "template name must be lowercase alphanumeric with underscores"
                        Min 1, max 512 characters
                        No spaces, no hyphens, no special characters

template.language       Must be valid BCP 47 code: en, en_US, hi, ar, etc.       "invalid language code"

template.category       Must be one of: marketing, utility, authentication       "category must be marketing, utility, or authentication"

campaign.name           Min 1, max 255 characters, trimmed                       "campaign name is required (max 255 chars)"

channel                 Must be one of: whatsapp, sms, email                     "channel must be whatsapp, sms, or email"

page                    Integer >= 1, default 1                                  "page must be >= 1"
per_page                Integer 1-100, default 20                                "per_page must be between 1 and 100"

uuid params             Must be valid UUID v4                                    "invalid ID format"

scheduled_at            Must be RFC 3339, must be in the future (> now + 5min)   "scheduled_at must be at least 5 minutes in the future"

email subject           Min 1, max 998 characters (RFC 2822 limit)               "subject is required (max 998 chars)"

email html_body         Min 1, max 5MB                                           "html_body is required (max 5MB)"

attachment.content      Base64 encoded, max 10MB per attachment                  "attachment exceeds 10MB limit"
                        Max 10 attachments per email
                        content_type must be a valid MIME type

tags                    Array of strings, max 10 tags, each max 50 chars         "max 10 tags allowed, each max 50 chars"

metadata                Map of strings, max 20 keys, each key max 50 chars,      "metadata: max 20 keys, key max 50, value max 500 chars"
                        each value max 500 chars

bulk import             Max 10,000 contacts per import request                   "bulk import limited to 10,000 contacts per request"

campaign contact_ids    Max 100,000 contacts per campaign                        "campaign limited to 100,000 contacts"

template_params         All referenced variables must have values                "missing template variable: {{variable_name}}"

segment rules           Max 10 rules per segment                                 "max 10 rules per segment"
                        operator must be: eq, neq, contains, gt, lt, in
                        field must be: tags, phone, email, name, opt_in_*,
                        attributes.* (custom attribute prefix)
```

### 10A.2 Business rule validation (service layer)

Services validate business logic. Return domain errors (not HTTP codes).

**Message sending — ALL channels:**
```
CHECK                                    WHEN                    ERROR
──────────────────────────────────────────────────────────────────────────────
Provider configured for channel?         Every send              ErrProviderNotConfigured
Provider credentials valid (decryptable)?Every send              ErrProviderError("invalid credentials")
Contact exists for recipient?            Every send              ErrNotFound("contact not found")
Contact opted in for channel?            Every send              ErrOptInRequired
Rate limit not exceeded for tenant?      Every send              ErrRateLimitExceeded
```

**WhatsApp-specific:**
```
CHECK                                    WHEN                    ERROR
──────────────────────────────────────────────────────────────────────────────
Template approved by Meta?               template message send   ErrTemplateNotApproved
Within 24hr session window?              text/media/interactive  ErrSessionWindowClosed
                                         (non-template sends)    → suggest using template instead
Phone number is valid WhatsApp number?   send                    Provider returns error → store as failed
Template exists with matching language?  template send           ErrNotFound("template not found")
All template variables provided?         template send           ErrInvalidInput("missing variable: {{name}}")
Media URL accessible?                    media message send      ErrInvalidInput("media URL not accessible")
```

**Email-specific:**
```
CHECK                                    WHEN                    ERROR
──────────────────────────────────────────────────────────────────────────────
Email on suppression list?               EVERY email send        Block silently, log as "dropped", do NOT return error
                                                                 to caller — just record it
Email unsubscribed (global)?             marketing email send    Block, log as "dropped"
Email unsubscribed (category)?           matching category send  Block, log as "dropped"
Email unsubscribed (campaign)?           matching campaign send  Block, log as "dropped"
Transactional email + unsubscribed?      transactional send      ALLOW — transactional skips unsubscribe check
Transactional email + suppressed?        transactional send      BLOCK — suppression is NEVER skipped
FROM address verified with provider?     send                    Provider returns error → ErrProviderError
HTML body contains unsubscribe link?     marketing email send    Auto-inject if missing (do NOT reject)
List-Unsubscribe header present?         marketing email send    Auto-inject (do NOT reject)
```

**SMS-specific:**
```
CHECK                                    WHEN                    ERROR
──────────────────────────────────────────────────────────────────────────────
DLT template ID provided (India)?        promotional SMS (India) ErrInvalidInput("DLT template ID required")
Sender ID valid for message type?        India SMS               ErrInvalidInput("sender ID not valid for this type")
Message length within limits?            send                    Auto-split into multi-part (warn in response)
DND/NCPR check passed?                   promotional SMS (India) Block, log as "dropped"
```

**Template operations:**
```
CHECK                                    WHEN                    ERROR
──────────────────────────────────────────────────────────────────────────────
Template in draft status?                submit for approval     ErrInvalidInput("only draft templates can be submitted")
Template in draft/rejected status?       update                  ErrInvalidInput("approved/pending templates cannot be edited")
Template not used by active campaign?    delete                  ErrConflict("template used by active campaign")
Unique name+language per tenant?         create/update           ErrAlreadyExists("template with this name and language exists")
```

**Campaign operations:**
```
CHECK                                    WHEN                    ERROR
──────────────────────────────────────────────────────────────────────────────
Campaign status is draft/scheduled?      update                  ErrInvalidInput("cannot update running/completed campaign")
Campaign status is draft?                execute                 ErrInvalidInput("campaign must be in draft or scheduled status")
Campaign status is running?              pause                   ErrInvalidInput("can only pause running campaigns")
Campaign status is paused?               resume                  ErrInvalidInput("can only resume paused campaigns")
Referenced template exists?              create/execute          ErrNotFound("template not found")
Referenced template approved?            execute (WhatsApp)      ErrTemplateNotApproved
Referenced segment exists?               create with segment     ErrNotFound("segment not found")
Segment resolves to > 0 contacts?       execute                 ErrInvalidInput("segment has no contacts")
All contacts opted in for channel?       execute                 Filter out non-opted-in, log count, proceed with opted-in
Scheduled time in future?                schedule                ErrInvalidInput("scheduled_at must be in the future")
Campaign not already completed?          execute again           ErrInvalidInput("campaign already completed")
```

**Contact operations:**
```
CHECK                                    WHEN                    ERROR
──────────────────────────────────────────────────────────────────────────────
Phone or email required?                 create                  ErrInvalidInput("phone or email is required")
Phone unique per tenant?                 create/update           ErrAlreadyExists("contact with this phone exists")
Email unique per tenant?                 create/update           ErrAlreadyExists("contact with this email exists")
Contact not in active campaign?          delete                  Allow delete, campaign skips missing contacts
Already opted in?                        opt-in                  Idempotent — return success, no error
Already opted out?                       opt-out                 Idempotent — return success, no error
```

**Provider config operations:**
```
CHECK                                    WHEN                    ERROR
──────────────────────────────────────────────────────────────────────────────
One active config per channel per tenant create                  ErrAlreadyExists("active config for this channel exists")
Credentials decryptable?                 create/update           ErrInvalidInput("invalid credentials format")
Can reach provider API?                  test endpoint           Return test result (success/failure) — not an error
```

### 10A.3 Edge cases and unhappy paths

**Duplicate webhook delivery (idempotency):**
```
- Every webhook event has a provider_event_id (wamid for WhatsApp, event ID for email providers)
- Before processing, check: email_events.provider_event_id or messages.provider_message_id
- If already exists, return 200 OK immediately (do NOT process again)
- This prevents double-counting opens, clicks, and double-updating statuses
```

**Out-of-order status updates:**
```
- WhatsApp may deliver "read" before "delivered" webhook arrives
- Status progression: queued → sent → delivered → read → (failed is terminal)
- Rule: NEVER downgrade status. If current status is "read", ignore incoming "delivered"
- Implement status_rank: queued=0, sent=1, delivered=2, read=3, failed=99
- Only update if incoming rank > current rank (except failed, which is always terminal)
```

**Campaign crash recovery:**
```
- Worker crashes mid-campaign (50,000 messages, crashed at 23,000)
- On restart, campaign_executor checks: status="running" campaigns
- For each running campaign, count messages already queued/sent vs total contacts
- Resume from where it stopped — do NOT re-send to already-sent contacts
- Use campaign_id + contact_id uniqueness: if message already exists for this
  campaign+contact, skip
```

**Provider timeout/failure:**
```
- HTTP call to provider times out (30 second timeout)
- Worker retries with exponential backoff: 5s, 30s, 120s (3 attempts max)
- If all retries fail: update message status to "failed", error_code="PROVIDER_TIMEOUT"
- For campaigns: increment stats.failed, continue with next message
- NEVER block the entire campaign for one failed message
```

**Provider rate limiting (429):**
```
- Meta Cloud API: 80 msg/s (Tier 1) to 1000 msg/s (Tier 4)
- SendGrid: varies by plan
- Response: HTTP 429 with Retry-After header
- Worker: pause sending for Retry-After seconds (or 60s if no header)
- Do NOT count as a failure — keep message in queue for retry
- Log rate limit event for analytics
```

**Provider credential revoked/expired:**
```
- Meta access token expires or is revoked
- Provider returns 401/403
- Update provider_config.is_active = false
- Stop all sending for that tenant+channel
- Log alert: "Provider credentials expired for tenant {id}, channel {channel}"
- Return ErrProviderError("credentials expired — reconfigure in settings")
- Do NOT retry — credential errors are not transient
```

**Concurrent campaign execution:**
```
- User clicks "Execute" twice rapidly
- Use SELECT ... FOR UPDATE on campaign row before status change
- If status already changed from "draft" to "running", second request gets
  ErrInvalidInput("campaign is already running")
- Database-level locking prevents race condition
```

**Contact deleted during campaign:**
```
- Campaign resolved 1000 contacts, but contact #500 is deleted mid-send
- Worker tries to send → contact lookup fails → skip, log as "dropped"
- Do NOT fail the campaign — increment a "skipped" counter
- Campaign stats: {"total": 1000, "sent": 800, "skipped": 5, "failed": 3, ...}
```

**Template deleted/deactivated while campaign references it:**
```
- Campaign was created with template_id pointing to an approved template
- Template is deleted or status changes to rejected before campaign executes
- On campaign.Execute(): re-check template status
- If template gone/rejected: fail the campaign with error, do NOT send partial
- This is a hard failure — campaign status → "failed", error: "template no longer available"
```

**Bulk import edge cases:**
```
- CSV has 10,000 rows, 500 have invalid phone numbers
- Process ALL rows, collect errors per row
- Return: {"imported": 9500, "errors": [{"row": 5, "error": "invalid phone"}, ...]}
- Do NOT fail entire import for individual row errors
- Duplicate detection: if phone/email already exists for tenant, skip (log as duplicate)
- Return duplicate count in response: {"imported": 9000, "duplicates": 500, "errors": [...]}
```

**Segment resolves differently at campaign time vs creation time:**
```
- Dynamic segment "opted_in_whatsapp = true" had 1000 contacts when campaign was created
- By execution time, 50 more opted in, 30 opted out
- ALWAYS resolve segment at execution time, not creation time
- This is correct behavior — document it in API response
```

**Redis connection failure:**
```
- Redis is down when message needs to be enqueued
- Return ErrInternalError to the API caller (500)
- Do NOT silently drop the message
- For campaigns: pause the campaign, set status to "failed" with error "queue unavailable"
- Worker: if Redis connection drops mid-processing, message stays in stream
  (acknowledged only after successful send)
```

**Database connection failure:**
```
- DB pool exhausted or connection drops
- All repository methods return error → service returns error → handler returns 503
- Worker: stop processing, wait for reconnection (pgxpool handles reconnection)
- Do NOT lose messages — Redis Streams retains unacknowledged messages
```

**Webhook endpoint receives malformed payload:**
```
- Provider sends unexpected JSON structure or missing fields
- Log the raw payload for debugging
- Return 200 OK to provider (returning errors causes aggressive retries)
- Do NOT crash the webhook handler — use defensive parsing with zero-value defaults
```

**Unsubscribe link clicked after contact re-subscribed:**
```
- Unsubscribe tokens do NOT expire (unlike password reset tokens)
- If user clicks old unsubscribe link, it SHOULD still work
- Process unsubscribe regardless of current opt-in status
- This is correct — user intent to unsubscribe is always honored
```

**Same email on suppression list AND explicitly opted in:**
```
- Suppression list ALWAYS wins over opt-in status
- Admin must explicitly remove from suppression list first
- Then contact can re-opt-in
- UI should warn admin: "this email is suppressed due to hard bounce — are you sure?"
- API: removing from suppression is a separate endpoint from opt-in
```

### 10A.4 Error response codes

Map every domain error to a consistent error code in the API response.

```go
// adapters/http/response.go

var errorCodeMap = map[error]struct{ code string; status int }{
    // Generic
    domain.ErrNotFound:             {"NOT_FOUND", 404},
    domain.ErrAlreadyExists:        {"ALREADY_EXISTS", 409},
    domain.ErrInvalidInput:         {"INVALID_INPUT", 400},
    domain.ErrUnauthorized:         {"UNAUTHORIZED", 401},
    domain.ErrForbidden:            {"FORBIDDEN", 403},
    domain.ErrRateLimitExceeded:    {"RATE_LIMIT_EXCEEDED", 429},

    // Messaging-specific
    domain.ErrProviderError:        {"PROVIDER_ERROR", 502},
    domain.ErrProviderNotConfigured:{"PROVIDER_NOT_CONFIGURED", 422},
    domain.ErrTemplateNotApproved:  {"TEMPLATE_NOT_APPROVED", 422},
    domain.ErrOptInRequired:        {"OPT_IN_REQUIRED", 422},
    domain.ErrSessionWindowClosed:  {"SESSION_WINDOW_CLOSED", 422},
    domain.ErrSuppressed:           {"EMAIL_SUPPRESSED", 422},
    domain.ErrUnsubscribed:         {"RECIPIENT_UNSUBSCRIBED", 422},
    domain.ErrCampaignNotExecutable:{"CAMPAIGN_NOT_EXECUTABLE", 422},
    domain.ErrTemplateInUse:        {"TEMPLATE_IN_USE", 409},
    domain.ErrCredentialsExpired:   {"CREDENTIALS_EXPIRED", 422},
    domain.ErrQueueUnavailable:     {"QUEUE_UNAVAILABLE", 503},
}
```

Every error response body:
```json
{
  "success": false,
  "error": {
    "code": "TEMPLATE_NOT_APPROVED",
    "message": "Template 'order_update' is pending approval — cannot send until approved by Meta",
    "field": "template_id"
  }
}
```

### 10A.5 Domain error types (UPDATED — replace Section 3.3)

```go
// core/domain/errors.go
package domain

import "errors"

var (
    // Generic
    ErrNotFound          = errors.New("not found")
    ErrAlreadyExists     = errors.New("already exists")
    ErrInvalidInput      = errors.New("invalid input")
    ErrUnauthorized      = errors.New("unauthorized")
    ErrForbidden         = errors.New("forbidden")

    // Provider
    ErrProviderError         = errors.New("provider error")
    ErrProviderNotConfigured = errors.New("provider not configured")
    ErrProviderTimeout       = errors.New("provider timeout")
    ErrCredentialsExpired    = errors.New("credentials expired")
    ErrRateLimitExceeded     = errors.New("rate limit exceeded")

    // WhatsApp
    ErrTemplateNotApproved   = errors.New("template not approved")
    ErrSessionWindowClosed   = errors.New("session window closed")

    // Email
    ErrSuppressed            = errors.New("email suppressed")
    ErrUnsubscribed          = errors.New("recipient unsubscribed")

    // Contacts
    ErrOptInRequired         = errors.New("opt-in required")

    // Campaigns
    ErrCampaignNotExecutable = errors.New("campaign not executable")
    ErrTemplateInUse         = errors.New("template in use")

    // Infrastructure
    ErrQueueUnavailable      = errors.New("queue unavailable")
)

// DomainError wraps a sentinel error with context
type DomainError struct {
    Err     error
    Message string // human-readable detail
    Field   string // which field caused the error (optional)
}

func (e *DomainError) Error() string { return e.Message }
func (e *DomainError) Unwrap() error { return e.Err }

// Convenience constructors
func NewValidationError(field, message string) *DomainError {
    return &DomainError{Err: ErrInvalidInput, Message: message, Field: field}
}

func NewNotFoundError(resource string) *DomainError {
    return &DomainError{Err: ErrNotFound, Message: resource + " not found"}
}

func NewConflictError(message string) *DomainError {
    return &DomainError{Err: ErrAlreadyExists, Message: message}
}

func NewProviderError(message string) *DomainError {
    return &DomainError{Err: ErrProviderError, Message: message}
}
```

---

## 11. CRITICAL RULES FOR AI AGENTS

1. **No frontend code.** Not a single line of HTML, CSS, JS, or TSX.
2. **No shared type repos.** This module imports only stdlib and third-party packages.
3. **No log.Fatal or os.Exit.** Return errors. Always.
4. **Domain entities have NO struct tags.** JSON tags go on handler request/response structs. DB tags don't exist — SQLC handles mapping.
5. **Handlers accept port interfaces, not concrete service structs.**
6. **Every handler function validates input before calling the service.**
7. **All credentials are encrypted with AES-256-GCM before storage.** Never store plaintext tokens.
8. **Webhook endpoints have NO auth middleware.** They verify provider signatures instead.
9. **All list endpoints return paginated results** using the Pagination meta struct.
10. **The OpenAPI spec is generated AFTER code is written**, not before. It must match the actual implementation.
11. **WhatsApp 24-hour rule**: text/media messages can only be sent within 24 hours of last customer message. Outside this window, only approved templates can be sent. The service layer MUST enforce this.
12. **Opt-in enforcement**: message_service.Send() MUST check that the contact has opted in to the target channel before enqueuing. Return `ErrOptInRequired` if not.
13. **Use `context.Context` everywhere.** Pass it through from handler → service → repo → provider.
14. **Redis Streams consumer group** for the message queue, not pub/sub. Messages must survive worker restarts.
15. **Direct Cloud API mode**: tenants provide their own Meta access tokens. The platform does NOT manage WABAs — it uses the tenant's credentials to call Meta's API on their behalf.
16. **This module does NOT implement auth.** Authentication, RBAC, and tenant isolation are handled by the identity module's middleware applied externally in app.go. This module only READS auth context via `pkg/authctx/authctx.go`.
17. **Every mutating handler MUST check permissions** using `authctx.HasPermission(ctx, "messaging:<resource>:<action>")`. See Section 1A.5 for the mandatory handler pattern.
18. **TenantID always comes from context**, never from request body, URL params, or headers. `authctx.TenantID(ctx)` is the only source.
19. **Services receive tenantID, not auth details.** The service layer is auth-agnostic — it receives `tenantID uuid.UUID` as a parameter and nothing else about the caller's identity.
20. **Webhook routes are mounted separately by app.go**, outside the auth middleware group. The `WebhookHandler` is exposed on the `Module` struct for this purpose.
21. **API key access is transparent.** The identity module's AuthGuard handles API keys identically to JWTs — same context values, same permission checks. The messaging module cannot and should not distinguish between them.
22. **Email unsubscribe links are MANDATORY.** Every marketing email MUST include a one-click unsubscribe link (CAN-SPAM, GDPR). The `List-Unsubscribe` and `List-Unsubscribe-Post` headers MUST be set on every marketing email.
23. **Email suppression list is checked BEFORE sending.** If a contact is on the suppression list (hard bounce, complaint, or manual unsubscribe), the send MUST be blocked. Never send to a suppressed address.
24. **Cross-module email sending** uses the `EmailSender` port interface (Section 12.3). Other modules call `Send()` with a structured request — they never interact with email providers directly.
25. **ALL input validation rules from Section 10A.1 are mandatory.** Every handler must validate every field listed. Do not skip validations.
26. **ALL business rule validations from Section 10A.2 are mandatory.** Every service method must check the business rules listed for that operation.
27. **Idempotent webhook processing.** Check `provider_event_id` before processing any webhook event. If already processed, return 200 OK without re-processing. See Section 10A.3.
28. **Never downgrade message status.** Status can only move forward: queued→sent→delivered→read. If current status is "read" and a "delivered" webhook arrives, ignore it. See Section 10A.3.
29. **Campaign crash recovery.** On worker restart, check for campaigns with status="running" and resume from where they stopped. Use campaign_id+contact_id uniqueness to avoid duplicate sends. See Section 10A.3.
30. **Use the expanded error types from Section 10A.5**, not the minimal set from Section 3.3. Section 10A.5 supersedes Section 3.3.
31. **Every error response must include the error code from Section 10A.4.** The `code` field is machine-readable (e.g. "TEMPLATE_NOT_APPROVED"), the `message` field is human-readable with context.

---

## 12. EMAIL SYSTEM

The email system is a core part of the messaging module — NOT a separate module.
It shares the same queue, contact management, campaign engine, and analytics
infrastructure as WhatsApp and SMS.

Email serves TWO purposes in the platform:

1. **Transactional emails** — sent by OTHER modules (identity sends password reset,
   campaigns module sends order confirmations, etc.) via the `EmailSender` interface.
2. **Email campaigns** — bulk marketing emails with templates, audience targeting,
   scheduling, analytics, and unsubscribe management.

### 12.1 Email provider selection

The platform supports three email providers. Selection is via env var with per-tenant override.

```
# .env — Platform defaults
MESSAGING_EMAIL_PROVIDER=sendgrid     # "sendgrid" | "mailgun" | "postmark"
SENDGRID_API_KEY=SG.xxxx
MAILGUN_API_KEY=key-xxxx
MAILGUN_DOMAIN=mg.citual.com
POSTMARK_SERVER_TOKEN=xxxx

MESSAGING_SMS_PROVIDER=msg91          # "msg91" | "twilio"
MSG91_AUTH_KEY=xxxx
MSG91_SENDER_ID=CITUAL
TWILIO_ACCOUNT_SID=ACxxxx
TWILIO_AUTH_TOKEN=xxxx
TWILIO_FROM_NUMBER=+1234567890
```

**Provider resolution order:**
1. Check tenant's `provider_configs` table for channel=email — if found, use tenant's own credentials
2. Fall back to platform default from env vars
3. If no provider configured, return `ErrProviderNotConfigured`

This means: small tenants use the platform's SendGrid account (you pay, included in SaaS fee),
large tenants bring their own SendGrid/Mailgun/Postmark API keys for better deliverability and
dedicated IP reputation.

### 12.2 Email-specific domain entities

```go
// core/domain/email_template.go
package domain

// EmailTemplate is a reusable HTML email template.
// SEPARATE from the WhatsApp Template entity — email templates have different
// structure (HTML body, subject, preview text) and don't need Meta approval.
type EmailTemplate struct {
    ID            uuid.UUID
    TenantID      uuid.UUID
    Name          string                // Unique per tenant: "welcome_email", "order_shipped"
    Subject       string                // Email subject line — supports {{variable}} substitution
    PreviewText   string                // Preview text shown in inbox (first ~90 chars)
    HTMLBody      string                // Full HTML email body — supports {{variable}} substitution
    TextBody      string                // Plain text fallback (auto-generated from HTML if empty)
    Category      EmailCategory         // transactional, marketing, notification
    Variables     []string              // List of variable names used in template: ["name", "order_id"]
    IsActive      bool
    Version       int                   // Auto-incremented on each update
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type EmailCategory string

const (
    EmailCategoryTransactional EmailCategory = "transactional"
    EmailCategoryMarketing     EmailCategory = "marketing"
    EmailCategoryNotification  EmailCategory = "notification"
)

// EmailMessage extends the base Message with email-specific fields.
// These fields are stored in the Message.Metadata JSONB column.
type EmailMessageMeta struct {
    Subject      string            `json:"subject"`
    FromEmail    string            `json:"from_email"`
    FromName     string            `json:"from_name"`
    ReplyTo      string            `json:"reply_to,omitempty"`
    CC           []string          `json:"cc,omitempty"`
    BCC          []string          `json:"bcc,omitempty"`
    Headers      map[string]string `json:"headers,omitempty"`    // Custom headers
    Attachments  []EmailAttachment `json:"attachments,omitempty"`
    TrackOpens   bool              `json:"track_opens"`
    TrackClicks  bool              `json:"track_clicks"`
    Tags         []string          `json:"tags,omitempty"`       // Provider tags for filtering
    Category     EmailCategory     `json:"category"`
    IPPool       string            `json:"ip_pool,omitempty"`    // SendGrid IP pool
}

type EmailAttachment struct {
    Filename    string `json:"filename"`
    ContentType string `json:"content_type"` // e.g. "application/pdf"
    Content     string `json:"content"`      // base64-encoded content
    ContentID   string `json:"content_id,omitempty"` // for inline images
}

// EmailEvent tracks granular email lifecycle events from provider webhooks.
// One message can have multiple events (sent → delivered → opened → clicked).
type EmailEvent struct {
    ID                uuid.UUID
    TenantID          uuid.UUID
    MessageID         uuid.UUID         // FK to messaging.messages
    CampaignID        *uuid.UUID
    EventType         EmailEventType
    Recipient         string            // email address
    Timestamp         time.Time
    ProviderEventID   string            // provider's unique event ID (for dedup)
    UserAgent         string            // for open/click events
    IPAddress         string            // for open/click events
    URL               string            // for click events: which link was clicked
    BounceType        *string           // "hard" or "soft" (for bounce events)
    BounceReason      *string           // SMTP error message
    ComplaintFeedback *string           // ISP complaint feedback type
    RawPayload        map[string]string // provider's raw webhook data for debugging
    CreatedAt         time.Time
}

type EmailEventType string

const (
    EmailEventDelivered    EmailEventType = "delivered"
    EmailEventBounce       EmailEventType = "bounce"
    EmailEventSoftBounce   EmailEventType = "soft_bounce"
    EmailEventOpen         EmailEventType = "open"
    EmailEventClick        EmailEventType = "click"
    EmailEventUnsubscribe  EmailEventType = "unsubscribe"
    EmailEventComplaint    EmailEventType = "complaint"    // ISP spam complaint
    EmailEventDropped      EmailEventType = "dropped"      // provider refused to send
    EmailEventDeferred     EmailEventType = "deferred"     // temp failure, provider retrying
)

// Unsubscribe tracks email opt-outs at multiple levels.
type Unsubscribe struct {
    ID         uuid.UUID
    TenantID   uuid.UUID
    Email      string
    Scope      UnsubscribeScope
    CampaignID *uuid.UUID      // only if scope is "campaign"
    Reason     string           // "manual", "link_click", "complaint", "bounce"
    CreatedAt  time.Time
}

type UnsubscribeScope string

const (
    UnsubscribeScopeGlobal   UnsubscribeScope = "global"    // all emails from tenant
    UnsubscribeScopeCampaign UnsubscribeScope = "campaign"   // specific campaign only
    UnsubscribeScopeCategory UnsubscribeScope = "category"   // all marketing, keep transactional
)

// SuppressionEntry — addresses that must NEVER receive email.
// Hard bounces and complaints are auto-added. Cannot be overridden.
type SuppressionEntry struct {
    ID        uuid.UUID
    TenantID  uuid.UUID
    Email     string
    Reason    SuppressionReason
    Source    string            // "bounce_webhook", "complaint_webhook", "manual", "import"
    CreatedAt time.Time
}

type SuppressionReason string

const (
    SuppressionHardBounce SuppressionReason = "hard_bounce"
    SuppressionComplaint  SuppressionReason = "complaint"
    SuppressionManual     SuppressionReason = "manual"       // admin manually suppressed
    SuppressionInvalid    SuppressionReason = "invalid"      // email validation failed
)
```

### 12.3 Cross-module email sending interface

Other Spur modules (identity, billing, etc.) send emails through this interface.
They do NOT interact with email providers directly.

```go
// core/ports/email_sender.go
package ports

// EmailSender is the interface exposed to OTHER modules for sending emails.
// It is available via module.Services.EmailSender
type EmailSender interface {
    // SendTransactional sends a single transactional email (password reset, OTP, invoice, etc.)
    // Uses the platform's default email provider (from env) unless tenant has own config.
    // Transactional emails SKIP unsubscribe checks (but still check suppression list).
    SendTransactional(ctx context.Context, tenantID uuid.UUID, req TransactionalEmailRequest) (*domain.Message, error)

    // SendWithTemplate renders an email template and sends it.
    SendWithTemplate(ctx context.Context, tenantID uuid.UUID, req TemplateEmailRequest) (*domain.Message, error)
}

type TransactionalEmailRequest struct {
    To          string            // recipient email
    Subject     string
    HTMLBody    string            // raw HTML
    TextBody    string            // plain text fallback (optional)
    FromEmail   string            // override default FROM (optional)
    FromName    string            // override default FROM name (optional)
    ReplyTo     string            // optional
    CC          []string          // optional
    BCC         []string          // optional
    Headers     map[string]string // custom headers (optional)
    Attachments []domain.EmailAttachment // optional
    Tags        []string          // for analytics grouping (e.g. "password_reset", "invoice")
    Metadata    map[string]string // custom tracking data
}

type TemplateEmailRequest struct {
    To           string
    TemplateName string            // name of EmailTemplate
    Variables    map[string]string // variable substitutions: {"name": "Dinesh", "order_id": "ORD-123"}
    FromEmail    string            // optional override
    FromName     string            // optional override
    ReplyTo      string            // optional
    CC           []string
    BCC          []string
    Attachments  []domain.EmailAttachment
    Tags         []string
    Metadata     map[string]string
}
```

**Wiring in app.go:**
```go
// Other modules can access email sending via messaging module's services
identityOpts.EmailSender = messagingModule.Services.EmailSender

// Example: identity module sends password reset email
func (s *AuthService) SendPasswordReset(ctx context.Context, tenantID uuid.UUID, email, token string) error {
    return s.emailSender.SendWithTemplate(ctx, tenantID, ports.TemplateEmailRequest{
        To:           email,
        TemplateName: "password_reset",
        Variables:    map[string]string{"reset_link": "https://app.citual.com/reset?token=" + token},
        Tags:         []string{"auth", "password_reset"},
    })
}
```

### 12.4 Email-specific port interfaces

```go
// core/ports/repositories.go (ADD these to the existing file)

type EmailTemplateRepository interface {
    Create(ctx context.Context, tmpl *domain.EmailTemplate) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.EmailTemplate, error)
    GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.EmailTemplate, error)
    List(ctx context.Context, tenantID uuid.UUID, category *domain.EmailCategory, page, perPage int) ([]domain.EmailTemplate, int, error)
    Update(ctx context.Context, tmpl *domain.EmailTemplate) error
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type EmailEventRepository interface {
    Create(ctx context.Context, event *domain.EmailEvent) error
    CreateBatch(ctx context.Context, events []domain.EmailEvent) error
    GetByMessageID(ctx context.Context, tenantID, messageID uuid.UUID) ([]domain.EmailEvent, error)
    GetByCampaignID(ctx context.Context, tenantID, campaignID uuid.UUID, eventType *domain.EmailEventType, page, perPage int) ([]domain.EmailEvent, int, error)
    ExistsByProviderEventID(ctx context.Context, providerEventID string) (bool, error) // dedup
    GetStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*domain.EmailStats, error)
    GetCampaignStats(ctx context.Context, tenantID, campaignID uuid.UUID) (*domain.EmailCampaignStats, error)
}

type UnsubscribeRepository interface {
    Create(ctx context.Context, unsub *domain.Unsubscribe) error
    IsUnsubscribed(ctx context.Context, tenantID uuid.UUID, email string, scope domain.UnsubscribeScope, campaignID *uuid.UUID) (bool, error)
    List(ctx context.Context, tenantID uuid.UUID, scope *domain.UnsubscribeScope, page, perPage int) ([]domain.Unsubscribe, int, error)
    Delete(ctx context.Context, tenantID, id uuid.UUID) error // re-subscribe
    GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) ([]domain.Unsubscribe, error)
}

type SuppressionRepository interface {
    Create(ctx context.Context, entry *domain.SuppressionEntry) error
    IsSuppressed(ctx context.Context, tenantID uuid.UUID, email string) (bool, error)
    List(ctx context.Context, tenantID uuid.UUID, reason *domain.SuppressionReason, page, perPage int) ([]domain.SuppressionEntry, int, error)
    Delete(ctx context.Context, tenantID, id uuid.UUID) error // remove from suppression (admin only)
    BulkCheck(ctx context.Context, tenantID uuid.UUID, emails []string) ([]string, error) // returns suppressed emails from list
}
```

```go
// core/ports/services.go (ADD these to the existing file)

type EmailTemplateService interface {
    Create(ctx context.Context, tenantID uuid.UUID, req CreateEmailTemplateRequest) (*domain.EmailTemplate, error)
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.EmailTemplate, error)
    List(ctx context.Context, tenantID uuid.UUID, category *domain.EmailCategory, page, perPage int) ([]domain.EmailTemplate, int, error)
    Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateEmailTemplateRequest) (*domain.EmailTemplate, error)
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
    Preview(ctx context.Context, tenantID, id uuid.UUID, variables map[string]string) (*EmailPreview, error)
    Duplicate(ctx context.Context, tenantID, id uuid.UUID, newName string) (*domain.EmailTemplate, error)
}

type CreateEmailTemplateRequest struct {
    Name        string
    Subject     string
    PreviewText string
    HTMLBody    string
    TextBody    string            // auto-generated from HTML if empty
    Category    domain.EmailCategory
    Variables   []string          // e.g. ["name", "order_id", "amount"]
}

type UpdateEmailTemplateRequest struct {
    Subject     *string
    PreviewText *string
    HTMLBody    *string
    TextBody    *string
    Category    *domain.EmailCategory
    Variables   *[]string
    IsActive    *bool
}

type EmailPreview struct {
    Subject  string // rendered with variables
    HTMLBody string // rendered with variables
    TextBody string
}

type EmailAnalyticsService interface {
    GetOverview(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*domain.EmailStats, error)
    GetCampaignReport(ctx context.Context, tenantID, campaignID uuid.UUID) (*domain.EmailCampaignStats, error)
    GetDomainReputation(ctx context.Context, tenantID uuid.UUID) (*domain.DomainReputation, error)
    GetTopLinks(ctx context.Context, tenantID, campaignID uuid.UUID, limit int) ([]domain.LinkStats, error)
    GetBounceReport(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*domain.BounceReport, error)
    GetEngagementByHour(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]domain.HourlyEngagement, error)
}

type UnsubscribeService interface {
    Unsubscribe(ctx context.Context, tenantID uuid.UUID, email string, scope domain.UnsubscribeScope, campaignID *uuid.UUID, reason string) error
    Resubscribe(ctx context.Context, tenantID, id uuid.UUID) error
    IsUnsubscribed(ctx context.Context, tenantID uuid.UUID, email string) (bool, error)
    List(ctx context.Context, tenantID uuid.UUID, scope *domain.UnsubscribeScope, page, perPage int) ([]domain.Unsubscribe, int, error)
    // HandleUnsubscribeWebhook is called from the public unsubscribe endpoint
    HandleUnsubscribeLink(ctx context.Context, token string) error
}

type SuppressionService interface {
    IsSuppressed(ctx context.Context, tenantID uuid.UUID, email string) (bool, error)
    AddToSuppression(ctx context.Context, tenantID uuid.UUID, email string, reason domain.SuppressionReason) error
    RemoveFromSuppression(ctx context.Context, tenantID, id uuid.UUID) error
    List(ctx context.Context, tenantID uuid.UUID, reason *domain.SuppressionReason, page, perPage int) ([]domain.SuppressionEntry, int, error)
    BulkCheck(ctx context.Context, tenantID uuid.UUID, emails []string) ([]string, error)
}
```

### 12.5 Email analytics domain types

```go
// core/domain/analytics.go (ADD these)

type EmailStats struct {
    TotalSent     int     `json:"total_sent"`
    Delivered     int     `json:"delivered"`
    DeliveryRate  float64 `json:"delivery_rate"`   // delivered / sent * 100
    Opens         int     `json:"opens"`
    UniqueOpens   int     `json:"unique_opens"`
    OpenRate      float64 `json:"open_rate"`       // unique_opens / delivered * 100
    Clicks        int     `json:"clicks"`
    UniqueClicks  int     `json:"unique_clicks"`
    ClickRate     float64 `json:"click_rate"`      // unique_clicks / delivered * 100
    Bounces       int     `json:"bounces"`
    HardBounces   int     `json:"hard_bounces"`
    SoftBounces   int     `json:"soft_bounces"`
    BounceRate    float64 `json:"bounce_rate"`     // bounces / sent * 100
    Complaints    int     `json:"complaints"`
    ComplaintRate float64 `json:"complaint_rate"`  // complaints / delivered * 100
    Unsubscribes  int     `json:"unsubscribes"`
    UnsubRate     float64 `json:"unsub_rate"`      // unsubscribes / delivered * 100
}

type EmailCampaignStats struct {
    CampaignID    uuid.UUID  `json:"campaign_id"`
    EmailStats                              // embedded
    TotalRecipients int      `json:"total_recipients"`
    Suppressed      int      `json:"suppressed"`       // blocked by suppression list
    Unsubscribed    int      `json:"unsubscribed"`      // blocked by unsubscribe list
    TopLinks        []LinkStats `json:"top_links"`
    OpensByHour     []HourlyEngagement `json:"opens_by_hour"`
    ClicksByHour    []HourlyEngagement `json:"clicks_by_hour"`
}

type LinkStats struct {
    URL          string `json:"url"`
    TotalClicks  int    `json:"total_clicks"`
    UniqueClicks int    `json:"unique_clicks"`
}

type HourlyEngagement struct {
    Hour  int `json:"hour"`  // 0-23
    Count int `json:"count"`
}

type DomainReputation struct {
    BounceRate    float64 `json:"bounce_rate_30d"`    // last 30 days
    ComplaintRate float64 `json:"complaint_rate_30d"` // last 30 days
    HealthStatus  string  `json:"health_status"`      // "good", "warning", "critical"
    // "good": bounce < 2%, complaint < 0.1%
    // "warning": bounce 2-5% or complaint 0.1-0.3%
    // "critical": bounce > 5% or complaint > 0.3%
}

type BounceReport struct {
    HardBounces    int               `json:"hard_bounces"`
    SoftBounces    int               `json:"soft_bounces"`
    TopReasons     []BounceReasonStat `json:"top_reasons"`
    TopDomains     []BounceDomainStat `json:"top_domains"`
}

type BounceReasonStat struct {
    Reason string `json:"reason"`
    Count  int    `json:"count"`
}

type BounceDomainStat struct {
    Domain      string  `json:"domain"`
    BounceRate  float64 `json:"bounce_rate"`
    TotalSent   int     `json:"total_sent"`
    TotalBounce int     `json:"total_bounce"`
}
```

### 12.6 Email provider adapter interface

The email provider adapter extends the base `Provider` interface with email-specific methods.

```go
// core/ports/provider.go (ADD this alongside existing Provider interface)

// EmailProvider extends Provider with email-specific capabilities.
// Implemented by sendgrid.go, mailgun.go, postmark.go
type EmailProvider interface {
    Provider // inherits Send, ParseWebhook, ValidateWebhookSignature

    // SendEmail sends a fully formed email with all email-specific fields.
    // This is called by the email service instead of the generic Send().
    SendEmail(ctx context.Context, cfg *domain.ProviderConfig, req EmailSendRequest) (*ProviderSendResult, error)

    // SendBatch sends multiple emails in a single API call (if provider supports it).
    // SendGrid supports up to 1000 per batch. Mailgun/Postmark have their own limits.
    SendBatch(ctx context.Context, cfg *domain.ProviderConfig, reqs []EmailSendRequest) ([]ProviderSendResult, error)
}

type EmailSendRequest struct {
    To          string
    CC          []string
    BCC         []string
    FromEmail   string
    FromName    string
    ReplyTo     string
    Subject     string
    HTMLBody    string
    TextBody    string
    Headers     map[string]string     // List-Unsubscribe, X-Custom-Header, etc.
    Attachments []domain.EmailAttachment
    Tags        []string              // provider tags for filtering/analytics
    Category    domain.EmailCategory  // affects sending priority
    TrackOpens  bool
    TrackClicks bool
    Metadata    map[string]string     // custom args passed to provider (returned in webhooks)
    IPPool      string                // SendGrid IP pool name (optional)
}
```

### 12.7 Email provider API reference

**SendGrid v3 API:**
```
Base URL: https://api.sendgrid.com/v3
Auth: Authorization: Bearer SG.xxxx

POST /mail/send
{
  "personalizations": [{"to": [{"email": "user@example.com"}]}],
  "from": {"email": "noreply@citual.com", "name": "Citual"},
  "subject": "Your order is confirmed",
  "content": [
    {"type": "text/plain", "value": "..."},
    {"type": "text/html", "value": "<html>...</html>"}
  ],
  "tracking_settings": {
    "open_tracking": {"enable": true},
    "click_tracking": {"enable": true}
  },
  "custom_args": {"message_id": "uuid", "tenant_id": "uuid"}
}

Webhook events: delivered, bounce, deferred, dropped, open, click, unsubscribe, spamreport
Signature: X-Twilio-Email-Event-Webhook-Signature (ECDSA)
```

**Mailgun API:**
```
Base URL: https://api.mailgun.net/v3
Auth: Basic api:key-xxxx

POST /{domain}/messages
Form data:
  from=Citual <noreply@mg.citual.com>
  to=user@example.com
  subject=Your order is confirmed
  html=<html>...</html>
  o:tracking-opens=yes
  o:tracking-clicks=htmlonly
  v:message_id=uuid
  v:tenant_id=uuid

Webhook events: delivered, failed (permanent=bounce, temporary=soft_bounce), opened, clicked, unsubscribed, complained
Signature: timestamp + token + signing key → HMAC-SHA256
```

**Postmark API:**
```
Base URL: https://api.postmarkapp.com
Auth: X-Postmark-Server-Token: xxxx

POST /email
{
  "From": "noreply@citual.com",
  "To": "user@example.com",
  "Subject": "Your order is confirmed",
  "HtmlBody": "<html>...</html>",
  "TextBody": "...",
  "TrackOpens": true,
  "TrackLinks": "HtmlAndText",
  "Metadata": {"message_id": "uuid", "tenant_id": "uuid"}
}

Webhook events: Delivery, Bounce, SpamComplaint, Open, Click, SubscriptionChange
Signature: No built-in signing — use webhook password or IP whitelisting
```

### 12.8 Database migration: 004_email_system.up.sql

```sql
-- sql/migrations/004_email_system.up.sql

-- HTML email templates (separate from WhatsApp templates)
CREATE TABLE messaging.email_templates (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    name          TEXT NOT NULL,
    subject       TEXT NOT NULL,
    preview_text  TEXT NOT NULL DEFAULT '',
    html_body     TEXT NOT NULL,
    text_body     TEXT NOT NULL DEFAULT '',    -- auto-generated from HTML if empty
    category      TEXT NOT NULL DEFAULT 'transactional' CHECK (category IN ('transactional', 'marketing', 'notification')),
    variables     TEXT[] NOT NULL DEFAULT '{}', -- variable names used in template
    is_active     BOOLEAN NOT NULL DEFAULT true,
    version       INT NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

-- Email lifecycle events (open, click, bounce, complaint, etc.)
-- High-volume table — partitioned by month if needed at scale
CREATE TABLE messaging.email_events (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    message_id        UUID NOT NULL REFERENCES messaging.messages(id),
    campaign_id       UUID,
    event_type        TEXT NOT NULL CHECK (event_type IN (
        'delivered', 'bounce', 'soft_bounce', 'open', 'click',
        'unsubscribe', 'complaint', 'dropped', 'deferred'
    )),
    recipient         TEXT NOT NULL,
    timestamp         TIMESTAMPTZ NOT NULL,
    provider_event_id TEXT,                   -- for dedup
    user_agent        TEXT,
    ip_address        TEXT,
    url               TEXT,                   -- clicked URL
    bounce_type       TEXT,                   -- 'hard' or 'soft'
    bounce_reason     TEXT,                   -- SMTP error
    complaint_feedback TEXT,
    raw_payload       JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_email_events_provider_id ON messaging.email_events (provider_event_id) WHERE provider_event_id IS NOT NULL;
CREATE INDEX idx_email_events_message ON messaging.email_events (tenant_id, message_id);
CREATE INDEX idx_email_events_campaign ON messaging.email_events (tenant_id, campaign_id) WHERE campaign_id IS NOT NULL;
CREATE INDEX idx_email_events_type_time ON messaging.email_events (tenant_id, event_type, timestamp DESC);

-- Unsubscribe list (multi-level: global, category, campaign)
CREATE TABLE messaging.unsubscribes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    email       TEXT NOT NULL,
    scope       TEXT NOT NULL CHECK (scope IN ('global', 'category', 'campaign')),
    campaign_id UUID,                  -- only if scope='campaign'
    reason      TEXT NOT NULL DEFAULT 'manual', -- 'manual', 'link_click', 'complaint', 'bounce'
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_unsubscribes_tenant_email_scope ON messaging.unsubscribes (tenant_id, email, scope, COALESCE(campaign_id, '00000000-0000-0000-0000-000000000000'::uuid));
CREATE INDEX idx_unsubscribes_tenant_email ON messaging.unsubscribes (tenant_id, email);

-- Suppression list (hard bounces, complaints — NEVER send to these)
CREATE TABLE messaging.suppressions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    email       TEXT NOT NULL,
    reason      TEXT NOT NULL CHECK (reason IN ('hard_bounce', 'complaint', 'manual', 'invalid')),
    source      TEXT NOT NULL DEFAULT 'manual', -- 'bounce_webhook', 'complaint_webhook', 'manual', 'import'
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, email)
);

CREATE INDEX idx_suppressions_tenant_email ON messaging.suppressions (tenant_id, email);

-- RLS
ALTER TABLE messaging.email_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.email_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.unsubscribes ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.suppressions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_email_templates ON messaging.email_templates
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_email_events ON messaging.email_events
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_unsubscribes ON messaging.unsubscribes
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_suppressions ON messaging.suppressions
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### 12.9 Email-specific HTTP routes

Add these to the route map in Section 7. All under `/messaging/` prefix, auth middleware applied.

```
# Email Templates                                         PERMISSION
POST   /messaging/email-templates                → CreateEmailTemplate    messaging:templates:write
GET    /messaging/email-templates                → ListEmailTemplates     messaging:templates:read
GET    /messaging/email-templates/{id}           → GetEmailTemplate       messaging:templates:read
PUT    /messaging/email-templates/{id}           → UpdateEmailTemplate    messaging:templates:write
DELETE /messaging/email-templates/{id}           → DeleteEmailTemplate    messaging:templates:write
POST   /messaging/email-templates/{id}/preview   → PreviewEmailTemplate   messaging:templates:read
POST   /messaging/email-templates/{id}/duplicate → DuplicateTemplate      messaging:templates:write
POST   /messaging/email-templates/{id}/send-test → SendTestEmail          messaging:messages:send

# Email Send (single)                                     PERMISSION
POST   /messaging/email/send                     → SendEmail              messaging:messages:send
POST   /messaging/email/send-template            → SendEmailWithTemplate  messaging:messages:send

# Email Campaigns (extend existing campaign routes — campaign.channel = "email")
# Use existing campaign routes: POST /messaging/campaigns with channel="email"
# The campaign engine handles email-specific logic (suppression check, unsubscribe link injection)

# Unsubscribe Management                                  PERMISSION
GET    /messaging/unsubscribes                   → ListUnsubscribes       messaging:contacts:read
DELETE /messaging/unsubscribes/{id}              → Resubscribe            messaging:contacts:manage_consent
GET    /messaging/unsubscribes/check             → CheckUnsubscribe       messaging:contacts:read
        ?email=user@example.com

# Suppression List                                        PERMISSION
GET    /messaging/suppressions                   → ListSuppressions       messaging:contacts:read
POST   /messaging/suppressions                   → AddToSuppression       messaging:contacts:manage_consent
DELETE /messaging/suppressions/{id}              → RemoveFromSuppression  messaging:contacts:manage_consent
POST   /messaging/suppressions/check             → BulkCheckSuppression   messaging:contacts:read
        body: {"emails": ["a@b.com", "c@d.com"]}

# Email Analytics                                         PERMISSION
GET    /messaging/analytics/email                → EmailOverviewStats     messaging:analytics:read
        ?from=&to=
GET    /messaging/analytics/email/campaigns/{id} → EmailCampaignReport    messaging:analytics:read
GET    /messaging/analytics/email/bounces        → BounceReport           messaging:analytics:read
        ?from=&to=
GET    /messaging/analytics/email/reputation     → DomainReputation       messaging:analytics:read
GET    /messaging/analytics/email/links/{campaignId} → TopLinksReport     messaging:analytics:read
GET    /messaging/analytics/email/engagement     → EngagementByHour       messaging:analytics:read
        ?from=&to=

# Public endpoints (NO auth — mounted by app.go outside auth group)
# Unsubscribe link handler — user clicks unsubscribe link in email
GET    /messaging/unsubscribe/{token}            → HandleUnsubscribeLink  (public, token-verified)
# Webhook endpoints for email providers
POST   /messaging/webhook/email/sendgrid         → WebhookHandler.HandleSendGrid
POST   /messaging/webhook/email/mailgun          → WebhookHandler.HandleMailgun
POST   /messaging/webhook/email/postmark         → WebhookHandler.HandlePostmark
```

### 12.10 Email send flow (critical business logic)

The email service MUST enforce this sequence before sending:

```
1. Resolve provider config (tenant's own → platform default)
2. CHECK suppression list — if suppressed, BLOCK send, log as "dropped"
3. CHECK unsubscribe list — if unsubscribed AND email is marketing, BLOCK send
   (transactional emails SKIP unsubscribe check but NOT suppression check)
4. Render template (if template-based): substitute variables in subject + body
5. Inject unsubscribe link — for marketing emails, add:
   - List-Unsubscribe header: <mailto:unsub@citual.com>, <https://api.citual.com/messaging/unsubscribe/{token}>
   - List-Unsubscribe-Post header: List-Unsubscribe=One-Click
   - Visible unsubscribe link in HTML footer
6. Inject tracking pixel (if track_opens=true): <img src="https://api.citual.com/messaging/track/open/{message_id}" width="1" height="1">
7. Rewrite links for click tracking (if track_clicks=true): original URL → https://api.citual.com/messaging/track/click/{message_id}/{link_id}?url=original
8. Enqueue to Redis Streams (same queue as WhatsApp/SMS, channel="email")
9. Worker dequeues → calls provider.SendEmail() → updates message status
```

### 12.11 Webhook processing (auto-suppression)

When email provider webhooks arrive, the system MUST auto-maintain the suppression list:

```
EVENT              ACTION
─────────────────────────────────────────────────────────
hard_bounce    →   Auto-add to suppression list (reason: hard_bounce)
complaint      →   Auto-add to suppression list (reason: complaint)
                   Auto-add to unsubscribe list (scope: global, reason: complaint)
unsubscribe    →   Auto-add to unsubscribe list (scope determined by link token)
soft_bounce    →   Track count — after 3 soft bounces to same address within 72hrs,
                   treat as hard bounce and add to suppression
delivered      →   Update message status to "delivered"
open           →   Create email_event, update first_opened_at on message
click          →   Create email_event with URL clicked
deferred       →   Log event, no action (provider is retrying)
dropped        →   Update message status to "failed"
```

### 12.12 Complete email feature list

| Feature | Description | Priority |
|---|---|---|
| **Sending** | | |
| Single email send | Send one email via API (raw HTML or template) | P0 |
| Template-based send | Render email template with variables and send | P0 |
| Bulk email send | Send to multiple recipients via campaign | P0 |
| Cross-module sending | Other modules send transactional emails via EmailSender port | P0 |
| Multi-provider support | SendGrid, Mailgun, Postmark — selected via env var | P0 |
| Per-tenant provider override | Tenant brings own API key, overrides platform default | P1 |
| Attachment support | PDF, images, etc. as base64-encoded attachments | P1 |
| CC/BCC support | Carbon copy and blind carbon copy recipients | P0 |
| Custom headers | X-Custom-Header, Reply-To, etc. | P1 |
| **Templates** | | |
| Email template CRUD | Create, update, delete HTML email templates | P0 |
| Variable substitution | {{name}}, {{order_id}} replaced at send time | P0 |
| Template preview | Render template with sample variables and return HTML | P0 |
| Template versioning | Auto-increment version on update, audit trail | P1 |
| Template duplication | Clone existing template with new name | P0 |
| Send test email | Send template to a test address before campaign | P0 |
| Plain text auto-generation | Strip HTML tags to create text/plain fallback | P0 |
| Template categories | transactional, marketing, notification — affects sending rules | P0 |
| **Campaigns** | | |
| Email campaign creation | Create campaign: select template, audience, schedule | P0 |
| Excel/CSV audience upload | Upload recipient list with email + variable columns | P0 |
| Segment targeting | Send to dynamic/static segments from contact module | P0 |
| Scheduled sending | Send at a specific date/time (timezone-aware) | P0 |
| Campaign pause/resume | Pause mid-send, resume from where it stopped | P0 |
| Pre-send validation | Check suppression, unsubscribe, invalid emails before sending | P0 |
| Campaign-level analytics | Real-time stats: sent, delivered, opened, clicked, bounced | P0 |
| A/B testing | Test 2 subject lines or bodies, auto-pick winner | P2 |
| **Tracking & Analytics** | | |
| Open tracking | Tracking pixel injected in HTML emails | P0 |
| Click tracking | Link rewriting for click-through tracking | P0 |
| Delivery tracking | Webhook-based delivery confirmation from provider | P0 |
| Bounce tracking | Hard bounce, soft bounce with SMTP error reasons | P0 |
| Complaint tracking | ISP spam complaint / feedback loop processing | P0 |
| Campaign reports | Delivery rate, open rate, click rate, bounce rate, unsub rate | P0 |
| Link click reports | Top clicked links per campaign with unique/total counts | P1 |
| Engagement by hour | Open/click distribution by hour of day | P1 |
| Domain reputation | 30-day bounce rate + complaint rate → health status | P1 |
| Bounce report | Top bounce reasons, top bouncing domains | P1 |
| Per-message event timeline | Full lifecycle: queued → sent → delivered → opened → clicked | P0 |
| **Compliance** | | |
| One-click unsubscribe | List-Unsubscribe + List-Unsubscribe-Post headers (RFC 8058) | P0 |
| Visible unsubscribe link | Footer link in every marketing email | P0 |
| Global unsubscribe | Opt out of all emails from tenant | P0 |
| Category unsubscribe | Opt out of marketing, keep transactional | P1 |
| Campaign unsubscribe | Opt out of specific campaign/list only | P1 |
| Suppression list | Auto-maintained: hard bounces + complaints never receive email | P0 |
| Suppression check before send | Every email send checks suppression list first | P0 |
| Re-subscribe flow | Admin can remove from unsubscribe list (not suppression) | P1 |
| CAN-SPAM compliance | Physical address in footer, unsubscribe within 10 days | P0 |
| GDPR compliance | Data export, deletion on request (via contact module) | P1 |
| **Deliverability** | | |
| SPF/DKIM/DMARC guidance | API endpoint returns required DNS records for tenant's domain | P1 |
| Bounce handling automation | 3 soft bounces → auto-suppress as hard bounce | P0 |
| Email validation | Basic format + MX record check before send | P1 |
| Warmup guidance | Documentation on IP/domain warmup for new senders | P2 |
| Dedicated IP support | SendGrid IP pool selection per tenant (enterprise) | P3 |

### 12.13 SMS feature list (same module)

| Feature | Description | Priority |
|---|---|---|
| **Sending** | | |
| Single SMS send | Send one SMS via API | P0 |
| Bulk SMS send | Send to segment or uploaded list | P0 |
| Template-based SMS | DLT-registered templates (India requirement) | P0 |
| OTP sending | Priority queue, auto-expiry tracking | P0 |
| Multi-provider | MSG91 (India), Twilio (international) | P0 |
| Unicode/long SMS | Auto-detect and handle multi-part messages | P0 |
| **Compliance (India)** | | |
| DLT registration | Store DLT template ID, entity ID, sender ID | P0 |
| DND check | Check NCPR/DND registry before promotional SMS | P1 |
| Sender ID management | Transactional vs promotional sender IDs | P0 |
| Opt-in/out | STOP/START keyword handling via webhooks | P0 |
| **Analytics** | | |
| Delivery reports | Sent, delivered, failed with error codes | P0 |
| Campaign analytics | Per-campaign delivery stats | P0 |
| Cost tracking | Per-SMS cost from provider | P1 |

### 12.14 Update to Services struct

```go
// module.go — Updated Services struct

type Services struct {
    // Existing
    MessageService  ports.MessageService
    TemplateService ports.TemplateService
    CampaignService ports.CampaignService
    ContactService  ports.ContactService

    // Email-specific
    EmailSender           ports.EmailSender            // Cross-module interface
    EmailTemplateService  ports.EmailTemplateService
    EmailAnalyticsService ports.EmailAnalyticsService
    UnsubscribeService    ports.UnsubscribeService
    SuppressionService    ports.SuppressionService
}
```

---

## 13. CONFIG RESOLUTION (3-TIER OVERRIDE PATTERN)

All configurable values (from_email, from_name, track_opens, track_clicks, sender_id, etc.)
follow a 3-tier resolution hierarchy. The most specific value wins.

```
Tier 1: API request fields         → highest priority (per-message override)
Tier 2: Tenant's provider_config   → middle priority (per-tenant default)
Tier 3: Platform .env defaults     → lowest priority (fallback)
```

### 13.1 Resolution helper (MUST be used by all services)

```go
// core/services/resolve.go
package services

// resolveString returns the first non-empty value in priority order:
// API request → tenant provider_config → platform default
func resolveString(apiValue, tenantValue, platformDefault string) string {
    if apiValue != "" {
        return apiValue
    }
    if tenantValue != "" {
        return tenantValue
    }
    return platformDefault
}

// resolveBool returns the first non-nil value in priority order.
// If all are nil, returns platformDefault.
func resolveBool(apiValue *bool, tenantValue *bool, platformDefault bool) bool {
    if apiValue != nil {
        return *apiValue
    }
    if tenantValue != nil {
        return *tenantValue
    }
    return platformDefault
}

// resolveInt returns the first non-zero value in priority order.
func resolveInt(apiValue, tenantValue, platformDefault int) int {
    if apiValue > 0 {
        return apiValue
    }
    if tenantValue > 0 {
        return tenantValue
    }
    return platformDefault
}
```

### 13.2 Where each tier is stored

```
TIER      SOURCE                    EXAMPLE FIELDS
──────────────────────────────────────────────────────────────────
Tier 3    Config struct (.env)      EmailFromAddress, EmailFromName,
(platform)                          EmailTrackOpens, EmailTrackClicks,
                                    SMSSenderID

Tier 2    provider_configs table    from_email, from_name, reply_to_email
(tenant)  (per-tenant, per-channel) (stored alongside encrypted credentials)

Tier 1    API request body          from_email, from_name, reply_to,
(message) (per-send)                track_opens, track_clicks, cc, bcc
```

### 13.3 Email send resolution example

```go
// core/services/message_service.go — inside the email send flow

func (s *messageService) resolveEmailParams(
    req ports.SendMessageRequest,      // Tier 1: API request
    cfg *domain.ProviderConfig,        // Tier 2: tenant config
    platformCfg Config,                // Tier 3: platform defaults
) resolvedEmailParams {
    return resolvedEmailParams{
        FromEmail:   resolveString(req.FromEmail, cfg.FromEmail, platformCfg.EmailFromAddress),
        FromName:    resolveString(req.FromName, cfg.FromName, platformCfg.EmailFromName),
        ReplyTo:     resolveString(req.ReplyTo, cfg.ReplyToEmail, ""),
        TrackOpens:  resolveBool(req.TrackOpens, cfg.TrackOpens, platformCfg.EmailTrackOpens),
        TrackClicks: resolveBool(req.TrackClicks, cfg.TrackClicks, platformCfg.EmailTrackClicks),
    }
}
```

### 13.4 SMS send resolution example

```go
func (s *messageService) resolveSMSParams(
    req ports.SendMessageRequest,
    cfg *domain.ProviderConfig,
    platformCfg Config,
) resolvedSMSParams {
    return resolvedSMSParams{
        SenderID: resolveString(req.SenderID, cfg.SMSSenderID, platformCfg.SMSSenderID),
    }
}
```

### 13.5 WhatsApp — no resolution needed

WhatsApp does not have configurable from/sender fields. The sender is always the
WhatsApp Business number registered with Meta. The phone_number_id in the tenant's
provider_config determines which number sends. No env var, no per-request override.

### 13.6 Fields that support 3-tier resolution

| Field | API Request Param | provider_configs Column | .env Default |
|---|---|---|---|
| from_email | `from_email` | `from_email` | `MESSAGING_EMAIL_FROM_ADDRESS` |
| from_name | `from_name` | `from_name` | `MESSAGING_EMAIL_FROM_NAME` |
| reply_to | `reply_to` | `reply_to_email` | *(none — optional)* |
| track_opens | `track_opens` | *(stored in credentials JSON)* | `MESSAGING_EMAIL_TRACK_OPENS` |
| track_clicks | `track_clicks` | *(stored in credentials JSON)* | `MESSAGING_EMAIL_TRACK_CLICKS` |
| sms_sender_id | `sender_id` | *(stored in credentials JSON)* | `MESSAGING_SMS_SENDER_ID` |

### 13.7 Update to SendMessageRequest (ports)

The existing `SendMessageRequest` in `core/ports/services.go` needs these optional fields
for per-request overrides:

```go
type SendMessageRequest struct {
    Channel          domain.Channel
    Recipient        string
    MessageType      domain.MessageType
    TemplateName     *string
    TemplateLanguage *string
    TemplateParams   map[string]string
    Text             *string
    MediaURL         *string
    MediaType        *string
    Metadata         map[string]string

    // Per-request overrides (Tier 1) — all optional
    // If empty, falls back to tenant config (Tier 2) then platform default (Tier 3)
    FromEmail    string  // Email: override sender address
    FromName     string  // Email: override sender name
    ReplyTo      string  // Email: override reply-to
    TrackOpens   *bool   // Email: override open tracking (pointer = nil means "use default")
    TrackClicks  *bool   // Email: override click tracking
    SenderID     string  // SMS: override sender ID
    CC           []string // Email only
    BCC          []string // Email only
}
```

Note: `TrackOpens` and `TrackClicks` are `*bool` (pointer) so nil = "not specified, use default"
vs false = "explicitly disabled by the caller".
