package consumer_service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryPGX struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *RepositoryPGX {
	return &RepositoryPGX{pool: pool}
}

func (r *RepositoryPGX) NextConsumerCode(ctx context.Context) (int64, error) {
	var code int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(code), 0) + 1 FROM consumer_service_consumers`).Scan(&code)
	return code, err
}

func (r *RepositoryPGX) NextCallNumber(ctx context.Context, enterpriseCode int64) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(call_number), 0) + 1 FROM consumer_service_calls WHERE enterprise_code=$1`, enterpriseCode).Scan(&n)
	return n, err
}

func (r *RepositoryPGX) CreateCallType(ctx context.Context, v *entity.CallType) (*entity.CallType, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO consumer_service_call_types (description, is_complaint, is_active, created_by)
		VALUES ($1,$2,$3,$4)
		RETURNING code, description, is_complaint, is_active, created_at, updated_at, created_by`,
		v.Description, v.IsComplaint, trueOrDefault(v.IsActive), pgutil.ToPgUUID(v.CreatedBy))
	return scanCallType(row)
}

func (r *RepositoryPGX) ListCallTypes(ctx context.Context, onlyActive bool) ([]*entity.CallType, error) {
	q := `SELECT code, description, is_complaint, is_active, created_at, updated_at, created_by FROM consumer_service_call_types`
	if onlyActive {
		q += ` WHERE is_active`
	}
	q += ` ORDER BY description`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.CallType
	for rows.Next() {
		row, err := scanCallType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) GetCallType(ctx context.Context, code int64) (*entity.CallType, error) {
	row := r.pool.QueryRow(ctx, `SELECT code, description, is_complaint, is_active, created_at, updated_at, created_by FROM consumer_service_call_types WHERE code=$1`, code)
	return scanCallType(row)
}

func (r *RepositoryPGX) CreateKnowledgeSource(ctx context.Context, v *entity.KnowledgeSource) (*entity.KnowledgeSource, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO consumer_service_knowledge_sources (description, is_active, created_by)
		VALUES ($1,$2,$3)
		RETURNING code, description, is_active, created_at, updated_at, created_by`,
		v.Description, trueOrDefault(v.IsActive), pgutil.ToPgUUID(v.CreatedBy))
	return scanKnowledgeSource(row)
}

func (r *RepositoryPGX) ListKnowledgeSources(ctx context.Context, onlyActive bool) ([]*entity.KnowledgeSource, error) {
	q := `SELECT code, description, is_active, created_at, updated_at, created_by FROM consumer_service_knowledge_sources`
	if onlyActive {
		q += ` WHERE is_active`
	}
	q += ` ORDER BY description`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.KnowledgeSource
	for rows.Next() {
		row, err := scanKnowledgeSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) CreateConsumer(ctx context.Context, v *entity.Consumer) (*entity.Consumer, error) {
	row := r.pool.QueryRow(ctx, consumerInsertSQL(), consumerArgs(v)...)
	return scanConsumer(row)
}

func (r *RepositoryPGX) UpdateConsumer(ctx context.Context, v *entity.Consumer) (*entity.Consumer, error) {
	row := r.pool.QueryRow(ctx, `UPDATE consumer_service_consumers SET
		name=$2, is_active=$3, person_type=$4, cpf=$5, rg=$6, cnpj=$7, state_registration=$8,
		zip_code=$9, city=$10, state=$11, address=$12, address_number=$13, complement=$14, district=$15,
		market_segment_code=$16, knowledge_code=$17, notes=$18, updated_at=NOW()
		WHERE code=$1
		RETURNING code, name, is_active, person_type, cpf, rg, cnpj, state_registration, zip_code, city, state,
		          address, address_number, complement, district, market_segment_code, knowledge_code, notes,
		          created_at, updated_at, created_by`,
		v.Code, v.Name, v.IsActive, v.PersonType, v.CPF, v.RG, v.CNPJ, v.StateRegistration, v.ZipCode, v.City,
		v.State, v.Address, v.AddressNumber, v.Complement, v.District, v.MarketSegmentCode, v.KnowledgeCode, v.Notes)
	consumer, err := scanConsumer(row)
	if err != nil {
		return nil, err
	}
	return r.hydrateConsumer(ctx, consumer)
}

func (r *RepositoryPGX) GetConsumer(ctx context.Context, code int64) (*entity.Consumer, error) {
	row := r.pool.QueryRow(ctx, consumerSelectSQL()+` WHERE code=$1`, code)
	consumer, err := scanConsumer(row)
	if err != nil {
		return nil, err
	}
	return r.hydrateConsumer(ctx, consumer)
}

func (r *RepositoryPGX) ListConsumers(ctx context.Context, filter repository.ConsumerFilter) ([]*entity.Consumer, error) {
	conds := []string{}
	args := []any{}
	if filter.Search != nil && strings.TrimSpace(*filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(*filter.Search))+"%")
		conds = append(conds, fmt.Sprintf("(LOWER(name) LIKE $%d OR CAST(code AS TEXT) LIKE $%d)", len(args), len(args)))
	}
	if filter.State != nil && strings.TrimSpace(*filter.State) != "" {
		args = append(args, strings.ToUpper(strings.TrimSpace(*filter.State)))
		conds = append(conds, fmt.Sprintf("UPPER(state)=$%d", len(args)))
	}
	if filter.City != nil && strings.TrimSpace(*filter.City) != "" {
		args = append(args, strings.ToLower(strings.TrimSpace(*filter.City)))
		conds = append(conds, fmt.Sprintf("LOWER(city)=$%d", len(args)))
	}
	if filter.OnlyActive {
		conds = append(conds, "is_active")
	}
	q := consumerSelectSQL()
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY name LIMIT 500"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.Consumer
	for rows.Next() {
		row, err := scanConsumer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) AddConsumerPhone(ctx context.Context, v *entity.ConsumerPhone) (*entity.ConsumerPhone, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO consumer_service_consumer_phones (consumer_code, contact_code, phone_type, number, is_primary)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING code, consumer_code, contact_code, phone_type, number, is_primary, created_at`,
		v.ConsumerCode, v.ContactCode, v.PhoneType, v.Number, v.IsPrimary)
	return scanConsumerPhone(row)
}

func (r *RepositoryPGX) AddConsumerEmail(ctx context.Context, v *entity.ConsumerEmail) (*entity.ConsumerEmail, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO consumer_service_consumer_emails (consumer_code, contact_code, email, is_primary)
		VALUES ($1,$2,$3,$4)
		RETURNING code, consumer_code, contact_code, email, is_primary, created_at`,
		v.ConsumerCode, v.ContactCode, v.Email, v.IsPrimary)
	return scanConsumerEmail(row)
}

func (r *RepositoryPGX) AddConsumerContact(ctx context.Context, v *entity.ConsumerContact) (*entity.ConsumerContact, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO consumer_service_consumer_contacts (consumer_code, name, role, contact_type, notes)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING code, consumer_code, name, role, contact_type, notes, created_at`,
		v.ConsumerCode, v.Name, v.Role, v.ContactType, v.Notes)
	return scanConsumerContact(row)
}

func (r *RepositoryPGX) CreateCustomerContact(ctx context.Context, v *entity.CustomerContactHistory) (*entity.CustomerContactHistory, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO consumer_service_customer_contacts
		(customer_code, opened_at, scheduled_at, user_code, contact_type, description, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING code, customer_code, opened_at, scheduled_at, user_code, contact_type, description, created_at, created_by`,
		v.CustomerCode, v.OpenedAt, v.ScheduledAt, v.UserCode, v.ContactType, v.Description, pgutil.ToPgUUID(v.CreatedBy))
	return scanCustomerContact(row)
}

func (r *RepositoryPGX) ListCustomerContacts(ctx context.Context, filter repository.CustomerContactFilter) ([]*entity.CustomerContactHistory, error) {
	conds := []string{}
	args := []any{}
	if filter.CustomerCode != nil {
		args = append(args, *filter.CustomerCode)
		conds = append(conds, fmt.Sprintf("customer_code=$%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		conds = append(conds, fmt.Sprintf("opened_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		conds = append(conds, fmt.Sprintf("opened_at < $%d", len(args)))
	}
	if filter.ContactType != nil && strings.TrimSpace(*filter.ContactType) != "" {
		args = append(args, strings.ToUpper(strings.TrimSpace(*filter.ContactType)))
		conds = append(conds, fmt.Sprintf("contact_type=$%d", len(args)))
	}
	q := `SELECT code, customer_code, opened_at, scheduled_at, user_code, contact_type, description, created_at, created_by FROM consumer_service_customer_contacts`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY opened_at DESC LIMIT 500"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.CustomerContactHistory
	for rows.Next() {
		row, err := scanCustomerContact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) CreateCall(ctx context.Context, v *entity.Call) (*entity.Call, error) {
	row := r.pool.QueryRow(ctx, callInsertSQL(), callArgs(v)...)
	return scanCall(row)
}

func (r *RepositoryPGX) UpdateCall(ctx context.Context, v *entity.Call) (*entity.Call, error) {
	row := r.pool.QueryRow(ctx, `UPDATE consumer_service_calls SET
		call_type_code=$2, direction=$3, in_warranty=$4, defect_group_code=$5, defect_reason_code=$6,
		responsible_user_code=$7, position=$8, situation=$9, return_date=$10, visit_requested_date=$11,
		visit_returned_date=$12, sale_store_code=$13, establishment_code=$14, technician_description=$15,
		symptoms=$16, forwarded_store_code=$17, subject=$18, description=$19, solution=$20, checklist_code=$21,
		updated_at=NOW()
		WHERE code=$1
		RETURNING `+callReturnColumns(),
		v.Code, v.CallTypeCode, string(v.Direction), v.InWarranty, v.DefectGroupCode, v.DefectReasonCode,
		v.ResponsibleUserCode, string(v.Position), string(v.Situation), datePtr(v.ReturnDate), datePtr(v.VisitRequestedDate),
		datePtr(v.VisitReturnedDate), v.SaleStoreCode, v.EstablishmentCode, v.TechnicianDescription, v.Symptoms,
		v.ForwardedStoreCode, v.Subject, v.Description, v.Solution, v.ChecklistCode)
	call, err := scanCall(row)
	if err != nil {
		return nil, err
	}
	return r.hydrateCall(ctx, call)
}

func (r *RepositoryPGX) GetCall(ctx context.Context, code int64) (*entity.Call, error) {
	row := r.pool.QueryRow(ctx, callSelectSQL()+` WHERE code=$1`, code)
	call, err := scanCall(row)
	if err != nil {
		return nil, err
	}
	return r.hydrateCall(ctx, call)
}

func (r *RepositoryPGX) ListCalls(ctx context.Context, filter repository.CallFilter) ([]*entity.Call, error) {
	q, args := buildCallFilterSQL(filter, false)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.Call
	for rows.Next() {
		row, err := scanCall(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) AddCallReturn(ctx context.Context, v *entity.CallReturn) (*entity.CallReturn, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO consumer_service_call_returns
		(call_code, contacted_at, contact_type, description, next_return_at, user_code, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING code, call_code, contacted_at, contact_type, description, next_return_at, user_code, created_at, created_by`,
		v.CallCode, v.ContactedAt, v.ContactType, v.Description, datePtr(v.NextReturnAt), v.UserCode, pgutil.ToPgUUID(v.CreatedBy))
	return scanCallReturn(row)
}

func (r *RepositoryPGX) AddCallAttachment(ctx context.Context, v *entity.CallAttachment) (*entity.CallAttachment, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO consumer_service_call_attachments
		(call_code, file_name, file_path, content_type, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING code, call_code, file_name, file_path, content_type, notes, created_at, created_by`,
		v.CallCode, v.FileName, v.FilePath, v.ContentType, v.Notes, pgutil.ToPgUUID(v.CreatedBy))
	return scanCallAttachment(row)
}

func (r *RepositoryPGX) AddChecklistItem(ctx context.Context, v *entity.CallChecklistItem) (*entity.CallChecklistItem, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO consumer_service_call_checklist_items (call_code, sequence, description, notes)
		VALUES ($1, COALESCE(NULLIF($2,0), (SELECT COALESCE(MAX(sequence),0)+1 FROM consumer_service_call_checklist_items WHERE call_code=$1)), $3, $4)
		RETURNING code, call_code, sequence, description, is_done, done_at, notes, created_at`,
		v.CallCode, v.Sequence, v.Description, v.Notes)
	return scanChecklistItem(row)
}

func (r *RepositoryPGX) SetChecklistItemDone(ctx context.Context, code int64, done bool, notes *string) (*entity.CallChecklistItem, error) {
	row := r.pool.QueryRow(ctx, `UPDATE consumer_service_call_checklist_items
		SET is_done=$2, done_at=CASE WHEN $2 THEN COALESCE(done_at, NOW()) ELSE NULL END, notes=COALESCE($3, notes)
		WHERE code=$1
		RETURNING code, call_code, sequence, description, is_done, done_at, notes, created_at`, code, done, notes)
	return scanChecklistItem(row)
}

func (r *RepositoryPGX) ReportCalls(ctx context.Context, filter repository.CallFilter) (*repository.CallReport, error) {
	q, args := buildCallFilterSQL(filter, true)
	row := r.pool.QueryRow(ctx, q, args...)
	var report repository.CallReport
	err := row.Scan(
		&report.TotalCalls, &report.PendingCalls, &report.ScheduledCalls, &report.ResolvedCalls,
		&report.TechnicalVisitCalls, &report.PendingVisitCalls, &report.ReturnedVisitCalls, &report.AverageResolutionHours,
	)
	return &report, err
}

func (r *RepositoryPGX) hydrateConsumer(ctx context.Context, c *entity.Consumer) (*entity.Consumer, error) {
	phones, err := r.listConsumerPhones(ctx, c.Code)
	if err != nil {
		return nil, err
	}
	emails, err := r.listConsumerEmails(ctx, c.Code)
	if err != nil {
		return nil, err
	}
	contacts, err := r.listConsumerContacts(ctx, c.Code)
	if err != nil {
		return nil, err
	}
	c.Phones, c.Emails, c.Contacts = phones, emails, contacts
	return c, nil
}

func (r *RepositoryPGX) hydrateCall(ctx context.Context, c *entity.Call) (*entity.Call, error) {
	returns, err := r.listCallReturns(ctx, c.Code)
	if err != nil {
		return nil, err
	}
	attachments, err := r.listCallAttachments(ctx, c.Code)
	if err != nil {
		return nil, err
	}
	items, err := r.listChecklistItems(ctx, c.Code)
	if err != nil {
		return nil, err
	}
	c.Returns, c.Attachments, c.ChecklistItems = returns, attachments, items
	return c, nil
}

func (r *RepositoryPGX) listConsumerPhones(ctx context.Context, consumerCode int64) ([]*entity.ConsumerPhone, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, consumer_code, contact_code, phone_type, number, is_primary, created_at FROM consumer_service_consumer_phones WHERE consumer_code=$1 ORDER BY is_primary DESC, code`, consumerCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.ConsumerPhone
	for rows.Next() {
		row, err := scanConsumerPhone(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) listConsumerEmails(ctx context.Context, consumerCode int64) ([]*entity.ConsumerEmail, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, consumer_code, contact_code, email, is_primary, created_at FROM consumer_service_consumer_emails WHERE consumer_code=$1 ORDER BY is_primary DESC, code`, consumerCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.ConsumerEmail
	for rows.Next() {
		row, err := scanConsumerEmail(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) listConsumerContacts(ctx context.Context, consumerCode int64) ([]*entity.ConsumerContact, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, consumer_code, name, role, contact_type, notes, created_at FROM consumer_service_consumer_contacts WHERE consumer_code=$1 ORDER BY name`, consumerCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.ConsumerContact
	for rows.Next() {
		row, err := scanConsumerContact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) listCallReturns(ctx context.Context, callCode int64) ([]*entity.CallReturn, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, call_code, contacted_at, contact_type, description, next_return_at, user_code, created_at, created_by FROM consumer_service_call_returns WHERE call_code=$1 ORDER BY contacted_at DESC`, callCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.CallReturn
	for rows.Next() {
		row, err := scanCallReturn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) listCallAttachments(ctx context.Context, callCode int64) ([]*entity.CallAttachment, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, call_code, file_name, file_path, content_type, notes, created_at, created_by FROM consumer_service_call_attachments WHERE call_code=$1 ORDER BY code`, callCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.CallAttachment
	for rows.Next() {
		row, err := scanCallAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) listChecklistItems(ctx context.Context, callCode int64) ([]*entity.CallChecklistItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, call_code, sequence, description, is_done, done_at, notes, created_at FROM consumer_service_call_checklist_items WHERE call_code=$1 ORDER BY sequence`, callCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.CallChecklistItem
	for rows.Next() {
		row, err := scanChecklistItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func buildCallFilterSQL(filter repository.CallFilter, aggregate bool) (string, []any) {
	conds := []string{}
	args := []any{}
	add := func(value any, condition string) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(condition, len(args)))
	}
	if filter.CallNumber != nil {
		add(*filter.CallNumber, "call_number=$%d")
	}
	if filter.CallTypeCode != nil {
		add(*filter.CallTypeCode, "call_type_code=$%d")
	}
	if filter.ConsumerCode != nil {
		add(*filter.ConsumerCode, "consumer_code=$%d")
	}
	if filter.ResponsibleUserCode != nil {
		add(*filter.ResponsibleUserCode, "responsible_user_code=$%d")
	}
	if filter.DefectGroupCode != nil {
		add(*filter.DefectGroupCode, "defect_group_code=$%d")
	}
	if filter.DefectReasonCode != nil {
		add(*filter.DefectReasonCode, "defect_reason_code=$%d")
	}
	if filter.Position != nil {
		add(string(*filter.Position), "position=$%d")
	}
	if filter.Situation != nil {
		add(string(*filter.Situation), "situation=$%d")
	}
	if filter.From != nil {
		add(*filter.From, "opened_at >= $%d")
	}
	if filter.To != nil {
		add(*filter.To, "opened_at < $%d")
	}
	if filter.ReturnFrom != nil {
		add(*filter.ReturnFrom, "return_date >= $%d")
	}
	if filter.ReturnTo != nil {
		add(*filter.ReturnTo, "return_date <= $%d")
	}
	if filter.VisitState != nil {
		switch strings.ToUpper(strings.TrimSpace(*filter.VisitState)) {
		case "PENDING", "PENDENTES":
			conds = append(conds, "situation='TECHNICAL_VISIT' AND visit_returned_date IS NULL")
		case "RETURNED", "REALIZADAS":
			conds = append(conds, "situation='TECHNICAL_VISIT' AND visit_returned_date IS NOT NULL")
		}
	}
	if filter.OnlyActive {
		conds = append(conds, "is_active")
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	if aggregate {
		return `SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE position='PENDING'),
			COUNT(*) FILTER (WHERE position='SCHEDULED'),
			COUNT(*) FILTER (WHERE position='RESOLVED'),
			COUNT(*) FILTER (WHERE situation='TECHNICAL_VISIT'),
			COUNT(*) FILTER (WHERE situation='TECHNICAL_VISIT' AND visit_returned_date IS NULL),
			COUNT(*) FILTER (WHERE situation='TECHNICAL_VISIT' AND visit_returned_date IS NOT NULL),
			COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(visit_returned_date::timestamp, updated_at) - opened_at))/3600) FILTER (WHERE position='RESOLVED' OR visit_returned_date IS NOT NULL), 0)
			FROM consumer_service_calls` + where, args
	}
	return callSelectSQL() + where + " ORDER BY opened_at DESC, code DESC LIMIT 500", args
}

func consumerInsertSQL() string {
	return `INSERT INTO consumer_service_consumers
		(code, name, is_active, person_type, cpf, rg, cnpj, state_registration, zip_code, city, state,
		 address, address_number, complement, district, market_segment_code, knowledge_code, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		RETURNING code, name, is_active, person_type, cpf, rg, cnpj, state_registration, zip_code, city, state,
		          address, address_number, complement, district, market_segment_code, knowledge_code, notes,
		          created_at, updated_at, created_by`
}

func consumerArgs(v *entity.Consumer) []any {
	return []any{v.Code, v.Name, trueOrDefault(v.IsActive), v.PersonType, v.CPF, v.RG, v.CNPJ, v.StateRegistration, v.ZipCode, v.City, v.State, v.Address, v.AddressNumber, v.Complement, v.District, v.MarketSegmentCode, v.KnowledgeCode, v.Notes, pgutil.ToPgUUID(v.CreatedBy)}
}

func consumerSelectSQL() string {
	return `SELECT code, name, is_active, person_type, cpf, rg, cnpj, state_registration, zip_code, city, state,
		address, address_number, complement, district, market_segment_code, knowledge_code, notes,
		created_at, updated_at, created_by FROM consumer_service_consumers`
}

func callInsertSQL() string {
	return `INSERT INTO consumer_service_calls
		(call_number, enterprise_code, consumer_code, customer_code, call_type_code, direction, in_warranty,
		 defect_group_code, defect_reason_code, responsible_user_code, position, situation, opened_at, return_date,
		 visit_requested_date, visit_returned_date, sale_store_code, establishment_code, technician_description,
		 symptoms, forwarded_store_code, subject, description, checklist_code, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
		RETURNING ` + callReturnColumns()
}

func callArgs(v *entity.Call) []any {
	return []any{v.CallNumber, v.EnterpriseCode, v.ConsumerCode, v.CustomerCode, v.CallTypeCode, string(v.Direction), v.InWarranty, v.DefectGroupCode, v.DefectReasonCode, v.ResponsibleUserCode, string(v.Position), string(v.Situation), v.OpenedAt, datePtr(v.ReturnDate), datePtr(v.VisitRequestedDate), datePtr(v.VisitReturnedDate), v.SaleStoreCode, v.EstablishmentCode, v.TechnicianDescription, v.Symptoms, v.ForwardedStoreCode, v.Subject, v.Description, v.ChecklistCode, pgutil.ToPgUUID(v.CreatedBy)}
}

func callSelectSQL() string {
	return `SELECT ` + callReturnColumns() + ` FROM consumer_service_calls`
}

func callReturnColumns() string {
	return `code, call_number, enterprise_code, consumer_code, customer_code, call_type_code, direction, in_warranty,
		defect_group_code, defect_reason_code, responsible_user_code, position, situation, opened_at, return_date,
		visit_requested_date, visit_returned_date, sale_store_code, establishment_code, technician_description,
		symptoms, forwarded_store_code, subject, description, solution, checklist_code, is_active, created_at, updated_at, created_by`
}

func scanCallType(row pgx.Row) (*entity.CallType, error) {
	var v entity.CallType
	err := row.Scan(&v.Code, &v.Description, &v.IsComplaint, &v.IsActive, &v.CreatedAt, &v.UpdatedAt, &v.CreatedBy)
	return &v, err
}

func scanKnowledgeSource(row pgx.Row) (*entity.KnowledgeSource, error) {
	var v entity.KnowledgeSource
	err := row.Scan(&v.Code, &v.Description, &v.IsActive, &v.CreatedAt, &v.UpdatedAt, &v.CreatedBy)
	return &v, err
}

func scanConsumer(row pgx.Row) (*entity.Consumer, error) {
	var v entity.Consumer
	err := row.Scan(&v.Code, &v.Name, &v.IsActive, &v.PersonType, &v.CPF, &v.RG, &v.CNPJ, &v.StateRegistration, &v.ZipCode, &v.City, &v.State, &v.Address, &v.AddressNumber, &v.Complement, &v.District, &v.MarketSegmentCode, &v.KnowledgeCode, &v.Notes, &v.CreatedAt, &v.UpdatedAt, &v.CreatedBy)
	return &v, err
}

func scanConsumerPhone(row pgx.Row) (*entity.ConsumerPhone, error) {
	var v entity.ConsumerPhone
	err := row.Scan(&v.Code, &v.ConsumerCode, &v.ContactCode, &v.PhoneType, &v.Number, &v.IsPrimary, &v.CreatedAt)
	return &v, err
}

func scanConsumerEmail(row pgx.Row) (*entity.ConsumerEmail, error) {
	var v entity.ConsumerEmail
	err := row.Scan(&v.Code, &v.ConsumerCode, &v.ContactCode, &v.Email, &v.IsPrimary, &v.CreatedAt)
	return &v, err
}

func scanConsumerContact(row pgx.Row) (*entity.ConsumerContact, error) {
	var v entity.ConsumerContact
	err := row.Scan(&v.Code, &v.ConsumerCode, &v.Name, &v.Role, &v.ContactType, &v.Notes, &v.CreatedAt)
	return &v, err
}

func scanCustomerContact(row pgx.Row) (*entity.CustomerContactHistory, error) {
	var v entity.CustomerContactHistory
	err := row.Scan(&v.Code, &v.CustomerCode, &v.OpenedAt, &v.ScheduledAt, &v.UserCode, &v.ContactType, &v.Description, &v.CreatedAt, &v.CreatedBy)
	return &v, err
}

func scanCall(row pgx.Row) (*entity.Call, error) {
	var v entity.Call
	var direction, position, situation string
	err := row.Scan(&v.Code, &v.CallNumber, &v.EnterpriseCode, &v.ConsumerCode, &v.CustomerCode, &v.CallTypeCode, &direction, &v.InWarranty, &v.DefectGroupCode, &v.DefectReasonCode, &v.ResponsibleUserCode, &position, &situation, &v.OpenedAt, &v.ReturnDate, &v.VisitRequestedDate, &v.VisitReturnedDate, &v.SaleStoreCode, &v.EstablishmentCode, &v.TechnicianDescription, &v.Symptoms, &v.ForwardedStoreCode, &v.Subject, &v.Description, &v.Solution, &v.ChecklistCode, &v.IsActive, &v.CreatedAt, &v.UpdatedAt, &v.CreatedBy)
	v.Direction = entity.CallDirection(direction)
	v.Position = entity.CallPosition(position)
	v.Situation = entity.CallSituation(situation)
	return &v, err
}

func scanCallReturn(row pgx.Row) (*entity.CallReturn, error) {
	var v entity.CallReturn
	err := row.Scan(&v.Code, &v.CallCode, &v.ContactedAt, &v.ContactType, &v.Description, &v.NextReturnAt, &v.UserCode, &v.CreatedAt, &v.CreatedBy)
	return &v, err
}

func scanCallAttachment(row pgx.Row) (*entity.CallAttachment, error) {
	var v entity.CallAttachment
	err := row.Scan(&v.Code, &v.CallCode, &v.FileName, &v.FilePath, &v.ContentType, &v.Notes, &v.CreatedAt, &v.CreatedBy)
	return &v, err
}

func scanChecklistItem(row pgx.Row) (*entity.CallChecklistItem, error) {
	var v entity.CallChecklistItem
	err := row.Scan(&v.Code, &v.CallCode, &v.Sequence, &v.Description, &v.IsDone, &v.DoneAt, &v.Notes, &v.CreatedAt)
	return &v, err
}

func trueOrDefault(v bool) bool {
	return v || !v
}

func datePtr(v *time.Time) any {
	if v == nil {
		return nil
	}
	return *v
}
