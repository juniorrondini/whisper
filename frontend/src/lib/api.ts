export type User = {
  id: string;
  company_id: string;
  name: string;
  email: string;
  role: "ADMIN" | "SUPERVISOR" | "ATENDENTE" | "VISUALIZADOR";
  status: string;
};

export type Customer = {
  id: string;
  name: string;
  phone: string;
  email?: string;
  notes?: string;
  status: string;
};

export type Conversation = {
  id: string;
  customer_id: string;
  customer_name: string;
  assigned_user_id?: string;
  assigned_user_name?: string;
  department_id?: string;
  department_name?: string;
  status: string;
  priority: string;
  origin: string;
  subject?: string;
  last_message_preview?: string;
  created_at: string;
};

export type Message = {
  id: string;
  conversation_id: string;
  sender_type: "agent" | "customer" | "system";
  sender_id?: string;
  content: string;
  status: string;
  created_at: string;
};

export type Department = {
  id: string;
  name: string;
  active: boolean;
};

export type Dashboard = {
  open_conversations: number;
  closed_today: number;
  waiting_customers: number;
  online_agents: number;
  messages_today: number;
  avg_first_response_seconds: number;
  avg_conversation_seconds: number;
  customers_total: number;
};

const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080/api";

export type Session = {
  access_token: string;
  refresh_token: string;
  user: User;
};

export class ApiClient {
  session: Session | null;

  constructor(session: Session | null) {
    this.session = session;
  }

  setSession(session: Session | null) {
    this.session = session;
  }

  async request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(`${API_URL}${path}`, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...(this.session?.access_token ? { Authorization: `Bearer ${this.session.access_token}` } : {}),
        ...(options.headers ?? {})
      }
    });
    if (!response.ok) {
      const body = await response.json().catch(() => ({ error: "Erro inesperado" }));
      throw new Error(body.error ?? "Erro inesperado");
    }
    if (response.status === 204) {
      return undefined as T;
    }
    return response.json() as Promise<T>;
  }

  login(email: string, password: string) {
    return this.request<Session>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password })
    });
  }

  registerCompany(data: { company_name: string; admin_name: string; email: string; password: string; cnpj?: string }) {
    return this.request<{ tokens: Session }>("/auth/register-company", {
      method: "POST",
      body: JSON.stringify(data)
    });
  }

  me() {
    return this.request<User>("/me");
  }

  dashboard() {
    return this.request<Dashboard>("/dashboard");
  }

  conversations() {
    return this.request<{ items: Conversation[] }>("/conversations");
  }

  messages(conversationId: string) {
    return this.request<{ items: Message[] }>(`/conversations/${conversationId}/messages`);
  }

  sendMessage(conversationId: string, content: string) {
    return this.request<Message>(`/conversations/${conversationId}/messages`, {
      method: "POST",
      body: JSON.stringify({ content })
    });
  }

  customers(q = "") {
    return this.request<{ items: Customer[] }>(`/customers${q ? `?q=${encodeURIComponent(q)}` : ""}`);
  }

  createCustomer(data: Pick<Customer, "name" | "phone" | "email" | "notes">) {
    return this.request<Customer>("/customers", {
      method: "POST",
      body: JSON.stringify(data)
    });
  }

  createConversation(data: { customer_id: string; department_id?: string; subject?: string; initial_text?: string }) {
    return this.request<Conversation>("/conversations", {
      method: "POST",
      body: JSON.stringify({ ...data, priority: "normal", origin: "panel" })
    });
  }

  departments() {
    return this.request<{ items: Department[] }>("/departments");
  }
}

export const wsUrl = () => import.meta.env.VITE_WS_URL ?? "ws://localhost:8080/ws";
