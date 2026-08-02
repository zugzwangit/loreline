export type ConnectionState = "checking" | "live" | "offline";

export type View =
  | "overview"
  | "library"
  | "review"
  | "assistant"
  | "sources"
  | "operations"
  | "audit"
  | "settings";

export type KnowledgeItem = {
  id: string;
  title: string;
  source: string;
  status: "Approved" | "Needs review" | "Rejected";
  owner: string;
  updated: string;
  confidence: number;
  content?: string;
};

export type LiveAnswer = {
  answer: string;
  confidence: number;
  grounded: boolean;
  citations: { id: string; title: string; source: string }[];
  message_id?: string;
  preview?: boolean;
};

export type Dashboard = {
  documents?: number;
  approved_knowledge?: number;
  needs_review?: number;
  active_jobs?: number;
  answer_quality?: number;
  questions_resolved?: number;
  freshness?: number;
  workers_online?: number;
  workspace_name?: string;
};

export type Session = {
  displayName: string;
  email: string;
};

export type SourceItem = {
  id: string;
  kind: string;
  name: string;
  status: string;
  last_synced_at?: string | null;
  last_error?: string | null;
};
