create extension if not exists pgcrypto;

create table companies (
    id uuid primary key default gen_random_uuid(),
    name text not null,
    cnpj text,
    plan text not null default 'starter',
    status text not null default 'active',
    config jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table roles (
    code text primary key,
    name text not null,
    description text not null
);

insert into roles(code, name, description) values
    ('ADMIN', 'Administrador', 'Gerencia empresa, usuarios, departamentos e relatorios'),
    ('SUPERVISOR', 'Supervisor', 'Acompanha atendimentos, transfere conversas e ve relatorios'),
    ('ATENDENTE', 'Atendente', 'Atende clientes e responde conversas atribuidas'),
    ('VISUALIZADOR', 'Visualizador', 'Visualiza historico e relatorios')
on conflict (code) do nothing;

create table users (
    id uuid primary key default gen_random_uuid(),
    company_id uuid not null references companies(id) on delete cascade,
    name text not null,
    email text not null unique,
    password_hash text not null,
    role text not null references roles(code),
    status text not null default 'active' check (status in ('active', 'inactive')),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table departments (
    id uuid primary key default gen_random_uuid(),
    company_id uuid not null references companies(id) on delete cascade,
    name text not null,
    active boolean not null default true,
    created_at timestamptz not null default now(),
    unique(company_id, name)
);

create table customers (
    id uuid primary key default gen_random_uuid(),
    company_id uuid not null references companies(id) on delete cascade,
    name text not null,
    phone text not null,
    email text,
    notes text,
    status text not null default 'active' check (status in ('active', 'inactive', 'blocked')),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique(company_id, phone)
);

create table conversations (
    id uuid primary key default gen_random_uuid(),
    company_id uuid not null references companies(id) on delete cascade,
    customer_id uuid not null references customers(id) on delete restrict,
    assigned_user_id uuid references users(id) on delete set null,
    department_id uuid references departments(id) on delete set null,
    status text not null default 'open' check (status in ('open', 'pending', 'resolved', 'closed')),
    priority text not null default 'normal' check (priority in ('low', 'normal', 'high', 'urgent')),
    origin text not null default 'panel',
    subject text,
    first_response_at timestamptz,
    closed_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table messages (
    id uuid primary key default gen_random_uuid(),
    company_id uuid not null references companies(id) on delete cascade,
    conversation_id uuid not null references conversations(id) on delete cascade,
    customer_id uuid not null references customers(id) on delete restrict,
    sender_type text not null check (sender_type in ('agent', 'customer', 'system')),
    sender_id uuid,
    type text not null default 'text' check (type in ('text', 'image', 'file', 'audio')),
    content text not null,
    status text not null default 'sent' check (status in ('sent', 'delivered', 'read')),
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create table quick_replies (
    id uuid primary key default gen_random_uuid(),
    company_id uuid not null references companies(id) on delete cascade,
    department_id uuid references departments(id) on delete set null,
    title text not null,
    shortcut text not null,
    content text not null,
    created_at timestamptz not null default now(),
    unique(company_id, shortcut)
);

create table tags (
    id uuid primary key default gen_random_uuid(),
    company_id uuid not null references companies(id) on delete cascade,
    name text not null,
    color text not null default '#2563eb',
    created_at timestamptz not null default now(),
    unique(company_id, name)
);

create table conversation_tags (
    conversation_id uuid not null references conversations(id) on delete cascade,
    tag_id uuid not null references tags(id) on delete cascade,
    primary key(conversation_id, tag_id)
);

create table customer_tags (
    customer_id uuid not null references customers(id) on delete cascade,
    tag_id uuid not null references tags(id) on delete cascade,
    primary key(customer_id, tag_id)
);

create table notifications (
    id uuid primary key default gen_random_uuid(),
    company_id uuid not null references companies(id) on delete cascade,
    user_id uuid references users(id) on delete cascade,
    type text not null,
    title text not null,
    body text,
    read_at timestamptz,
    created_at timestamptz not null default now()
);

create table audit_logs (
    id uuid primary key default gen_random_uuid(),
    company_id uuid references companies(id) on delete cascade,
    user_id uuid references users(id) on delete set null,
    action text not null,
    resource_type text not null,
    resource_id uuid,
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create table refresh_tokens (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    token_hash text not null unique,
    expires_at timestamptz not null,
    revoked_at timestamptz,
    created_at timestamptz not null default now()
);

create index idx_users_company on users(company_id);
create index idx_customers_company on customers(company_id);
create index idx_conversations_company_status on conversations(company_id, status);
create index idx_conversations_assigned on conversations(company_id, assigned_user_id);
create index idx_messages_conversation on messages(company_id, conversation_id, created_at);
create index idx_refresh_tokens_hash on refresh_tokens(token_hash);
