# WhatsApp API Coverage Audit

Date: 2026-05-12

Scope: audit of the current `spur-messaging` repository for WhatsApp Tech Provider readiness. This is a code-read audit only; no application code was changed.

Legend:

- `PRESENT`: implemented in a usable way for the current backend.
- `PARTIAL`: important pieces exist, but the feature is incomplete for Tech Provider readiness.
- `MISSING`: no implementation found.
- `UNSAFE`: implementation exists but would be unsafe or misleading in production.
- `UNKNOWN`: not enough evidence in this repository to determine readiness.

## Coverage Matrix

| # | Area | Status | Evidence | Gap / Risk |
|---|---|---|---|---|
| 1 | Provider configuration APIs | PARTIAL | `adapters/http/provider_handler.go`: `CreateProvider`, `ListProviders`, `GetProvider`, `UpdateProvider`, `DeleteProvider`, `TestProvider`; `sql/queries/provider_configs.sql`; `adapters/postgres/store.go`: `CreateProviderConfig`, `GetProviderConfigByWABAID` | CRUD exists and WhatsApp configs require `phone_number_id` and `waba_id`, but there are no Tech Provider lifecycle APIs for WABA onboarding, phone registration, quality, limits, health, or credential rotation. Credentials are accepted/stored as `[]byte`; encryption helpers exist in `adapters/crypto/encrypt.go`, but the provider handler/store path does not apply encryption or decryption. |
| 2 | WhatsApp provider adapter | UNSAFE | `adapters/providers/whatsapp/stub.go`: `NewWhatsAppProvider`, `Send`, `SubmitTemplate`, `GetTemplateStatus`, `ParseWebhook`, `ValidateWebhookSignature`; `module.go`: registers `meta_cloud` with `whatsapp.NewWhatsAppProvider()` | Adapter is a stub. `Send` and `SubmitTemplate` return "not fully implemented"; `GetTemplateStatus` returns pending with no API call; `ParseWebhook` returns no events; `ValidateWebhookSignature` always returns `true`. |
| 3 | Meta Cloud API send support | MISSING | `adapters/providers/whatsapp/stub.go`: `Send`; `adapters/providers/whatsapp/types.go`: `SendMessageRequest`, `TemplateData`, `TextData` | Meta request structs exist, but there is no HTTP client, access token use, phone-number endpoint call, media/template formatting, response/error normalization, or retry classification. |
| 4 | Template CRUD | PRESENT | `adapters/http/template_handler.go`: `CreateTemplate`, `ListTemplates`, `GetTemplate`, `UpdateTemplate`, `DeleteTemplate`; `core/services/template_service.go`: `Create`, `List`, `GetByID`, `Update`, `Delete`; `core/domain/template.go`; `sql/queries/templates.sql` | Basic tenant-scoped template CRUD is present, including uniqueness by name/language and edit restrictions for non-draft/non-rejected templates. |
| 5 | Template submission to Meta | UNSAFE | `adapters/http/template_handler.go`: `SubmitForApproval`; `core/services/template_service.go`: `SubmitForApproval`; `adapters/providers/whatsapp/stub.go`: `SubmitTemplate` | Service flow exists, but the registered WhatsApp provider cannot submit templates to Meta. Frontend submit actions would fail for WhatsApp. |
| 6 | Template status sync | UNSAFE | `adapters/http/template_handler.go`: `SyncStatus`; `core/services/template_service.go`: `SyncStatus`; `adapters/providers/whatsapp/stub.go`: `GetTemplateStatus` | Sync endpoint exists, but WhatsApp provider returns `pending` without calling Meta, so status sync is misleading. |
| 7 | Webhook GET verification | PRESENT | `adapters/http/webhook_handler.go`: `Verify`; `spur.json`: `GET /messaging/webhook/whatsapp`; `openapi.json`: `/webhook/whatsapp` GET | Meta verification challenge is handled with platform verify token and challenge echo. |
| 8 | Webhook POST handling | PARTIAL | `adapters/http/webhook_handler.go`: `Handle`, `HandleWhatsApp`; `core/services/webhook_service.go`: `HandleWhatsAppWebhook` | POST endpoint reads the body and returns 200 to Meta. Service parses native WhatsApp payload directly for messages/statuses, but does not use the provider adapter and does not validate signatures. |
| 9 | Webhook signature validation | UNSAFE | `adapters/providers/whatsapp/stub.go`: `ValidateWebhookSignature`; `core/services/webhook_service.go`: `HandleWhatsAppWebhook` | WhatsApp signature validation is effectively absent. The registered provider returns `true`, and `HandleWhatsAppWebhook` does not call any validation path before processing the event. |
| 10 | Inbound message parsing | PARTIAL | `adapters/providers/whatsapp/types.go`: `WebhookPayload`, `WebhookMessage`; `core/services/webhook_service.go`: `processWhatsAppIncoming`, `processInboundConsentKeyword` | Text inbound messages are parsed and stored, conversations are updated, and consent keywords are processed. Non-text WhatsApp types are modeled only as placeholders and not fully parsed. |
| 11 | Status update parsing | PARTIAL | `adapters/providers/whatsapp/types.go`: `WebhookStatus`; `core/services/webhook_service.go`: `processWhatsAppStatus`, `mapWhatsAppStatus` | `sent`, `delivered`, `read`, and `failed` are mapped and out-of-order downgrades are avoided. Provider error details are not persisted, status parsing is limited, and no adapter-level normalization exists. |
| 12 | Tenant resolution by WABA ID and phone number ID | PARTIAL | `core/services/webhook_service.go`: `HandleWhatsAppWebhook`; `ports.ProviderConfigRepository.GetByWABAID`; `sql/queries/provider_configs.sql`: `GetProviderConfigByWABAID`; `adapters/postgres/store.go`: `GetProviderConfigByWABAID` | Webhook tenant resolution by WABA ID exists. There is no lookup or fallback by `phone_number_id`, which is required when multiple phone numbers or provider payload variants are involved. |
| 13 | Embedded Signup | MISSING | Search for embedded signup / signup callback found no endpoint or service | No Meta Embedded Signup session, code exchange, business mapping, or frontend callback API exists. |
| 14 | WABA onboarding callback | MISSING | No onboarding callback handler found; provider APIs only expose generic provider config CRUD | No callback flow to receive Meta onboarding results, store WABA/phone IDs, exchange tokens, or mark onboarding state. |
| 15 | Phone number registration/verification | MISSING | `domain.ProviderConfig` has `PhoneNumberID`, `WABAID`, `DisplayPhone`; `provider_handler.go` validates presence | Data fields exist, but no API or provider adapter code registers, verifies, subscribes, or health-checks phone numbers with Meta. |
| 16 | Contact opt-in | PRESENT | `adapters/http/contact_handler.go`: `OptIn`, `ConfirmOptIn`; `core/services/contact_service.go`: `OptIn`, `ConfirmOptIn`, `HandleInboundConsentKeyword`; `core/services/contact_service_test.go` | Channel-specific opt-in exists, consent evidence is recorded, and double opt-in is supported. |
| 17 | Contact opt-out | PRESENT | `adapters/http/contact_handler.go`: `OptOut`; `core/services/contact_service.go`: `OptOut`, `HandleInboundConsentKeyword`; `core/domain/consent_keywords_test.go` | Channel-specific opt-out exists, consent evidence is recorded, and inbound keywords can opt out a contact. |
| 18 | Suppression list | PRESENT | `adapters/http/suppression_handler.go`: `ListSuppressions`, `AddToSuppression`, `RemoveFromSuppression`, `BulkCheckSuppression`; `core/services/suppression_service.go`; `core/services/message_service.go`: suppression check before send; `worker/campaign_executor.go`: suppression check before campaign fan-out | Tenant/channel suppression exists and is enforced in message send and scheduled campaign execution. |
| 19 | Campaign execution | UNSAFE | `adapters/http/campaign_handler.go`: `ExecuteCampaign`; `core/services/campaign_service.go`: `Execute`; `worker/campaign_executor.go`: `ExecuteCampaign` | Basic campaign model and scheduled executor exist, but `CampaignService.Execute` builds message IDs and queues them without calling `messageRepo.Create`, so direct API execution can enqueue messages that workers cannot load. No wallet reservation, throttling, frequency caps, test-send requirement, or WhatsApp provider readiness. |
| 20 | Inbox/conversation updates | PARTIAL | `core/services/webhook_service.go`: `processWhatsAppIncoming`; `core/services/message_service.go`: WhatsApp outbound conversation handling; `core/services/conversation_service.go`; `adapters/http/conversation_handler.go` | Conversations are created/updated for inbound and outbound WhatsApp messages, and tenant APIs support list/update/notes. Full inbox features like transcript export, collision detection, teams, SLA automation, and rich media display are not present. |
| 21 | Billing/wallet usage | PARTIAL | `core/services/billing_service.go`: `GetWalletBalance`, `CreditWallet`, `EstimateMessageCost`, `RecordMessageCharge`; `worker/sender.go`: `RecordMessageCharge`; `adapters/http/routes.go`: wallet/billing routes | Wallet ledger, rate cards, and message debit recording exist after provider acceptance. There is no pre-send balance enforcement in `MessageService.Send`, no campaign reservation/estimate enforcement before launch, and WhatsApp provider cost is unavailable because the adapter is missing. |
| 22 | OpenAPI coverage | PARTIAL | `openapi.json`: providers, templates, contacts, campaigns, suppressions, wallet/billing, and `/webhook/whatsapp` paths | Existing routes are documented, but WhatsApp Tech Provider endpoints are absent: embedded signup, onboarding callback, phone registration/verification, phone/WABA health, template submit/status details, provider webhook signature requirements, and phone-number tenant resolution. |
| 23 | Tests | PARTIAL | Tests found in `core/services/contact_service_test.go`, `core/services/message_service_test.go`, `core/services/billing_service_test.go`, `core/services/conversation_service_test.go`, `core/services/tenant_webhook_service_test.go`, `adapters/providers/email/webhook_test.go`, `adapters/providers/sms/webhook_test.go` | There are useful tests for consent, message validation, billing, conversations, and tenant webhooks. There are no WhatsApp adapter tests, no Meta API send tests, no WhatsApp webhook signature tests, no WhatsApp webhook service tests, no template submit/sync tests against Meta behavior, and no campaign direct-execute regression test for message persistence. |

## Must Fix Before Frontend

1. Replace `adapters/providers/whatsapp/stub.go` with a real Meta Cloud API adapter:
   - implement `Send` for text and approved template messages first;
   - map Meta response IDs into `ProviderMessageID`;
   - normalize Meta errors into domain errors;
   - return accurate send status and cost metadata where available.

2. Implement WhatsApp webhook signature validation before processing POST events:
   - validate Meta `X-Hub-Signature-256` using the app secret;
   - reject or ignore unsigned/invalid payloads;
   - add tests for valid, invalid, and missing signatures.

3. Complete webhook routing and parsing:
   - resolve tenant by WABA ID and `phone_number_id`;
   - parse status errors and persist failure reason/provider code;
   - expand inbound parsing beyond text for media, location, interactive replies, and template button replies.

4. Make template submit/sync real:
   - call Meta template creation APIs in `SubmitTemplate`;
   - call Meta template status APIs in `GetTemplateStatus`;
   - persist rejection reason and provider template ID accurately;
   - add template submission/status tests.

5. Add onboarding APIs needed by the first frontend screen:
   - embedded signup start/config endpoint;
   - onboarding callback/code exchange endpoint;
   - WABA and phone-number persistence;
   - phone registration/verification/status endpoints.

6. Fix direct campaign execution before exposing launch controls:
   - ensure `CampaignService.Execute` stores each message before enqueueing;
   - add an automated regression test for direct API campaign launch;
   - add wallet estimate/balance checks before launch.

7. Secure provider credentials:
   - apply AES-256-GCM encryption on create/update;
   - decrypt only inside provider adapters;
   - ensure API responses never expose raw credentials.

8. Update `openapi.json` after the backend contract is defined:
   - document onboarding, WABA, phone-number, template submit/sync, and webhook security contracts;
   - include request/response examples the frontend can use directly.

9. Add WhatsApp-focused test coverage:
   - adapter send request formation;
   - Meta success/error normalization;
   - webhook GET verification;
   - webhook POST parsing and signature validation;
   - tenant resolution by WABA ID and phone number ID;
   - opt-out/suppression enforcement for WhatsApp sends.
