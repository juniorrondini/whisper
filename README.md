# Whisper

![Whisper](assets/readme-cover.svg)

Whisper e uma plataforma SaaS de atendimento ao cliente em tempo real. O projeto nao usa marca, API privada, engenharia reversa ou identidade visual do WhatsApp. A arquitetura esta preparada para canais oficiais futuros, mas o MVP implementa apenas painel interno e webchat/canal interno.

## Stack

- Backend: Go, Gin, WebSocket, PostgreSQL, Redis, JWT, bcrypt
- Frontend: React, TypeScript, Vite, Tailwind CSS
- Infra local: Docker Compose, PostgreSQL, Redis, API Go e frontend estatico
- Assets: icone original, logo SVG, favicon ICO/SVG e imagem do README em `assets/`

## Arquitetura

```text
backend/
  cmd/server              entrada HTTP
  internal/config         variaveis de ambiente
  internal/database       conexoes e migrations
  internal/auth           JWT, refresh token e bcrypt
  internal/middleware     auth, CORS, rate limit
  internal/repository     acesso PostgreSQL
  internal/service        regras de negocio
  internal/handler        handlers REST
  internal/websocket      hub WebSocket
  migrations              schema SQL
frontend/
  src                     React + Tailwind
assets/                   marca e imagens
scripts/                  setup e geracao do favicon
```

O isolamento multiempresa e aplicado por `company_id` nas consultas do backend. Usuarios autenticados recebem `company_id`, `user_id` e `role` no JWT.

## Funcionalidades do MVP

- Cadastro de empresa com usuario ADMIN
- Login com JWT access token e refresh token revogavel
- Hash de senha com bcrypt
- Rate limit basico no login via Redis
- Middleware de autenticacao e permissoes
- Empresas, usuarios, departamentos, clientes, conversas e mensagens
- Conversas/tickets com status, prioridade, departamento e responsavel
- Mensagens em tempo real via WebSocket
- Dashboard basico
- Tela de atendimento com lista de conversas, chat e dados do cliente
- Migrations SQL para tabelas principais
- Seed local com `admin@demo.local` / `admin12345`

## Ainda marcado para evolucao

- Webchat publico embutivel por script
- Filas com distribuicao automatica por departamento
- Tags editaveis pela interface
- Notificacoes persistidas na UI
- Relatorios avancados e ranking de atendentes
- Recuperacao de senha com provedor de e-mail
- Integracoes oficiais futuras: WhatsApp Business API, Instagram, Messenger, Telegram, e-mail e webhooks

## Rodando com Docker

Crie o `.env`:

```bash
cp .env.example .env
```

Suba tudo:

```bash
docker compose up --build
```

Acesse:

- Frontend: http://localhost:5173
- Backend healthcheck: http://localhost:8080/health
- PostgreSQL: localhost:5432
- Redis: localhost:6379

Credenciais seed:

```text
admin@demo.local
admin12345
```

## Rodando em desenvolvimento

Backend:

```bash
cd backend
go mod download
go run ./cmd/server
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

## Variaveis principais

Veja `.env.example`.

```env
DATABASE_URL=postgres://whisper:whisper@localhost:5432/whisper?sslmode=disable
REDIS_ADDR=localhost:6379
JWT_SECRET=change-this-local-secret
CORS_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
AUTO_MIGRATE=true
SEED_DEMO=true
```

## Endpoints principais

- `POST /api/auth/register-company`
- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `POST /api/auth/logout`
- `GET /api/me`
- `GET /api/dashboard`
- `GET/POST /api/customers`
- `GET/POST /api/conversations`
- `GET/POST /api/conversations/:id/messages`
- `POST /api/conversations/:id/assign`
- `POST /api/conversations/:id/status`
- `GET/POST /api/departments`
- `GET/POST /api/quick-replies`
- `GET /ws?token=<access_token>`

## Validacao

Com Go e Docker instalados:

```bash
cd backend
go test ./...
go run ./cmd/server
```

Frontend:

```bash
cd frontend
npm run build
npm audit --audit-level=moderate
```

## Observacoes de seguranca

- Troque `JWT_SECRET` antes de qualquer ambiente compartilhado.
- Use HTTPS em producao.
- Configure CORS por dominio real.
- O hub WebSocket atual e em memoria. Para multiplas replicas, use Redis Pub/Sub ou outro barramento.
- Refresh tokens sao armazenados como hash e podem ser revogados.
