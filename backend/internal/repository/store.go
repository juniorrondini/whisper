package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

type Company struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	CNPJ   string         `json:"cnpj,omitempty"`
	Plan   string         `json:"plan"`
	Status string         `json:"status"`
	Config map[string]any `json:"config,omitempty"`
}

type User struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type UserWithPassword struct {
	User
	PasswordHash string
}

type Customer struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Department struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type Conversation struct {
	ID                  string     `json:"id"`
	CompanyID           string     `json:"company_id"`
	CustomerID          string     `json:"customer_id"`
	CustomerName        string     `json:"customer_name"`
	AssignedUserID      string     `json:"assigned_user_id,omitempty"`
	AssignedUserName    string     `json:"assigned_user_name,omitempty"`
	DepartmentID        string     `json:"department_id,omitempty"`
	DepartmentName      string     `json:"department_name,omitempty"`
	Status              string     `json:"status"`
	Priority            string     `json:"priority"`
	Origin              string     `json:"origin"`
	Subject             string     `json:"subject"`
	FirstResponseAt     *time.Time `json:"first_response_at,omitempty"`
	ClosedAt            *time.Time `json:"closed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	LastMessagePreview   string     `json:"last_message_preview,omitempty"`
	LastMessageCreatedAt *time.Time `json:"last_message_created_at,omitempty"`
}

type Message struct {
	ID             string         `json:"id"`
	CompanyID      string         `json:"company_id"`
	ConversationID string         `json:"conversation_id"`
	CustomerID     string         `json:"customer_id"`
	SenderType     string         `json:"sender_type"`
	SenderID       string         `json:"sender_id,omitempty"`
	Type           string         `json:"type"`
	Content        string         `json:"content"`
	Status         string         `json:"status"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

type QuickReply struct {
	ID           string    `json:"id"`
	CompanyID    string    `json:"company_id"`
	DepartmentID string    `json:"department_id,omitempty"`
	Title        string    `json:"title"`
	Shortcut     string    `json:"shortcut"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

type Dashboard struct {
	OpenConversations     int64 `json:"open_conversations"`
	ClosedToday           int64 `json:"closed_today"`
	WaitingCustomers      int64 `json:"waiting_customers"`
	OnlineAgents          int64 `json:"online_agents"`
	MessagesToday         int64 `json:"messages_today"`
	AvgFirstResponseSecs  int64 `json:"avg_first_response_seconds"`
	AvgConversationSecs   int64 `json:"avg_conversation_seconds"`
	CustomersTotal        int64 `json:"customers_total"`
}

func (s *Store) CreateCompanyWithAdmin(ctx context.Context, companyName, cnpj, adminName, email, passwordHash string) (Company, User, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Company{}, User{}, err
	}
	defer tx.Rollback(ctx)

	var company Company
	if err := tx.QueryRow(ctx, `
		insert into companies(name, cnpj, plan, status, config)
		values($1, nullif($2, ''), 'starter', 'active', '{}'::jsonb)
		returning id::text, name, coalesce(cnpj, ''), plan, status
	`, companyName, cnpj).Scan(&company.ID, &company.Name, &company.CNPJ, &company.Plan, &company.Status); err != nil {
		return Company{}, User{}, err
	}

	var user User
	if err := tx.QueryRow(ctx, `
		insert into users(company_id, name, email, password_hash, role, status)
		values($1, $2, lower($3), $4, 'ADMIN', 'active')
		returning id::text, company_id::text, name, email, role, status, created_at
	`, company.ID, adminName, email, passwordHash).Scan(&user.ID, &user.CompanyID, &user.Name, &user.Email, &user.Role, &user.Status, &user.CreatedAt); err != nil {
		return Company{}, User{}, err
	}

	for _, name := range []string{"Financeiro", "Suporte tecnico", "Comercial", "Pos-venda"} {
		if _, err := tx.Exec(ctx, `insert into departments(company_id, name) values($1, $2)`, company.ID, name); err != nil {
			return Company{}, User{}, err
		}
	}

	return company, user, tx.Commit(ctx)
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (UserWithPassword, error) {
	var user UserWithPassword
	err := s.db.QueryRow(ctx, `
		select id::text, company_id::text, name, email, role, status, created_at, password_hash
		from users
		where email = lower($1)
	`, email).Scan(&user.ID, &user.CompanyID, &user.Name, &user.Email, &user.Role, &user.Status, &user.CreatedAt, &user.PasswordHash)
	return user, err
}

func (s *Store) GetUser(ctx context.Context, companyID, userID uuid.UUID) (User, error) {
	var user User
	err := s.db.QueryRow(ctx, `
		select id::text, company_id::text, name, email, role, status, created_at
		from users
		where company_id = $1 and id = $2
	`, companyID, userID).Scan(&user.ID, &user.CompanyID, &user.Name, &user.Email, &user.Role, &user.Status, &user.CreatedAt)
	return user, err
}

func (s *Store) ListUsers(ctx context.Context, companyID uuid.UUID) ([]User, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, company_id::text, name, email, role, status, created_at
		from users
		where company_id = $1
		order by created_at desc
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var item User
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.Name, &item.Email, &item.Role, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CreateUser(ctx context.Context, companyID uuid.UUID, name, email, role, passwordHash string) (User, error) {
	var user User
	err := s.db.QueryRow(ctx, `
		insert into users(company_id, name, email, password_hash, role, status)
		values($1, $2, lower($3), $4, $5, 'active')
		returning id::text, company_id::text, name, email, role, status, created_at
	`, companyID, name, email, passwordHash, role).Scan(&user.ID, &user.CompanyID, &user.Name, &user.Email, &user.Role, &user.Status, &user.CreatedAt)
	return user, err
}

func (s *Store) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.Exec(ctx, `
		insert into refresh_tokens(user_id, token_hash, expires_at)
		values($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

func (s *Store) FindRefreshTokenUser(ctx context.Context, tokenHash string) (UserWithPassword, error) {
	var user UserWithPassword
	err := s.db.QueryRow(ctx, `
		select u.id::text, u.company_id::text, u.name, u.email, u.role, u.status, u.created_at, u.password_hash
		from refresh_tokens rt
		join users u on u.id = rt.user_id
		where rt.token_hash = $1 and rt.revoked_at is null and rt.expires_at > now()
	`, tokenHash).Scan(&user.ID, &user.CompanyID, &user.Name, &user.Email, &user.Role, &user.Status, &user.CreatedAt, &user.PasswordHash)
	return user, err
}

func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.db.Exec(ctx, `update refresh_tokens set revoked_at = now() where token_hash = $1 and revoked_at is null`, tokenHash)
	return err
}

func (s *Store) CreateCustomer(ctx context.Context, companyID uuid.UUID, name, phone, email, notes string) (Customer, error) {
	var customer Customer
	err := s.db.QueryRow(ctx, `
		insert into customers(company_id, name, phone, email, notes, status)
		values($1, $2, $3, nullif($4, ''), nullif($5, ''), 'active')
		returning id::text, company_id::text, name, phone, coalesce(email, ''), coalesce(notes, ''), status, created_at
	`, companyID, name, phone, email, notes).Scan(&customer.ID, &customer.CompanyID, &customer.Name, &customer.Phone, &customer.Email, &customer.Notes, &customer.Status, &customer.CreatedAt)
	return customer, err
}

func (s *Store) ListCustomers(ctx context.Context, companyID uuid.UUID, search string) ([]Customer, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, company_id::text, name, phone, coalesce(email, ''), coalesce(notes, ''), status, created_at
		from customers
		where company_id = $1
		  and ($2 = '' or name ilike '%' || $2 || '%' or phone ilike '%' || $2 || '%' or coalesce(email, '') ilike '%' || $2 || '%')
		order by created_at desc
	`, companyID, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Customer
	for rows.Next() {
		var item Customer
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.Name, &item.Phone, &item.Email, &item.Notes, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) GetCustomer(ctx context.Context, companyID uuid.UUID, id string) (Customer, error) {
	var customer Customer
	err := s.db.QueryRow(ctx, `
		select id::text, company_id::text, name, phone, coalesce(email, ''), coalesce(notes, ''), status, created_at
		from customers
		where company_id = $1 and id = $2
	`, companyID, id).Scan(&customer.ID, &customer.CompanyID, &customer.Name, &customer.Phone, &customer.Email, &customer.Notes, &customer.Status, &customer.CreatedAt)
	return customer, err
}

func (s *Store) ListDepartments(ctx context.Context, companyID uuid.UUID) ([]Department, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, company_id::text, name, active, created_at
		from departments
		where company_id = $1
		order by name
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Department
	for rows.Next() {
		var item Department
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.Name, &item.Active, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CreateDepartment(ctx context.Context, companyID uuid.UUID, name string) (Department, error) {
	var item Department
	err := s.db.QueryRow(ctx, `
		insert into departments(company_id, name)
		values($1, $2)
		returning id::text, company_id::text, name, active, created_at
	`, companyID, name).Scan(&item.ID, &item.CompanyID, &item.Name, &item.Active, &item.CreatedAt)
	return item, err
}

func (s *Store) CreateConversation(ctx context.Context, companyID uuid.UUID, customerID, departmentID, subject, priority, origin string) (Conversation, error) {
	var conversation Conversation
	err := s.db.QueryRow(ctx, `
		insert into conversations(company_id, customer_id, department_id, status, priority, origin, subject)
		values($1, $2, nullif($3, '')::uuid, 'open', $4, $5, nullif($6, ''))
		returning id::text, company_id::text, customer_id::text, '', '', coalesce(department_id::text, ''), '', status, priority, origin, coalesce(subject, ''), first_response_at, closed_at, created_at
	`, companyID, customerID, departmentID, priority, origin, subject).Scan(
		&conversation.ID, &conversation.CompanyID, &conversation.CustomerID, &conversation.CustomerName,
		&conversation.AssignedUserID, &conversation.DepartmentID, &conversation.DepartmentName,
		&conversation.Status, &conversation.Priority, &conversation.Origin, &conversation.Subject,
		&conversation.FirstResponseAt, &conversation.ClosedAt, &conversation.CreatedAt,
	)
	return conversation, err
}

func (s *Store) ListConversations(ctx context.Context, companyID uuid.UUID, status string) ([]Conversation, error) {
	rows, err := s.db.Query(ctx, `
		select c.id::text, c.company_id::text, c.customer_id::text, cu.name,
		       coalesce(c.assigned_user_id::text, ''), coalesce(u.name, ''),
		       coalesce(c.department_id::text, ''), coalesce(d.name, ''),
		       c.status, c.priority, c.origin, coalesce(c.subject, ''),
		       c.first_response_at, c.closed_at, c.created_at,
		       coalesce(m.content, ''), m.created_at
		from conversations c
		join customers cu on cu.id = c.customer_id
		left join users u on u.id = c.assigned_user_id
		left join departments d on d.id = c.department_id
		left join lateral (
			select content, created_at
			from messages
			where conversation_id = c.id
			order by created_at desc
			limit 1
		) m on true
		where c.company_id = $1 and ($2 = '' or c.status = $2)
		order by coalesce(m.created_at, c.created_at) desc
	`, companyID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Conversation
	for rows.Next() {
		var item Conversation
		if err := rows.Scan(
			&item.ID, &item.CompanyID, &item.CustomerID, &item.CustomerName,
			&item.AssignedUserID, &item.AssignedUserName,
			&item.DepartmentID, &item.DepartmentName,
			&item.Status, &item.Priority, &item.Origin, &item.Subject,
			&item.FirstResponseAt, &item.ClosedAt, &item.CreatedAt,
			&item.LastMessagePreview, &item.LastMessageCreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) GetConversation(ctx context.Context, companyID uuid.UUID, id string) (Conversation, error) {
	var item Conversation
	err := s.db.QueryRow(ctx, `
		select c.id::text, c.company_id::text, c.customer_id::text, cu.name,
		       coalesce(c.assigned_user_id::text, ''), coalesce(u.name, ''),
		       coalesce(c.department_id::text, ''), coalesce(d.name, ''),
		       c.status, c.priority, c.origin, coalesce(c.subject, ''),
		       c.first_response_at, c.closed_at, c.created_at,
		       '', null::timestamptz
		from conversations c
		join customers cu on cu.id = c.customer_id
		left join users u on u.id = c.assigned_user_id
		left join departments d on d.id = c.department_id
		where c.company_id = $1 and c.id = $2
	`, companyID, id).Scan(
		&item.ID, &item.CompanyID, &item.CustomerID, &item.CustomerName,
		&item.AssignedUserID, &item.AssignedUserName,
		&item.DepartmentID, &item.DepartmentName,
		&item.Status, &item.Priority, &item.Origin, &item.Subject,
		&item.FirstResponseAt, &item.ClosedAt, &item.CreatedAt,
		&item.LastMessagePreview, &item.LastMessageCreatedAt,
	)
	return item, err
}

func (s *Store) ListMessages(ctx context.Context, companyID uuid.UUID, conversationID string) ([]Message, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, company_id::text, conversation_id::text, customer_id::text,
		       sender_type, coalesce(sender_id::text, ''), type, content, status, metadata, created_at
		from messages
		where company_id = $1 and conversation_id = $2
		order by created_at asc
	`, companyID, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var item Message
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.ConversationID, &item.CustomerID, &item.SenderType, &item.SenderID, &item.Type, &item.Content, &item.Status, &item.Metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CreateMessage(ctx context.Context, companyID uuid.UUID, conversationID string, senderType string, senderID *uuid.UUID, content string) (Message, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer tx.Rollback(ctx)

	var customerID string
	if err := tx.QueryRow(ctx, `select customer_id::text from conversations where company_id = $1 and id = $2`, companyID, conversationID).Scan(&customerID); err != nil {
		return Message{}, err
	}

	var message Message
	err = tx.QueryRow(ctx, `
		insert into messages(company_id, conversation_id, customer_id, sender_type, sender_id, type, content, status, metadata)
		values($1, $2, $3, $4, $5, 'text', $6, 'sent', '{}'::jsonb)
		returning id::text, company_id::text, conversation_id::text, customer_id::text, sender_type, coalesce(sender_id::text, ''), type, content, status, metadata, created_at
	`, companyID, conversationID, customerID, senderType, senderID, content).Scan(&message.ID, &message.CompanyID, &message.ConversationID, &message.CustomerID, &message.SenderType, &message.SenderID, &message.Type, &message.Content, &message.Status, &message.Metadata, &message.CreatedAt)
	if err != nil {
		return Message{}, err
	}

	if senderType == "agent" {
		_, _ = tx.Exec(ctx, `update conversations set first_response_at = coalesce(first_response_at, now()) where company_id = $1 and id = $2`, companyID, conversationID)
	}
	if err := tx.Commit(ctx); err != nil {
		return Message{}, err
	}
	return message, nil
}

func (s *Store) AssignConversation(ctx context.Context, companyID uuid.UUID, conversationID string, userID uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `
		update conversations set assigned_user_id = $3, status = case when status = 'pending' then 'open' else status end
		where company_id = $1 and id = $2
	`, companyID, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) UpdateConversationStatus(ctx context.Context, companyID uuid.UUID, conversationID, status string) error {
	closedExpr := "closed_at"
	if status == "resolved" || status == "closed" {
		closedExpr = "now()"
	}
	tag, err := s.db.Exec(ctx, `
		update conversations
		set status = $3, closed_at = `+closedExpr+`
		where company_id = $1 and id = $2
	`, companyID, conversationID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) Dashboard(ctx context.Context, companyID uuid.UUID) (Dashboard, error) {
	var out Dashboard
	err := s.db.QueryRow(ctx, `
		select
			(select count(*) from conversations where company_id = $1 and status in ('open','pending')) as open_conversations,
			(select count(*) from conversations where company_id = $1 and closed_at::date = current_date) as closed_today,
			(select count(*) from conversations where company_id = $1 and status = 'pending') as waiting_customers,
			0::bigint as online_agents,
			(select count(*) from messages where company_id = $1 and created_at::date = current_date) as messages_today,
			coalesce((select avg(extract(epoch from (first_response_at - created_at)))::bigint from conversations where company_id = $1 and first_response_at is not null), 0),
			coalesce((select avg(extract(epoch from (closed_at - created_at)))::bigint from conversations where company_id = $1 and closed_at is not null), 0),
			(select count(*) from customers where company_id = $1)
	`, companyID).Scan(&out.OpenConversations, &out.ClosedToday, &out.WaitingCustomers, &out.OnlineAgents, &out.MessagesToday, &out.AvgFirstResponseSecs, &out.AvgConversationSecs, &out.CustomersTotal)
	return out, err
}

func (s *Store) ListQuickReplies(ctx context.Context, companyID uuid.UUID) ([]QuickReply, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, company_id::text, coalesce(department_id::text, ''), title, shortcut, content, created_at
		from quick_replies
		where company_id = $1
		order by shortcut
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QuickReply
	for rows.Next() {
		var item QuickReply
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.DepartmentID, &item.Title, &item.Shortcut, &item.Content, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CreateQuickReply(ctx context.Context, companyID uuid.UUID, departmentID, title, shortcut, content string) (QuickReply, error) {
	var item QuickReply
	err := s.db.QueryRow(ctx, `
		insert into quick_replies(company_id, department_id, title, shortcut, content)
		values($1, nullif($2, '')::uuid, $3, $4, $5)
		returning id::text, company_id::text, coalesce(department_id::text, ''), title, shortcut, content, created_at
	`, companyID, departmentID, title, shortcut, content).Scan(&item.ID, &item.CompanyID, &item.DepartmentID, &item.Title, &item.Shortcut, &item.Content, &item.CreatedAt)
	return item, err
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
