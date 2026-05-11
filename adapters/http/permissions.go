package http

import msgpermissions "github.com/ranakdinesh/spur-messaging/pkg/permissions"

const (
	permAnalyticsRead = msgpermissions.AnalyticsRead

	permProvidersRead  = msgpermissions.ProvidersRead
	permProvidersWrite = msgpermissions.ProvidersWrite
	permProvidersTest  = msgpermissions.ProvidersTest

	permTemplatesRead   = msgpermissions.TemplatesRead
	permTemplatesWrite  = msgpermissions.TemplatesWrite
	permTemplatesSubmit = msgpermissions.TemplatesSubmit

	permContactsRead          = msgpermissions.ContactsRead
	permContactsWrite         = msgpermissions.ContactsWrite
	permContactsImport        = msgpermissions.ContactsImport
	permContactsManageConsent = msgpermissions.ContactsManageConsent

	permSegmentsRead  = msgpermissions.SegmentsRead
	permSegmentsWrite = msgpermissions.SegmentsWrite

	permCampaignsRead    = msgpermissions.CampaignsRead
	permCampaignsWrite   = msgpermissions.CampaignsWrite
	permCampaignsExecute = msgpermissions.CampaignsExecute

	permMessagesRead     = msgpermissions.MessagesRead
	permMessagesSend     = msgpermissions.MessagesSend
	permMessagesSendBulk = msgpermissions.MessagesSendBulk
)
