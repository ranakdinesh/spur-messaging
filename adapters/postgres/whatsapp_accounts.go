package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/ranakdinesh/spur-messaging/adapters/postgres/gen"
	"github.com/ranakdinesh/spur-messaging/core/domain"
)

func (s *Store) CreateWhatsAppBusinessAccount(ctx context.Context, account *domain.WhatsAppBusinessAccount) error {
	row, err := s.q.CreateWhatsAppBusinessAccount(ctx, gen.CreateWhatsAppBusinessAccountParams{
		TenantID:                   account.TenantID,
		MetaBusinessID:             account.MetaBusinessID,
		WabaID:                     account.WABAID,
		Name:                       account.Name,
		Currency:                   account.Currency,
		TimezoneID:                 account.TimezoneID,
		BusinessVerificationStatus: string(account.BusinessVerificationStatus),
		OnboardingStatus:           string(account.OnboardingStatus),
		ProviderConfigID:           fromUUIDPtr(account.ProviderConfigID),
		LastSyncedAt:               fromTimePtr(account.LastSyncedAt),
	})
	if err != nil {
		return err
	}
	*account = toWhatsAppBusinessAccountDomain(row)
	return nil
}

func (s *Store) GetWhatsAppBusinessAccount(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppBusinessAccount, error) {
	row, err := s.q.GetWhatsAppBusinessAccount(ctx, gen.GetWhatsAppBusinessAccountParams{TenantID: tenantID, ID: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	account := toWhatsAppBusinessAccountDomain(row)
	return &account, nil
}

func (s *Store) GetWhatsAppBusinessAccountByWABAID(ctx context.Context, tenantID uuid.UUID, wabaID string) (*domain.WhatsAppBusinessAccount, error) {
	row, err := s.q.GetWhatsAppBusinessAccountByWABAID(ctx, gen.GetWhatsAppBusinessAccountByWABAIDParams{TenantID: tenantID, WabaID: wabaID})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	account := toWhatsAppBusinessAccountDomain(row)
	return &account, nil
}

func (s *Store) ListWhatsAppBusinessAccounts(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]domain.WhatsAppBusinessAccount, int, error) {
	rows, err := s.q.ListWhatsAppBusinessAccounts(ctx, gen.ListWhatsAppBusinessAccountsParams{
		TenantID: tenantID,
		Limit:    int32(perPage),
		Offset:   int32((page - 1) * perPage),
	})
	if err != nil {
		return nil, 0, err
	}
	accounts := make([]domain.WhatsAppBusinessAccount, 0, len(rows))
	total := 0
	for _, row := range rows {
		accounts = append(accounts, toWhatsAppBusinessAccountDomainFromList(row))
		total = int(row.TotalCount)
	}
	return accounts, total, nil
}

func (s *Store) UpdateWhatsAppBusinessAccountStatus(ctx context.Context, tenantID, id uuid.UUID, businessStatus domain.WhatsAppBusinessVerificationStatus, onboardingStatus domain.WhatsAppOnboardingStatus) (*domain.WhatsAppBusinessAccount, error) {
	row, err := s.q.UpdateWhatsAppBusinessAccountStatus(ctx, gen.UpdateWhatsAppBusinessAccountStatusParams{
		TenantID:                   tenantID,
		ID:                         id,
		BusinessVerificationStatus: string(businessStatus),
		OnboardingStatus:           string(onboardingStatus),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	account := toWhatsAppBusinessAccountDomain(row)
	return &account, nil
}

func (s *Store) UpdateWhatsAppBusinessAccountSync(ctx context.Context, account *domain.WhatsAppBusinessAccount) (*domain.WhatsAppBusinessAccount, error) {
	row, err := s.q.UpdateWhatsAppBusinessAccountSync(ctx, gen.UpdateWhatsAppBusinessAccountSyncParams{
		TenantID:                   account.TenantID,
		ID:                         account.ID,
		MetaBusinessID:             account.MetaBusinessID,
		Name:                       account.Name,
		Currency:                   account.Currency,
		TimezoneID:                 account.TimezoneID,
		BusinessVerificationStatus: string(account.BusinessVerificationStatus),
		OnboardingStatus:           string(account.OnboardingStatus),
		ProviderConfigID:           fromUUIDPtr(account.ProviderConfigID),
		LastSyncedAt:               fromTimePtr(account.LastSyncedAt),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	updated := toWhatsAppBusinessAccountDomain(row)
	return &updated, nil
}

func (s *Store) DeleteWhatsAppBusinessAccount(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.q.DeleteWhatsAppBusinessAccount(ctx, gen.DeleteWhatsAppBusinessAccountParams{TenantID: tenantID, ID: id})
}

func (s *Store) CreateWhatsAppPhoneNumber(ctx context.Context, phone *domain.WhatsAppPhoneNumber) error {
	row, err := s.q.CreateWhatsAppPhoneNumber(ctx, gen.CreateWhatsAppPhoneNumberParams{
		TenantID:               phone.TenantID,
		WabaID:                 phone.WABAID,
		PhoneNumberID:          phone.PhoneNumberID,
		DisplayPhoneNumber:     phone.DisplayPhoneNumber,
		VerifiedName:           phone.VerifiedName,
		QualityRating:          string(phone.QualityRating),
		MessagingLimitTier:     phone.MessagingLimitTier,
		Status:                 string(phone.Status),
		CodeVerificationStatus: string(phone.CodeVerificationStatus),
		LastSyncedAt:           fromTimePtr(phone.LastSyncedAt),
	})
	if err != nil {
		return err
	}
	*phone = toWhatsAppPhoneNumberDomain(row)
	return nil
}

func (s *Store) GetWhatsAppPhoneNumber(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppPhoneNumber, error) {
	row, err := s.q.GetWhatsAppPhoneNumber(ctx, gen.GetWhatsAppPhoneNumberParams{TenantID: tenantID, ID: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	phone := toWhatsAppPhoneNumberDomain(row)
	return &phone, nil
}

func (s *Store) GetWhatsAppPhoneNumberByPhoneNumberID(ctx context.Context, tenantID uuid.UUID, phoneNumberID string) (*domain.WhatsAppPhoneNumber, error) {
	row, err := s.q.GetWhatsAppPhoneNumberByPhoneNumberID(ctx, gen.GetWhatsAppPhoneNumberByPhoneNumberIDParams{TenantID: tenantID, PhoneNumberID: phoneNumberID})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	phone := toWhatsAppPhoneNumberDomain(row)
	return &phone, nil
}

func (s *Store) ListWhatsAppPhoneNumbersByWABA(ctx context.Context, tenantID uuid.UUID, wabaID string, page, perPage int) ([]domain.WhatsAppPhoneNumber, int, error) {
	rows, err := s.q.ListWhatsAppPhoneNumbersByWABA(ctx, gen.ListWhatsAppPhoneNumbersByWABAParams{
		TenantID: tenantID,
		WabaID:   wabaID,
		Limit:    int32(perPage),
		Offset:   int32((page - 1) * perPage),
	})
	if err != nil {
		return nil, 0, err
	}
	phones := make([]domain.WhatsAppPhoneNumber, 0, len(rows))
	total := 0
	for _, row := range rows {
		phones = append(phones, toWhatsAppPhoneNumberDomainFromWABAList(row))
		total = int(row.TotalCount)
	}
	return phones, total, nil
}

func (s *Store) ListWhatsAppPhoneNumbers(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]domain.WhatsAppPhoneNumber, int, error) {
	rows, err := s.q.ListWhatsAppPhoneNumbers(ctx, gen.ListWhatsAppPhoneNumbersParams{
		TenantID: tenantID,
		Limit:    int32(perPage),
		Offset:   int32((page - 1) * perPage),
	})
	if err != nil {
		return nil, 0, err
	}
	phones := make([]domain.WhatsAppPhoneNumber, 0, len(rows))
	total := 0
	for _, row := range rows {
		phones = append(phones, toWhatsAppPhoneNumberDomainFromList(row))
		total = int(row.TotalCount)
	}
	return phones, total, nil
}

func (s *Store) UpdateWhatsAppPhoneNumberStatus(ctx context.Context, tenantID, id uuid.UUID, quality domain.WhatsAppQualityRating, limitTier string, status domain.WhatsAppPhoneNumberStatus, codeStatus domain.WhatsAppCodeVerificationStatus) (*domain.WhatsAppPhoneNumber, error) {
	row, err := s.q.UpdateWhatsAppPhoneNumberStatus(ctx, gen.UpdateWhatsAppPhoneNumberStatusParams{
		TenantID:               tenantID,
		ID:                     id,
		QualityRating:          string(quality),
		MessagingLimitTier:     limitTier,
		Status:                 string(status),
		CodeVerificationStatus: string(codeStatus),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	phone := toWhatsAppPhoneNumberDomain(row)
	return &phone, nil
}

func (s *Store) UpdateWhatsAppPhoneNumberSync(ctx context.Context, phone *domain.WhatsAppPhoneNumber) (*domain.WhatsAppPhoneNumber, error) {
	row, err := s.q.UpdateWhatsAppPhoneNumberSync(ctx, gen.UpdateWhatsAppPhoneNumberSyncParams{
		TenantID:               phone.TenantID,
		ID:                     phone.ID,
		DisplayPhoneNumber:     phone.DisplayPhoneNumber,
		VerifiedName:           phone.VerifiedName,
		QualityRating:          string(phone.QualityRating),
		MessagingLimitTier:     phone.MessagingLimitTier,
		Status:                 string(phone.Status),
		CodeVerificationStatus: string(phone.CodeVerificationStatus),
		LastSyncedAt:           fromTimePtr(phone.LastSyncedAt),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	updated := toWhatsAppPhoneNumberDomain(row)
	return &updated, nil
}

func (s *Store) DeleteWhatsAppPhoneNumber(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.q.DeleteWhatsAppPhoneNumber(ctx, gen.DeleteWhatsAppPhoneNumberParams{TenantID: tenantID, ID: id})
}

func (s *Store) CreateWhatsAppOnboardingSession(ctx context.Context, session *domain.WhatsAppOnboardingSession) error {
	row, err := s.q.CreateWhatsAppOnboardingSession(ctx, gen.CreateWhatsAppOnboardingSessionParams{
		TenantID: session.TenantID,
		State:    session.State,
		Status:   string(session.Status),
	})
	if err != nil {
		return err
	}
	*session = toWhatsAppOnboardingSessionDomain(row)
	return nil
}

func (s *Store) GetWhatsAppOnboardingSession(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppOnboardingSession, error) {
	row, err := s.q.GetWhatsAppOnboardingSession(ctx, gen.GetWhatsAppOnboardingSessionParams{TenantID: tenantID, ID: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	session := toWhatsAppOnboardingSessionDomain(row)
	return &session, nil
}

func (s *Store) GetWhatsAppOnboardingSessionByState(ctx context.Context, tenantID uuid.UUID, state string) (*domain.WhatsAppOnboardingSession, error) {
	row, err := s.q.GetWhatsAppOnboardingSessionByState(ctx, gen.GetWhatsAppOnboardingSessionByStateParams{TenantID: tenantID, State: state})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	session := toWhatsAppOnboardingSessionDomain(row)
	return &session, nil
}

func (s *Store) UpdateWhatsAppOnboardingSessionStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.WhatsAppOnboardingStatus) (*domain.WhatsAppOnboardingSession, error) {
	row, err := s.q.UpdateWhatsAppOnboardingSessionStatus(ctx, gen.UpdateWhatsAppOnboardingSessionStatusParams{TenantID: tenantID, ID: id, Status: string(status)})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	session := toWhatsAppOnboardingSessionDomain(row)
	return &session, nil
}

func (s *Store) CompleteWhatsAppOnboardingSession(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppOnboardingSession, error) {
	row, err := s.q.CompleteWhatsAppOnboardingSession(ctx, gen.CompleteWhatsAppOnboardingSessionParams{TenantID: tenantID, ID: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	session := toWhatsAppOnboardingSessionDomain(row)
	return &session, nil
}

func (s *Store) FailWhatsAppOnboardingSession(ctx context.Context, tenantID, id uuid.UUID, message string) (*domain.WhatsAppOnboardingSession, error) {
	row, err := s.q.FailWhatsAppOnboardingSession(ctx, gen.FailWhatsAppOnboardingSessionParams{
		TenantID:     tenantID,
		ID:           id,
		ErrorMessage: fromString(message),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	session := toWhatsAppOnboardingSessionDomain(row)
	return &session, nil
}

func toWhatsAppBusinessAccountDomain(row gen.MessagingWhatsappBusinessAccount) domain.WhatsAppBusinessAccount {
	return domain.WhatsAppBusinessAccount{
		ID:                         row.ID,
		TenantID:                   row.TenantID,
		MetaBusinessID:             row.MetaBusinessID,
		WABAID:                     row.WabaID,
		Name:                       row.Name,
		Currency:                   row.Currency,
		TimezoneID:                 row.TimezoneID,
		BusinessVerificationStatus: domain.WhatsAppBusinessVerificationStatus(row.BusinessVerificationStatus),
		OnboardingStatus:           domain.WhatsAppOnboardingStatus(row.OnboardingStatus),
		ProviderConfigID:           pgUUIDToPtr(row.ProviderConfigID),
		LastSyncedAt:               pgTimestamptzToPtr(row.LastSyncedAt),
		CreatedAt:                  row.CreatedAt,
		UpdatedAt:                  row.UpdatedAt,
	}
}

func toWhatsAppBusinessAccountDomainFromList(row gen.ListWhatsAppBusinessAccountsRow) domain.WhatsAppBusinessAccount {
	return domain.WhatsAppBusinessAccount{
		ID:                         row.ID,
		TenantID:                   row.TenantID,
		MetaBusinessID:             row.MetaBusinessID,
		WABAID:                     row.WabaID,
		Name:                       row.Name,
		Currency:                   row.Currency,
		TimezoneID:                 row.TimezoneID,
		BusinessVerificationStatus: domain.WhatsAppBusinessVerificationStatus(row.BusinessVerificationStatus),
		OnboardingStatus:           domain.WhatsAppOnboardingStatus(row.OnboardingStatus),
		ProviderConfigID:           pgUUIDToPtr(row.ProviderConfigID),
		LastSyncedAt:               pgTimestamptzToPtr(row.LastSyncedAt),
		CreatedAt:                  row.CreatedAt,
		UpdatedAt:                  row.UpdatedAt,
	}
}

func toWhatsAppPhoneNumberDomain(row gen.MessagingWhatsappPhoneNumber) domain.WhatsAppPhoneNumber {
	return domain.WhatsAppPhoneNumber{
		ID:                     row.ID,
		TenantID:               row.TenantID,
		WABAID:                 row.WabaID,
		PhoneNumberID:          row.PhoneNumberID,
		DisplayPhoneNumber:     row.DisplayPhoneNumber,
		VerifiedName:           row.VerifiedName,
		QualityRating:          domain.WhatsAppQualityRating(row.QualityRating),
		MessagingLimitTier:     row.MessagingLimitTier,
		Status:                 domain.WhatsAppPhoneNumberStatus(row.Status),
		CodeVerificationStatus: domain.WhatsAppCodeVerificationStatus(row.CodeVerificationStatus),
		LastSyncedAt:           pgTimestamptzToPtr(row.LastSyncedAt),
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func toWhatsAppPhoneNumberDomainFromList(row gen.ListWhatsAppPhoneNumbersRow) domain.WhatsAppPhoneNumber {
	return domain.WhatsAppPhoneNumber{
		ID:                     row.ID,
		TenantID:               row.TenantID,
		WABAID:                 row.WabaID,
		PhoneNumberID:          row.PhoneNumberID,
		DisplayPhoneNumber:     row.DisplayPhoneNumber,
		VerifiedName:           row.VerifiedName,
		QualityRating:          domain.WhatsAppQualityRating(row.QualityRating),
		MessagingLimitTier:     row.MessagingLimitTier,
		Status:                 domain.WhatsAppPhoneNumberStatus(row.Status),
		CodeVerificationStatus: domain.WhatsAppCodeVerificationStatus(row.CodeVerificationStatus),
		LastSyncedAt:           pgTimestamptzToPtr(row.LastSyncedAt),
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func toWhatsAppPhoneNumberDomainFromWABAList(row gen.ListWhatsAppPhoneNumbersByWABARow) domain.WhatsAppPhoneNumber {
	return domain.WhatsAppPhoneNumber{
		ID:                     row.ID,
		TenantID:               row.TenantID,
		WABAID:                 row.WabaID,
		PhoneNumberID:          row.PhoneNumberID,
		DisplayPhoneNumber:     row.DisplayPhoneNumber,
		VerifiedName:           row.VerifiedName,
		QualityRating:          domain.WhatsAppQualityRating(row.QualityRating),
		MessagingLimitTier:     row.MessagingLimitTier,
		Status:                 domain.WhatsAppPhoneNumberStatus(row.Status),
		CodeVerificationStatus: domain.WhatsAppCodeVerificationStatus(row.CodeVerificationStatus),
		LastSyncedAt:           pgTimestamptzToPtr(row.LastSyncedAt),
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func toWhatsAppOnboardingSessionDomain(row gen.MessagingWhatsappOnboardingSession) domain.WhatsAppOnboardingSession {
	return domain.WhatsAppOnboardingSession{
		ID:           row.ID,
		TenantID:     row.TenantID,
		State:        row.State,
		Status:       domain.WhatsAppOnboardingStatus(row.Status),
		ErrorMessage: pgTextToStringPtr(row.ErrorMessage),
		CompletedAt:  pgTimestamptzToPtr(row.CompletedAt),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
