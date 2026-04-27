import { FormEvent, useEffect, useMemo, useState } from "react";
import {
  BarChart3,
  Building2,
  CheckCircle2,
  Headphones,
  LayoutDashboard,
  LogOut,
  MessageSquareText,
  Moon,
  Plus,
  Search,
  Send,
  Settings,
  Sparkles,
  Users
} from "lucide-react";
import { ApiClient, Conversation, Customer, Dashboard, Department, Message, Session, User, wsUrl } from "./lib/api";
import logoUrl from "./assets/logo.svg";

type Page = "dashboard" | "chat" | "customers" | "team" | "settings";

const savedSession = () => {
  const raw = localStorage.getItem("whisper.session");
  return raw ? (JSON.parse(raw) as Session) : null;
};

export function App() {
  const [session, setSessionState] = useState<Session | null>(savedSession);
  const api = useMemo(() => new ApiClient(session), [session]);

  const setSession = (next: Session | null) => {
    if (next) {
      localStorage.setItem("whisper.session", JSON.stringify(next));
    } else {
      localStorage.removeItem("whisper.session");
    }
    setSessionState(next);
  };

  if (!session) {
    return <AuthScreen api={api} onSession={setSession} />;
  }

  return <Workspace api={api} session={session} onLogout={() => setSession(null)} />;
}

function AuthScreen({ api, onSession }: { api: ApiClient; onSession: (session: Session) => void }) {
  const [mode, setMode] = useState<"login" | "register">("login");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    const form = new FormData(event.currentTarget);
    try {
      if (mode === "login") {
        const session = await api.login(String(form.get("email")), String(form.get("password")));
        onSession(session);
      } else {
        const response = await api.registerCompany({
          company_name: String(form.get("company_name")),
          admin_name: String(form.get("admin_name")),
          email: String(form.get("email")),
          password: String(form.get("password"))
        });
        onSession(response.tokens);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Nao foi possivel autenticar");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="grid min-h-screen bg-[#f4f7fb] lg:grid-cols-[1.1fr_0.9fr]">
      <section className="flex items-center justify-center px-6 py-10">
        <div className="w-full max-w-md">
          <img src={logoUrl} alt="Whisper" className="mb-8 h-12 w-auto" />
          <div className="rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
            <div className="mb-6 flex rounded-md bg-slate-100 p-1">
              <button className={tabClass(mode === "login")} onClick={() => setMode("login")} type="button">
                Login
              </button>
              <button className={tabClass(mode === "register")} onClick={() => setMode("register")} type="button">
                Cadastro
              </button>
            </div>
            <form onSubmit={submit} className="space-y-4">
              {mode === "register" && (
                <>
                  <Field label="Empresa" name="company_name" placeholder="Acme Atendimento" />
                  <Field label="Administrador" name="admin_name" placeholder="Ana Silva" />
                </>
              )}
              <Field label="E-mail" name="email" type="email" placeholder="admin@demo.local" />
              <Field label="Senha" name="password" type="password" placeholder="admin12345" />
              {error && <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}
              <button
                disabled={loading}
                className="flex w-full items-center justify-center gap-2 rounded-md bg-brand-600 px-4 py-3 text-sm font-semibold text-white transition hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-70"
              >
                <CheckCircle2 size={18} />
                {loading ? "Processando..." : mode === "login" ? "Entrar" : "Criar empresa"}
              </button>
            </form>
            <p className="mt-4 text-sm text-slate-500">Demo local: admin@demo.local / admin12345</p>
          </div>
        </div>
      </section>
      <section className="hidden bg-ink p-10 text-white lg:flex lg:flex-col lg:justify-between">
        <div className="max-w-xl">
          <div className="mb-8 inline-flex items-center gap-2 rounded-md border border-white/15 px-3 py-2 text-sm text-sky-100">
            <Sparkles size={16} />
            SaaS de atendimento em tempo real
          </div>
          <h1 className="text-5xl font-semibold leading-tight tracking-normal">Whisper</h1>
          <p className="mt-5 text-lg leading-8 text-slate-300">
            Central de suporte multitenant com filas, conversas, clientes, atendentes e mensagens instantaneas.
          </p>
        </div>
        <div className="grid grid-cols-3 gap-4 text-sm text-slate-300">
          <MetricPreview label="WebSocket" value="tempo real" />
          <MetricPreview label="JWT" value="seguro" />
          <MetricPreview label="PostgreSQL" value="multiempresa" />
        </div>
      </section>
    </main>
  );
}

function Workspace({ api, session, onLogout }: { api: ApiClient; session: Session; onLogout: () => void }) {
  const [page, setPage] = useState<Page>("dashboard");
  const [dashboard, setDashboard] = useState<Dashboard | null>(null);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [users, setUsers] = useState<User[]>([session.user]);
  const [selected, setSelected] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [error, setError] = useState("");

  async function loadBase() {
    setError("");
    try {
      const [dash, conv, cust, deps] = await Promise.all([
        api.dashboard(),
        api.conversations(),
        api.customers(),
        api.departments()
      ]);
      setDashboard(dash);
      setConversations(conv.items);
      setCustomers(cust.items);
      setDepartments(deps.items);
      if (!selected && conv.items[0]) {
        setSelected(conv.items[0]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao carregar dados");
    }
  }

  useEffect(() => {
    loadBase();
  }, []);

  useEffect(() => {
    if (!selected) {
      setMessages([]);
      return;
    }
    api.messages(selected.id).then((response) => setMessages(response.items)).catch(() => setMessages([]));
  }, [selected?.id]);

  useEffect(() => {
    let socket: WebSocket | null = null;
    let retry: number | undefined;
    const connect = () => {
      socket = new WebSocket(`${wsUrl()}?token=${encodeURIComponent(session.access_token)}`);
      socket.onmessage = (event) => {
        const data = JSON.parse(event.data) as { type: string; conversation_id?: string; payload: Message };
        if (data.type === "message.created") {
          if (data.conversation_id === selected?.id) {
            setMessages((current) => (current.some((item) => item.id === data.payload.id) ? current : [...current, data.payload]));
          }
          api.conversations().then((response) => setConversations(response.items)).catch(() => undefined);
          api.dashboard().then(setDashboard).catch(() => undefined);
        }
      };
      socket.onclose = () => {
        retry = window.setTimeout(connect, 2000);
      };
    };
    connect();
    return () => {
      if (retry) window.clearTimeout(retry);
      socket?.close();
    };
  }, [session.access_token, selected?.id]);

  return (
    <div className="min-h-screen bg-[#f5f7fb] text-slate-900">
      <aside className="fixed inset-y-0 left-0 hidden w-64 border-r border-slate-200 bg-white px-4 py-5 lg:block">
        <img src={logoUrl} alt="Whisper" className="h-10 w-auto" />
        <nav className="mt-8 space-y-1">
          <NavButton active={page === "dashboard"} icon={<LayoutDashboard size={18} />} label="Dashboard" onClick={() => setPage("dashboard")} />
          <NavButton active={page === "chat"} icon={<MessageSquareText size={18} />} label="Atendimento" onClick={() => setPage("chat")} />
          <NavButton active={page === "customers"} icon={<Users size={18} />} label="Clientes" onClick={() => setPage("customers")} />
          <NavButton active={page === "team"} icon={<Headphones size={18} />} label="Equipe" onClick={() => setPage("team")} />
          <NavButton active={page === "settings"} icon={<Settings size={18} />} label="Configuracoes" onClick={() => setPage("settings")} />
        </nav>
      </aside>
      <div className="lg:pl-64">
        <header className="sticky top-0 z-10 flex h-16 items-center justify-between border-b border-slate-200 bg-white/95 px-4 backdrop-blur lg:px-8">
          <div>
            <p className="text-sm text-slate-500">Central de atendimento</p>
            <h2 className="font-semibold">{pageTitle(page)}</h2>
          </div>
          <div className="flex items-center gap-3">
            <button className="hidden rounded-md border border-slate-200 p-2 text-slate-500 md:block" aria-label="Tema">
              <Moon size={18} />
            </button>
            <div className="text-right">
              <p className="text-sm font-semibold">{session.user.name}</p>
              <p className="text-xs text-slate-500">{session.user.role}</p>
            </div>
            <button onClick={onLogout} className="rounded-md border border-slate-200 p-2 text-slate-600" aria-label="Sair">
              <LogOut size={18} />
            </button>
          </div>
        </header>
        {error && <div className="mx-4 mt-4 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700 lg:mx-8">{error}</div>}
        <main className="p-4 lg:p-8">
          {page === "dashboard" && <DashboardPage data={dashboard} conversations={conversations} />}
          {page === "chat" && (
            <ChatPage
              api={api}
              conversations={conversations}
              customers={customers}
              departments={departments}
              selected={selected}
              messages={messages}
              onSelect={setSelected}
              onRefresh={loadBase}
            />
          )}
          {page === "customers" && <CustomersPage api={api} customers={customers} onRefresh={loadBase} />}
          {page === "team" && <TeamPage users={users} setUsers={setUsers} api={api} />}
          {page === "settings" && <SettingsPage departments={departments} />}
        </main>
      </div>
    </div>
  );
}

function DashboardPage({ data, conversations }: { data: Dashboard | null; conversations: Conversation[] }) {
  const metrics = [
    ["Conversas abertas", data?.open_conversations ?? 0, MessageSquareText],
    ["Encerradas hoje", data?.closed_today ?? 0, CheckCircle2],
    ["Clientes aguardando", data?.waiting_customers ?? 0, Users],
    ["Mensagens hoje", data?.messages_today ?? 0, BarChart3]
  ] as const;
  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {metrics.map(([label, value, Icon]) => (
          <div key={label} className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
            <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-md bg-brand-50 text-brand-700">
              <Icon size={20} />
            </div>
            <p className="text-sm text-slate-500">{label}</p>
            <p className="mt-1 text-3xl font-semibold">{value}</p>
          </div>
        ))}
      </div>
      <section className="rounded-lg border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-200 px-5 py-4">
          <h3 className="font-semibold">Atendimentos recentes</h3>
        </div>
        <div className="divide-y divide-slate-100">
          {conversations.slice(0, 6).map((item) => (
            <div key={item.id} className="flex items-center justify-between px-5 py-4">
              <div>
                <p className="font-medium">{item.customer_name}</p>
                <p className="text-sm text-slate-500">{item.last_message_preview || item.subject || "Sem mensagens"}</p>
              </div>
              <StatusBadge status={item.status} />
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

function ChatPage(props: {
  api: ApiClient;
  conversations: Conversation[];
  customers: Customer[];
  departments: Department[];
  selected: Conversation | null;
  messages: Message[];
  onSelect: (conversation: Conversation) => void;
  onRefresh: () => Promise<void>;
}) {
  const [draft, setDraft] = useState("");
  const [creating, setCreating] = useState(false);

  async function send(event: FormEvent) {
    event.preventDefault();
    if (!props.selected || !draft.trim()) return;
    await props.api.sendMessage(props.selected.id, draft.trim());
    setDraft("");
  }

  async function createConversation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const customerId = String(form.get("customer_id"));
    if (!customerId) return;
    setCreating(true);
    try {
      const conversation = await props.api.createConversation({
        customer_id: customerId,
        department_id: String(form.get("department_id")),
        subject: String(form.get("subject")),
        initial_text: String(form.get("initial_text"))
      });
      await props.onRefresh();
      props.onSelect(conversation);
      event.currentTarget.reset();
    } finally {
      setCreating(false);
    }
  }

  return (
    <div className="grid min-h-[calc(100vh-8rem)] gap-4 xl:grid-cols-[320px_1fr_300px]">
      <section className="rounded-lg border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-200 p-4">
          <div className="flex items-center gap-2 rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-500">
            <Search size={16} />
            Buscar conversas
          </div>
        </div>
        <div className="scrollbar-thin max-h-[70vh] overflow-auto">
          {props.conversations.map((item) => (
            <button
              key={item.id}
              onClick={() => props.onSelect(item)}
              className={`w-full border-b border-slate-100 px-4 py-3 text-left transition hover:bg-slate-50 ${props.selected?.id === item.id ? "bg-brand-50" : ""}`}
            >
              <div className="flex items-center justify-between gap-3">
                <p className="font-medium">{item.customer_name}</p>
                <StatusBadge status={item.status} />
              </div>
              <p className="mt-1 line-clamp-1 text-sm text-slate-500">{item.last_message_preview || item.subject || "Atendimento aberto"}</p>
            </button>
          ))}
        </div>
      </section>

      <section className="flex rounded-lg border border-slate-200 bg-white shadow-sm">
        {props.selected ? (
          <div className="flex min-w-0 flex-1 flex-col">
            <div className="border-b border-slate-200 px-5 py-4">
              <h3 className="font-semibold">{props.selected.customer_name}</h3>
              <p className="text-sm text-slate-500">{props.selected.department_name || "Sem departamento"} · {props.selected.priority}</p>
            </div>
            <div className="scrollbar-thin flex-1 space-y-3 overflow-auto bg-slate-50 p-5">
              {props.messages.map((message) => {
                const agent = message.sender_type === "agent";
                return (
                  <div key={message.id} className={`flex ${agent ? "justify-end" : "justify-start"}`}>
                    <div className={`max-w-[78%] rounded-lg px-4 py-3 text-sm shadow-sm ${agent ? "bg-brand-600 text-white" : "bg-white text-slate-800"}`}>
                      <p>{message.content}</p>
                      <p className={`mt-1 text-[11px] ${agent ? "text-sky-100" : "text-slate-400"}`}>{message.status}</p>
                    </div>
                  </div>
                );
              })}
            </div>
            <form onSubmit={send} className="flex gap-3 border-t border-slate-200 p-4">
              <input value={draft} onChange={(event) => setDraft(event.target.value)} className="flex-1 rounded-md border border-slate-200 px-4 py-3 outline-none focus:border-brand-500" placeholder="Escreva uma mensagem" />
              <button className="flex items-center gap-2 rounded-md bg-brand-600 px-4 py-3 font-semibold text-white">
                <Send size={18} />
                Enviar
              </button>
            </form>
          </div>
        ) : (
          <div className="flex flex-1 items-center justify-center text-slate-500">Selecione uma conversa</div>
        )}
      </section>

      <section className="space-y-4">
        <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
          <h3 className="mb-3 font-semibold">Nova conversa</h3>
          <form onSubmit={createConversation} className="space-y-3">
            <select name="customer_id" className="w-full rounded-md border border-slate-200 px-3 py-2 text-sm">
              <option value="">Cliente</option>
              {props.customers.map((customer) => (
                <option key={customer.id} value={customer.id}>{customer.name}</option>
              ))}
            </select>
            <select name="department_id" className="w-full rounded-md border border-slate-200 px-3 py-2 text-sm">
              <option value="">Departamento</option>
              {props.departments.map((department) => (
                <option key={department.id} value={department.id}>{department.name}</option>
              ))}
            </select>
            <input name="subject" className="w-full rounded-md border border-slate-200 px-3 py-2 text-sm" placeholder="Assunto" />
            <textarea name="initial_text" className="min-h-24 w-full rounded-md border border-slate-200 px-3 py-2 text-sm" placeholder="Mensagem inicial do cliente" />
            <button disabled={creating} className="flex w-full items-center justify-center gap-2 rounded-md bg-ink px-3 py-2 text-sm font-semibold text-white">
              <Plus size={16} />
              Criar
            </button>
          </form>
        </div>
        <CustomerSummary conversation={props.selected} customers={props.customers} />
      </section>
    </div>
  );
}

function CustomersPage({ api, customers, onRefresh }: { api: ApiClient; customers: Customer[]; onRefresh: () => Promise<void> }) {
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    await api.createCustomer({
      name: String(form.get("name")),
      phone: String(form.get("phone")),
      email: String(form.get("email")),
      notes: String(form.get("notes"))
    });
    event.currentTarget.reset();
    await onRefresh();
  }

  return (
    <div className="grid gap-4 xl:grid-cols-[360px_1fr]">
      <form onSubmit={submit} className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <h3 className="mb-4 font-semibold">Cadastrar cliente</h3>
        <div className="space-y-3">
          <Field label="Nome" name="name" />
          <Field label="Telefone" name="phone" />
          <Field label="E-mail" name="email" type="email" required={false} />
          <label className="block text-sm font-medium text-slate-700">
            Observacoes
            <textarea name="notes" className="mt-1 min-h-24 w-full rounded-md border border-slate-200 px-3 py-2 outline-none focus:border-brand-500" />
          </label>
          <button className="flex w-full items-center justify-center gap-2 rounded-md bg-brand-600 px-4 py-3 text-sm font-semibold text-white">
            <Plus size={16} />
            Salvar cliente
          </button>
        </div>
      </form>
      <section className="rounded-lg border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-200 px-5 py-4">
          <h3 className="font-semibold">Clientes</h3>
        </div>
        <div className="divide-y divide-slate-100">
          {customers.map((customer) => (
            <div key={customer.id} className="grid gap-2 px-5 py-4 md:grid-cols-3">
              <p className="font-medium">{customer.name}</p>
              <p className="text-sm text-slate-500">{customer.phone}</p>
              <p className="text-sm text-slate-500">{customer.email || "Sem e-mail"}</p>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

function TeamPage({ users, setUsers, api }: { users: User[]; setUsers: (users: User[]) => void; api: ApiClient }) {
  useEffect(() => {
    api.request<{ items: User[] }>("/users").then((response) => setUsers(response.items)).catch(() => undefined);
  }, []);

  return (
    <section className="rounded-lg border border-slate-200 bg-white shadow-sm">
      <div className="border-b border-slate-200 px-5 py-4">
        <h3 className="font-semibold">Equipe</h3>
      </div>
      <div className="divide-y divide-slate-100">
        {users.map((user) => (
          <div key={user.id} className="flex items-center justify-between px-5 py-4">
            <div>
              <p className="font-medium">{user.name}</p>
              <p className="text-sm text-slate-500">{user.email}</p>
            </div>
            <span className="rounded-md bg-slate-100 px-2 py-1 text-xs font-semibold text-slate-600">{user.role}</span>
          </div>
        ))}
      </div>
    </section>
  );
}

function SettingsPage({ departments }: { departments: Department[] }) {
  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <div className="mb-4 flex items-center gap-2">
          <Building2 size={18} />
          <h3 className="font-semibold">Departamentos</h3>
        </div>
        <div className="flex flex-wrap gap-2">
          {departments.map((department) => (
            <span key={department.id} className="rounded-md bg-brand-50 px-3 py-2 text-sm font-medium text-brand-700">{department.name}</span>
          ))}
        </div>
      </section>
      <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <h3 className="font-semibold">Roadmap preparado</h3>
        <p className="mt-2 text-sm leading-6 text-slate-600">
          Tags, notificacoes, webchat publico e integracoes oficiais futuras ja possuem schema e fronteiras de API para evolucao incremental.
        </p>
      </section>
    </div>
  );
}

function CustomerSummary({ conversation, customers }: { conversation: Conversation | null; customers: Customer[] }) {
  const customer = customers.find((item) => item.id === conversation?.customer_id);
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
      <h3 className="mb-3 font-semibold">Dados do cliente</h3>
      {customer ? (
        <div className="space-y-2 text-sm">
          <p className="font-medium">{customer.name}</p>
          <p className="text-slate-500">{customer.phone}</p>
          <p className="text-slate-500">{customer.email || "Sem e-mail"}</p>
          <p className="rounded-md bg-slate-50 p-3 text-slate-600">{customer.notes || "Sem observacoes"}</p>
        </div>
      ) : (
        <p className="text-sm text-slate-500">Selecione uma conversa</p>
      )}
    </div>
  );
}

function Field({ label, name, type = "text", placeholder = "", required = true }: { label: string; name: string; type?: string; placeholder?: string; required?: boolean }) {
  return (
    <label className="block text-sm font-medium text-slate-700">
      {label}
      <input required={required} name={name} type={type} placeholder={placeholder} className="mt-1 w-full rounded-md border border-slate-200 px-3 py-2 outline-none transition focus:border-brand-500" />
    </label>
  );
}

function NavButton({ active, icon, label, onClick }: { active: boolean; icon: React.ReactNode; label: string; onClick: () => void }) {
  return (
    <button onClick={onClick} className={`flex w-full items-center gap-3 rounded-md px-3 py-2.5 text-sm font-medium transition ${active ? "bg-ink text-white" : "text-slate-600 hover:bg-slate-100"}`}>
      {icon}
      {label}
    </button>
  );
}

function StatusBadge({ status }: { status: string }) {
  const styles: Record<string, string> = {
    open: "bg-emerald-50 text-emerald-700",
    pending: "bg-amber-50 text-amber-700",
    resolved: "bg-sky-50 text-sky-700",
    closed: "bg-slate-100 text-slate-600"
  };
  return <span className={`rounded-md px-2 py-1 text-xs font-semibold ${styles[status] ?? styles.closed}`}>{status}</span>;
}

function MetricPreview({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-white/10 p-4">
      <p className="text-xs uppercase tracking-normal text-slate-500">{label}</p>
      <p className="mt-2 font-semibold text-white">{value}</p>
    </div>
  );
}

function tabClass(active: boolean) {
  return `flex-1 rounded-md px-3 py-2 text-sm font-semibold transition ${active ? "bg-white text-ink shadow-sm" : "text-slate-500"}`;
}

function pageTitle(page: Page) {
  const titles: Record<Page, string> = {
    dashboard: "Dashboard",
    chat: "Atendimento",
    customers: "Clientes",
    team: "Equipe",
    settings: "Configuracoes"
  };
  return titles[page];
}
