package http

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router,
	msgH *MessageHandler,
	tmplH *TemplateHandler,
	emailTmplH *EmailTemplateHandler,
	campH *CampaignHandler,
	contactH *ContactHandler,
	convH *ConversationHandler,
	webhookH *TenantWebhookHandler,
	billingH *BillingHandler,
	segmentH *SegmentHandler,
	provH *ProviderHandler,
	unsubH *UnsubscribeHandler,
	suppH *SuppressionHandler,
	analH *AnalyticsHandler) {

	r.Route("/messaging", func(r chi.Router) {
		// Providers
		r.Post("/providers", provH.CreateProvider)         // messaging:providers:write
		r.Get("/providers", provH.ListProviders)           // messaging:providers:read
		r.Get("/providers/{id}", provH.GetProvider)        // messaging:providers:read
		r.Put("/providers/{id}", provH.UpdateProvider)     // messaging:providers:write
		r.Delete("/providers/{id}", provH.DeleteProvider)  // messaging:providers:write
		r.Post("/providers/{id}/test", provH.TestProvider) // messaging:providers:test

		// Tenant webhooks
		r.Post("/webhooks", webhookH.CreateWebhook)                                 // messaging:webhooks:write
		r.Get("/webhooks", webhookH.ListWebhooks)                                   // messaging:webhooks:read
		r.Get("/webhooks/{id}", webhookH.GetWebhook)                                // messaging:webhooks:read
		r.Patch("/webhooks/{id}", webhookH.UpdateWebhook)                           // messaging:webhooks:write
		r.Delete("/webhooks/{id}", webhookH.DeleteWebhook)                          // messaging:webhooks:write
		r.Post("/webhooks/{id}/test", webhookH.TestWebhook)                         // messaging:webhooks:test
		r.Get("/webhooks/{id}/deliveries", webhookH.ListDeliveries)                 // messaging:webhooks:read
		r.Post("/webhooks/deliveries/{deliveryID}/replay", webhookH.ReplayDelivery) // messaging:webhooks:replay

		// Billing / wallet
		r.Get("/wallet/balance", billingH.WalletBalance)         // messaging:billing:read
		r.Post("/wallet/credits", billingH.CreditWallet)         // messaging:billing:manage
		r.Get("/billing/ledger", billingH.Ledger)                // messaging:billing:read
		r.Post("/billing/adjustments", billingH.AdjustWallet)    // messaging:billing:manage
		r.Post("/billing/rate-cards", billingH.CreateRateCard)   // messaging:billing:manage
		r.Get("/billing/estimate", billingH.EstimateMessageCost) // messaging:billing:read
		r.Get("/usage", billingH.Usage)                          // messaging:billing:read

		// Templates (WhatsApp)
		r.Post("/templates", tmplH.CreateTemplate)                // messaging:templates:write
		r.Get("/templates", tmplH.ListTemplates)                  // messaging:templates:read
		r.Get("/templates/{id}", tmplH.GetTemplate)               // messaging:templates:read
		r.Put("/templates/{id}", tmplH.UpdateTemplate)            // messaging:templates:write
		r.Delete("/templates/{id}", tmplH.DeleteTemplate)         // messaging:templates:write
		r.Post("/templates/{id}/submit", tmplH.SubmitForApproval) // messaging:templates:submit
		r.Post("/templates/{id}/sync", tmplH.SyncStatus)          // messaging:templates:read

		// Email Templates
		r.Post("/email-templates", emailTmplH.CreateEmailTemplate)               // messaging:templates:write
		r.Get("/email-templates", emailTmplH.ListEmailTemplates)                 // messaging:templates:read
		r.Get("/email-templates/{id}", emailTmplH.GetEmailTemplate)              // messaging:templates:read
		r.Put("/email-templates/{id}", emailTmplH.UpdateEmailTemplate)           // messaging:templates:write
		r.Delete("/email-templates/{id}", emailTmplH.DeleteEmailTemplate)        // messaging:templates:write
		r.Post("/email-templates/{id}/preview", emailTmplH.PreviewEmailTemplate) // messaging:templates:read
		r.Post("/email-templates/{id}/duplicate", emailTmplH.DuplicateTemplate)  // messaging:templates:write
		r.Post("/email-templates/{id}/send-test", emailTmplH.SendTestEmail)      // messaging:messages:send

		// Messages
		r.Post("/send", msgH.SendMessage)           // messaging:messages:send
		r.Post("/send-bulk", msgH.SendBulkMessages) // messaging:messages:send_bulk
		r.Get("/messages", msgH.ListMessages)       // messaging:messages:read
		r.Get("/messages/{id}", msgH.GetMessage)    // messaging:messages:read

		// Conversations / Inbox
		r.Get("/conversations", convH.ListConversations)         // messaging:conversations:read
		r.Get("/conversations/{id}", convH.GetConversation)      // messaging:conversations:read
		r.Patch("/conversations/{id}", convH.UpdateConversation) // messaging:conversations:write / assign
		r.Post("/conversations/{id}/notes", convH.AddNote)       // messaging:conversations:write

		// Contacts
		r.Post("/contacts", contactH.CreateContact)                    // messaging:contacts:write
		r.Get("/contacts", contactH.ListContacts)                      // messaging:contacts:read
		r.Get("/contacts/{id}", contactH.GetContact)                   // messaging:contacts:read
		r.Put("/contacts/{id}", contactH.UpdateContact)                // messaging:contacts:write
		r.Delete("/contacts/{id}", contactH.DeleteContact)             // messaging:contacts:write
		r.Post("/contacts/import", contactH.BulkImport)                // messaging:contacts:import
		r.Post("/contacts/{id}/opt-in", contactH.OptIn)                // messaging:contacts:manage_consent
		r.Post("/contacts/{id}/confirm-opt-in", contactH.ConfirmOptIn) // messaging:contacts:manage_consent
		r.Post("/contacts/{id}/opt-out", contactH.OptOut)              // messaging:contacts:manage_consent
		r.Get("/contacts/{id}/consents", contactH.ListConsentRecords)  // messaging:contacts:read

		// Segments
		r.Post("/segments", segmentH.CreateSegment)                // messaging:segments:write
		r.Get("/segments", segmentH.ListSegments)                  // messaging:segments:read
		r.Get("/segments/{id}", segmentH.GetSegment)               // messaging:segments:read
		r.Put("/segments/{id}", segmentH.UpdateSegment)            // messaging:segments:write
		r.Delete("/segments/{id}", segmentH.DeleteSegment)         // messaging:segments:write
		r.Get("/segments/{id}/contacts", segmentH.ResolveContacts) // messaging:segments:read

		// Campaigns
		r.Post("/campaigns", campH.CreateCampaign)               // messaging:campaigns:write
		r.Get("/campaigns", campH.ListCampaigns)                 // messaging:campaigns:read
		r.Get("/campaigns/{id}", campH.GetCampaign)              // messaging:campaigns:read
		r.Put("/campaigns/{id}", campH.UpdateCampaign)           // messaging:campaigns:write
		r.Delete("/campaigns/{id}", campH.DeleteCampaign)        // messaging:campaigns:write
		r.Post("/campaigns/{id}/execute", campH.ExecuteCampaign) // messaging:campaigns:execute
		r.Post("/campaigns/{id}/pause", campH.PauseCampaign)     // messaging:campaigns:execute
		r.Post("/campaigns/{id}/resume", campH.ResumeCampaign)   // messaging:campaigns:execute
		r.Get("/campaigns/{id}/stats", campH.GetCampaignStats)   // messaging:campaigns:read

		// Unsubscribes
		r.Get("/unsubscribes", unsubH.ListUnsubscribes)       // messaging:contacts:read
		r.Delete("/unsubscribes/{id}", unsubH.Resubscribe)    // messaging:contacts:manage_consent
		r.Get("/unsubscribes/check", unsubH.CheckUnsubscribe) // messaging:contacts:read

		// Suppression
		r.Get("/suppressions", suppH.ListSuppressions)              // messaging:contacts:read
		r.Post("/suppressions", suppH.AddToSuppression)             // messaging:contacts:manage_consent
		r.Delete("/suppressions/{id}", suppH.RemoveFromSuppression) // messaging:contacts:manage_consent
		r.Post("/suppressions/check", suppH.BulkCheckSuppression)   // messaging:contacts:read

		// Analytics
		r.Get("/analytics/messages", analH.MessageAnalytics)                // messaging:analytics:read
		r.Get("/analytics/overview", analH.DashboardOverview)               // messaging:analytics:read
		r.Get("/analytics/email", analH.EmailOverviewStats)                 // messaging:analytics:read
		r.Get("/analytics/email/campaigns/{id}", analH.EmailCampaignReport) // messaging:analytics:read
		r.Get("/analytics/email/bounces", analH.BounceReport)               // messaging:analytics:read
		r.Get("/analytics/email/reputation", analH.DomainReputation)        // messaging:analytics:read
		r.Get("/analytics/email/links/{id}", analH.TopLinksReport)          // messaging:analytics:read
		r.Get("/analytics/email/engagement", analH.EngagementByHour)        // messaging:analytics:read
	})
}
