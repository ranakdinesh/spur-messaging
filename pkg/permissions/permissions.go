// Package permissions defines the public permission keys exposed by the
// messaging module. Keep these keys stable because Identity, API keys, and
// frontend route guards all depend on them.
package permissions

const (
	ModuleCode = "spur-messaging"

	AnalyticsRead = "messaging:analytics:read"

	ProvidersRead  = "messaging:providers:read"
	ProvidersWrite = "messaging:providers:write"
	ProvidersTest  = "messaging:providers:test"

	TemplatesRead   = "messaging:templates:read"
	TemplatesWrite  = "messaging:templates:write"
	TemplatesSubmit = "messaging:templates:submit"

	ContactsRead          = "messaging:contacts:read"
	ContactsWrite         = "messaging:contacts:write"
	ContactsImport        = "messaging:contacts:import"
	ContactsManageConsent = "messaging:contacts:manage_consent"

	SegmentsRead  = "messaging:segments:read"
	SegmentsWrite = "messaging:segments:write"

	CampaignsRead    = "messaging:campaigns:read"
	CampaignsWrite   = "messaging:campaigns:write"
	CampaignsExecute = "messaging:campaigns:execute"

	MessagesRead     = "messaging:messages:read"
	MessagesSend     = "messaging:messages:send"
	MessagesSendBulk = "messaging:messages:send_bulk"

	ConversationsRead   = "messaging:conversations:read"
	ConversationsWrite  = "messaging:conversations:write"
	ConversationsAssign = "messaging:conversations:assign"

	WebhooksRead   = "messaging:webhooks:read"
	WebhooksWrite  = "messaging:webhooks:write"
	WebhooksTest   = "messaging:webhooks:test"
	WebhooksReplay = "messaging:webhooks:replay"

	BillingRead   = "messaging:billing:read"
	BillingManage = "messaging:billing:manage"
)

// Permission describes a permission that can be synced into Identity.
type Permission struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

// RoleTemplate describes a default tenant role that Identity can instantiate
// from the messaging module manifest.
type RoleTemplate struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// Catalog is the complete permission list for this module.
var Catalog = []Permission{
	{Key: AnalyticsRead, Description: "Read messaging dashboards, campaign reports, and delivery analytics."},
	{Key: ProvidersRead, Description: "Read messaging provider configuration."},
	{Key: ProvidersWrite, Description: "Create and update messaging provider credentials."},
	{Key: ProvidersTest, Description: "Send provider test messages."},
	{Key: TemplatesRead, Description: "Read WhatsApp and email templates."},
	{Key: TemplatesWrite, Description: "Create and update WhatsApp and email templates."},
	{Key: TemplatesSubmit, Description: "Submit WhatsApp templates for provider approval and sync review state."},
	{Key: ContactsRead, Description: "Read contacts, segments, consent, unsubscribe, and suppression data."},
	{Key: ContactsWrite, Description: "Create, update, and delete contacts."},
	{Key: ContactsImport, Description: "Import contacts in bulk."},
	{Key: ContactsManageConsent, Description: "Manage opt-in, opt-out, unsubscribe, and suppression state."},
	{Key: SegmentsRead, Description: "Read audience segments and resolved segment contacts."},
	{Key: SegmentsWrite, Description: "Create, update, and delete audience segments."},
	{Key: CampaignsRead, Description: "Read campaign lists, status, and reporting."},
	{Key: CampaignsWrite, Description: "Create and update campaigns."},
	{Key: CampaignsExecute, Description: "Execute, pause, resume, and manage campaign runs."},
	{Key: MessagesRead, Description: "Read message delivery logs."},
	{Key: MessagesSend, Description: "Send individual messages and test sends."},
	{Key: MessagesSendBulk, Description: "Send bulk messages."},
	{Key: ConversationsRead, Description: "Read conversation inbox threads, notes, assignment, and SLA state."},
	{Key: ConversationsWrite, Description: "Update conversation status, handoff state, tags, notes, priority, and SLA fields."},
	{Key: ConversationsAssign, Description: "Assign conversations to agents or teams."},
	{Key: WebhooksRead, Description: "Read tenant webhook endpoints and delivery logs."},
	{Key: WebhooksWrite, Description: "Create, update, disable, and delete tenant webhook endpoints."},
	{Key: WebhooksTest, Description: "Send test events to tenant webhook endpoints."},
	{Key: WebhooksReplay, Description: "Replay tenant webhook deliveries."},
	{Key: BillingRead, Description: "Read wallet balances, usage charges, rate cards, and billing ledger entries."},
	{Key: BillingManage, Description: "Create wallet credits, billing adjustments, and rate cards."},
}

// RoleTemplates groups the messaging permissions into practical tenant roles.
// The same permission keys are used as API key scopes.
var RoleTemplates = []RoleTemplate{
	{
		Code:        "messaging_admin",
		Name:        "Messaging Admin",
		Description: "Manage messaging providers, templates, contacts, campaigns, sends, and analytics.",
		Permissions: Keys(),
	},
	{
		Code:        "messaging_developer",
		Name:        "Messaging Developer",
		Description: "Configure providers, manage templates, send test messages, and read delivery logs.",
		Permissions: []string{
			ProvidersRead,
			ProvidersWrite,
			ProvidersTest,
			TemplatesRead,
			TemplatesWrite,
			TemplatesSubmit,
			ContactsRead,
			MessagesRead,
			MessagesSend,
			ConversationsRead,
			ConversationsWrite,
			ConversationsAssign,
			WebhooksRead,
			WebhooksWrite,
			WebhooksTest,
			WebhooksReplay,
			BillingRead,
			AnalyticsRead,
		},
	},
	{
		Code:        "messaging_campaign_manager",
		Name:        "Campaign Manager",
		Description: "Build audiences, manage consent, launch campaigns, and read campaign analytics.",
		Permissions: []string{
			AnalyticsRead,
			TemplatesRead,
			ContactsRead,
			ContactsWrite,
			ContactsImport,
			ContactsManageConsent,
			SegmentsRead,
			SegmentsWrite,
			CampaignsRead,
			CampaignsWrite,
			CampaignsExecute,
			MessagesRead,
			MessagesSendBulk,
			ConversationsRead,
			WebhooksRead,
			BillingRead,
		},
	},
	{
		Code:        "messaging_support_agent",
		Name:        "Support Agent",
		Description: "Read contacts and message history, send replies, and review basic analytics.",
		Permissions: []string{
			AnalyticsRead,
			ContactsRead,
			MessagesRead,
			MessagesSend,
			ConversationsRead,
			ConversationsWrite,
		},
	},
	{
		Code:        "messaging_read_only",
		Name:        "Messaging Read Only",
		Description: "View messaging configuration, templates, contacts, campaigns, messages, and analytics.",
		Permissions: []string{
			AnalyticsRead,
			ProvidersRead,
			TemplatesRead,
			ContactsRead,
			SegmentsRead,
			CampaignsRead,
			MessagesRead,
			ConversationsRead,
			WebhooksRead,
			BillingRead,
		},
	},
	{
		Code:        "messaging_finance",
		Name:        "Finance User",
		Description: "Review usage, wallet balance, billing ledger, and manage billing adjustments.",
		Permissions: []string{
			AnalyticsRead,
			CampaignsRead,
			MessagesRead,
			BillingRead,
			BillingManage,
		},
	},
}

// Keys returns the permission keys in catalog order.
func Keys() []string {
	keys := make([]string, 0, len(Catalog))
	for _, permission := range Catalog {
		keys = append(keys, permission.Key)
	}
	return keys
}
