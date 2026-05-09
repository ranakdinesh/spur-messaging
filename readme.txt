# Spur Messaging Module

## Overview
Spur Messaging is a robust, multi-channel communication backend for the Spur platform. It provides a unified API and infrastructure for sending messages via WhatsApp, SMS, and Email. Built with Go 1.25+, it follows a strict hexagonal architecture and integrates seamlessly with the Spur Identity module for authentication and tenant isolation.

## Key Features

### Multi-Channel Messaging
- **WhatsApp**: Integration with Meta Cloud API for template-based and session-based messaging.
- **SMS**: Support for MSG91 (India) and Twilio (International) providers.
- **Email**: Integrated transactional and marketing email system with support for SendGrid, Mailgun, and Postmark.

### Campaign Management
- Create and execute bulk messaging campaigns across all channels.
- Targeted messaging using static contact lists or dynamic segments.
- Real-time campaign analytics (sent, delivered, read, failed).
- Crash recovery: Automatically resumes interrupted campaigns.

### Contact & Segment Management
- Centralized contact repository with opt-in/opt-out management.
- Dynamic segmentation based on contact attributes and tags.
- Bulk import capabilities with detailed error reporting.

### Reliability & Compliance
- **Queue-based Processing**: Uses Redis Streams for reliable message delivery and survivor worker restarts.
- **Auto-Suppression**: Automatic management of email suppression lists (hard bounces, complaints).
- **Compliance**: Built-in support for unsubscribe links, DLT registration (SMS India), and WhatsApp 24-hour session rules.
- **Security**: AES-256-GCM encryption for all provider credentials.

### Analytics
- Comprehensive delivery and engagement tracking for all channels.
- Email-specific metrics: Open rates, click rates, and domain reputation monitoring.
- Per-message event timelines.

## How to Use

### Prerequisites
- Go 1.25+
- PostgreSQL 16+
- Redis 7+
- SQLC (for database code generation)

### Installation
1. Clone the repository:
   ```bash
   git clone github.com/ranakdinesh/spur-messaging
   cd spur-messaging
   ```
2. Install dependencies:
   ```bash
   go mod download
   ```

### Configuration
Configuration is handled via environment variables and per-tenant settings in the database.
- `MESSAGING_EMAIL_PROVIDER`: "sendgrid" | "mailgun" | "postmark"
- `MESSAGING_SMS_PROVIDER`: "msg91" | "twilio"
- `REDIS_URL`: Connection string for Redis.
- Database connection details for PostgreSQL.

### Database Setup
1. Apply migrations located in `sql/migrations/`.
2. Generate SQLC code (if modifying queries):
   ```bash
   sqlc generate
   ```

### Running the Module
The messaging module is designed to be composed into the `spur-template` binary. It exposes a `New()` function and `RegisterRoutes()` method for integration.

To build the package:
```bash
go build ./...
```

### API Documentation
The module exposes an OpenAPI 3.1 specification at the project root (`openapi.json`). This file contains detailed information about all available endpoints, request structures, and response envelopes.

## Architecture
- **Hexagonal Architecture**: Clear separation between domain logic (`core/`) and infrastructure/interfaces (`adapters/`).
- **Stateless Services**: All business logic is contained within services that depend on port interfaces.
- **Background Workers**: Dedicated workers handle message sending (`worker/sender.go`) and campaign execution (`worker/campaign_executor.go`).
