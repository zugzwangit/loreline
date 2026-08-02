"use client";

import { useEffect, useMemo, useState } from "react";
import { AuditLog, Settings } from "../management/Management";
import { Operations, Sources } from "../operations/Operations";
import { demoKnowledge, demoSources } from "../../lib/demo-data";
import type {
  ConnectionState,
  Dashboard,
  KnowledgeItem,
  LiveAnswer,
  Session,
  SourceItem,
  View,
} from "../../lib/domain";

function Mark() {
  return (
    <span className="mark" aria-hidden="true">
      <i />
      <i />
      <i />
    </span>
  );
}
function Icon({ children }: { children: React.ReactNode }) {
  return (
    <span className="nav-icon" aria-hidden="true">
      {children}
    </span>
  );
}
function initials(name: string) {
  return (
    name
      .split(/\s+/)
      .map((part) => part[0])
      .join("")
      .slice(0, 2)
      .toUpperCase() || "LO"
  );
}

export default function Workspace() {
  const [view, setView] = useState<View>("overview");
  const [reviewed, setReviewed] = useState<string[]>([]);
  const [query, setQuery] = useState("");
  const [asked, setAsked] = useState(false);
  const [knowledge, setKnowledge] = useState<KnowledgeItem[]>(demoKnowledge);
  const [connection, setConnection] = useState<ConnectionState>("checking");
  const [dashboard, setDashboard] = useState<Dashboard>({});
  const [sources, setSources] = useState<SourceItem[]>(demoSources);
  const [session, setSession] = useState<Session>({
    displayName: "Demo operator",
    email: "preview@loreline.local",
  });
  const [today] = useState(() =>
    new Intl.DateTimeFormat(undefined, {
      weekday: "long",
      month: "long",
      day: "numeric",
    }).format(new Date()),
  );
  useEffect(() => {
    void fetch("/api/session")
      .then((r) => r.json())
      .then(setSession)
      .catch(() => {});
    let active = true;
    void (async () => {
      try {
        const [d, c, s] = await Promise.all([
          fetch("/api/loreline/v1/dashboard"),
          fetch("/api/loreline/v1/candidates?limit=200"),
          fetch("/api/loreline/v1/sources"),
        ]);
        if (!d.ok || !c.ok || !s.ok) throw new Error("api unavailable");
        const stats = await d.json();
        const candidates = await c.json();
        const sourceData = await s.json();
        if (!active) return;
        setDashboard(stats);
        setKnowledge(
          (candidates.items || []).map(
            (item: {
              id: string;
              title: string;
              source: string;
              status: string;
              confidence: number;
              content: string;
              created_at: string;
            }) => ({
              id: item.id,
              title: item.title,
              source: item.source,
              status:
                item.status === "pending"
                  ? "Needs review"
                  : item.status === "approved"
                    ? "Approved"
                    : "Rejected",
              owner: "Knowledge Ops",
              updated: new Date(item.created_at).toLocaleString(),
              confidence: Math.round(item.confidence * 100),
              content: item.content,
            }),
          ),
        );
        setSources(sourceData.items || []);
        setConnection("live");
      } catch {
        if (active) {
          setKnowledge(demoKnowledge);
          setSources(demoSources);
          setConnection("offline");
        }
      }
    })();
    return () => {
      active = false;
    };
  }, []);
  useEffect(() => {
    const key = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setView("assistant");
        requestAnimationFrame(() =>
          document
            .querySelector<HTMLTextAreaElement>(".ask-box textarea")
            ?.focus(),
        );
      }
    };
    window.addEventListener("keydown", key);
    return () => window.removeEventListener("keydown", key);
  }, []);
  const pending = useMemo(
    () =>
      knowledge.filter(
        (item) => item.status === "Needs review" && !reviewed.includes(item.id),
      ),
    [knowledge, reviewed],
  );
  const workspace = dashboard.workspace_name || "Acme Labs";
  const nav: {
    id: View;
    label: string;
    icon: string;
    group: "workspace" | "manage";
  }[] = [
    { id: "overview", label: "Overview", icon: "⌂", group: "workspace" },
    { id: "library", label: "Knowledge", icon: "◇", group: "workspace" },
    { id: "review", label: "Review queue", icon: "✓", group: "workspace" },
    { id: "assistant", label: "Ask Loreline", icon: "✦", group: "workspace" },
    { id: "sources", label: "Sources", icon: "⌁", group: "manage" },
    { id: "operations", label: "Operations", icon: "◎", group: "manage" },
    { id: "audit", label: "Audit log", icon: "▤", group: "manage" },
    { id: "settings", label: "Settings", icon: "⚙", group: "manage" },
  ];
  const navButton = (item: (typeof nav)[number]) => (
    <button
      key={item.id}
      className={view === item.id ? "active" : ""}
      aria-current={view === item.id ? "page" : undefined}
      onClick={() => setView(item.id)}
    >
      <Icon>{item.icon}</Icon>
      <span className="nav-label">{item.label}</span>
      {item.id === "review" && pending.length > 0 && (
        <span className="count">{pending.length}</span>
      )}
    </button>
  );
  return (
    <main className="shell">
      <aside className="sidebar">
        <div className="brand">
          <Mark />
          <span>Loreline</span>
        </div>
        <div className="workspace">
          <span className="workspace-logo">{workspace[0]}</span>
          <div>
            <b>{workspace}</b>
            <small>Knowledge operations</small>
          </div>
        </div>
        <nav aria-label="Primary navigation">
          <p className="eyebrow">Workspace</p>
          {nav.filter((item) => item.group === "workspace").map(navButton)}
          <p className="eyebrow manage">Manage</p>
          {nav.filter((item) => item.group === "manage").map(navButton)}
        </nav>
        <div className="sidebar-foot">
          <div className={`health-dot ${connection}`} />
          <div>
            <b>
              {connection === "live"
                ? "Production API connected"
                : connection === "checking"
                  ? "Checking services"
                  : "Interactive demo mode"}
            </b>
            <small>
              {connection === "live"
                ? `${dashboard.workers_online ?? 0} workers online · durable writes active`
                : "Sample data · writes disabled"}
            </small>
          </div>
        </div>
        <div className="profile">
          <span>{initials(session.displayName)}</span>
          <div>
            <b>{session.displayName}</b>
            <small>{session.email}</small>
          </div>
          <a href="/signout-with-chatgpt?return_to=/" aria-label="Sign out">
            ↗
          </a>
        </div>
      </aside>
      <section className="content">
        <header className="topbar">
          <div className="breadcrumbs">
            {workspace}
            <span>/</span>
            {nav.find((item) => item.id === view)?.label}
          </div>
          <div className="top-actions">
            <button className="search" onClick={() => setView("assistant")}>
              ⌕<span>Search anything</span>
              <kbd>Ctrl K</kbd>
            </button>
            <button
              className="round"
              onClick={() => setView("audit")}
              aria-label="Open audit log"
            >
              ▤<i />
            </button>
            <button className="primary" onClick={() => setView("assistant")}>
              ✦ Ask Loreline
            </button>
          </div>
        </header>
        {view === "overview" && (
          <Overview
            setView={setView}
            knowledge={knowledge}
            dashboard={dashboard}
            sources={sources}
            today={today}
            connection={connection}
          />
        )}{" "}
        {view === "library" && (
          <Library knowledge={knowledge} onAdd={() => setView("sources")} />
        )}{" "}
        {view === "review" && (
          <Review
            pending={pending}
            reviewed={reviewed}
            setReviewed={setReviewed}
            connection={connection}
          />
        )}{" "}
        {view === "assistant" && (
          <Assistant
            query={query}
            setQuery={setQuery}
            asked={asked}
            setAsked={setAsked}
            connection={connection}
          />
        )}{" "}
        {view === "sources" && (
          <Sources
            items={sources}
            connection={connection}
            onCreated={(item) => setSources((current) => [item, ...current])}
          />
        )}{" "}
        {view === "operations" && <Operations connection={connection} />}{" "}
        {view === "audit" && <AuditLog connection={connection} />}{" "}
        {view === "settings" && <Settings connection={connection} />}
      </section>
    </main>
  );
}

function Overview({
  setView,
  knowledge,
  dashboard,
  sources,
  today,
  connection,
}: {
  setView: (view: View) => void;
  knowledge: KnowledgeItem[];
  dashboard: Dashboard;
  sources: SourceItem[];
  today: string;
  connection: ConnectionState;
}) {
  const approved =
    dashboard.approved_knowledge ??
    knowledge.filter((item) => item.status === "Approved").length;
  const review =
    dashboard.needs_review ??
    knowledge.filter((item) => item.status === "Needs review").length;
  const documents = dashboard.documents ?? knowledge.length;
  const freshness = dashboard.freshness ?? 94;
  const hour = new Date().getHours();
  const greeting = `Good ${hour < 12 ? "morning" : hour < 18 ? "afternoon" : "evening"}.`;
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <p className="kicker">{today.toUpperCase()}</p>
          <h1>{greeting}</h1>
          <p>
            {review
              ? `${review} candidates need a decision.`
              : "Your review queue is clear."}
          </p>
        </div>
        <button className="outline" onClick={() => setView("review")}>
          Open review queue <span>→</span>
        </button>
      </div>
      <div className="metric-grid">
        <Metric
          label="Approved knowledge"
          value={approved.toLocaleString()}
          delta="Reviewed and indexed"
          tone="violet"
          spark={[20, 28, 25, 39, 42, 55, 58, 69, 76, 82]}
        />
        <Metric
          label="Questions resolved"
          value={(dashboard.questions_resolved ?? 1284).toLocaleString()}
          delta="Persisted assistant answers"
          tone="green"
          spark={[26, 34, 31, 46, 43, 59, 55, 70, 68, 87]}
        />
        <Metric
          label="Answer quality"
          value={`${(dashboard.answer_quality ?? 96.8).toFixed(1)}%`}
          delta="From employee feedback"
          tone="blue"
          spark={[44, 46, 51, 50, 57, 61, 66, 69, 72, 78]}
        />
        <Metric
          label="Needs review"
          value={review.toLocaleString()}
          delta={`${dashboard.active_jobs ?? 4} active jobs`}
          tone="amber"
          spark={[72, 65, 66, 55, 59, 46, 41, 35, 30, 24]}
        />
      </div>
      <div className="main-grid">
        <section className="panel pipeline">
          <PanelTitle
            title="Knowledge pipeline"
            subtitle="Content moving from source to trusted answer"
            action="View library"
            onAction={() => setView("library")}
          />
          <div className="pipeline-flow">
            <Stage
              icon="↧"
              label="Ingested"
              value={documents.toLocaleString()}
              sub={`${sources.length} connected sources`}
              tone="slate"
            />
            <Arrow />
            <Stage
              icon="≋"
              label="Normalized"
              value={Math.max(0, documents - review).toLocaleString()}
              sub="Deduplicated records"
              tone="blue"
            />
            <Arrow />
            <Stage
              icon="✓"
              label="Approved"
              value={approved.toLocaleString()}
              sub={`${review} await review`}
              tone="violet"
            />
            <Arrow />
            <Stage
              icon="⌕"
              label="Indexed"
              value={approved.toLocaleString()}
              sub="Ready for answers"
              tone="green"
            />
          </div>
          <div className="freshness">
            <div>
              <span>Knowledge freshness</span>
              <b>{Math.round(freshness)}%</b>
            </div>
            <div className="bar">
              <i
                style={{ width: `${Math.max(0, Math.min(100, freshness))}%` }}
              />
            </div>
            <p>Approved knowledge verified in the last 90 days.</p>
          </div>
        </section>
        <section className="panel coverage">
          <PanelTitle
            title="Source coverage"
            subtitle="Health across connected systems"
            action="Manage"
            onAction={() => setView("sources")}
          />
          {sources.slice(0, 4).map((source, index) => (
            <div className="source" key={source.id}>
              <span
                className={`source-icon ${["violet", "blue", "green"][index % 3]}`}
              >
                {source.name[0]}
              </span>
              <div className="source-info">
                <b>{source.name}</b>
                <small>{source.kind}</small>
              </div>
              <div className="source-health">
                <span>
                  {source.last_error
                    ? "Needs attention"
                    : source.last_synced_at
                      ? `Synced ${new Date(source.last_synced_at).toLocaleDateString()}`
                      : "Not synced"}
                </span>
                <div>
                  <i style={{ width: source.last_error ? "25%" : "100%" }} />
                </div>
              </div>
            </div>
          ))}
          {!sources.length && (
            <div className="table-empty">No sources connected.</div>
          )}
        </section>
      </div>
      <div className="bottom-grid">
        <section className="panel review-preview">
          <PanelTitle
            title="Review queue"
            subtitle="Candidates that need a human decision"
            action="Review all"
            onAction={() => setView("review")}
          />
          <table>
            <thead>
              <tr>
                <th>Knowledge candidate</th>
                <th>Source</th>
                <th>Confidence</th>
                <th>Age</th>
              </tr>
            </thead>
            <tbody>
              {knowledge
                .filter((item) => item.status === "Needs review")
                .slice(0, 5)
                .map((item) => (
                  <tr
                    key={item.id}
                    tabIndex={0}
                    onClick={() => setView("review")}
                    onKeyDown={(event) => {
                      if (event.key === "Enter") setView("review");
                    }}
                  >
                    <td>
                      <b>{item.title}</b>
                      <small>
                        {item.owner} · {item.id}
                      </small>
                    </td>
                    <td>
                      <span className="tag">{item.source}</span>
                    </td>
                    <td>
                      <span className="confidence">
                        <i style={{ width: `${item.confidence}%` }} />
                      </span>
                      {item.confidence}%
                    </td>
                    <td>{item.updated}</td>
                  </tr>
                ))}
            </tbody>
          </table>
          {!review && (
            <div className="table-empty">Everything is reviewed.</div>
          )}
        </section>
        <section className="panel activity">
          <PanelTitle
            title="System posture"
            subtitle="Current production signals"
            action="Operations"
            onAction={() => setView("operations")}
          />
          <div className="activity-row">
            <time>API</time>
            <span className="activity-dot" />
            <div>
              <b>
                {connection === "live"
                  ? dashboard.workers_online
                    ? "Workers online"
                    : "Workers unavailable"
                  : "Demo preview"}
              </b>
              <small>
                {dashboard.workers_online ?? 0} active worker heartbeats
              </small>
            </div>
          </div>
          <div className="activity-row">
            <time>QUEUE</time>
            <span className="activity-dot" />
            <div>
              <b>{dashboard.active_jobs ?? 0} active jobs</b>
              <small>Queued, running, or retrying</small>
            </div>
          </div>
          <div className="activity-row">
            <time>DATA</time>
            <span className="activity-dot" />
            <div>
              <b>{approved} approved records</b>
              <small>Available to the answer service</small>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}

function Metric({
  label,
  value,
  delta,
  tone,
  spark,
}: {
  label: string;
  value: string;
  delta: string;
  tone: string;
  spark: number[];
}) {
  return (
    <article className="metric">
      <div>
        <span>{label}</span>
        <b>{value}</b>
        <small className={tone}>{delta}</small>
      </div>
      <div className={`spark ${tone}`} aria-hidden="true">
        {spark.map((height, index) => (
          <i key={index} style={{ height: `${height}%` }} />
        ))}
      </div>
    </article>
  );
}
function PanelTitle({
  title,
  subtitle,
  action,
  onAction,
}: {
  title: string;
  subtitle: string;
  action: string;
  onAction: () => void;
}) {
  return (
    <div className="panel-title">
      <div>
        <h2>{title}</h2>
        <p>{subtitle}</p>
      </div>
      <button onClick={onAction}>
        {action}
        <span>→</span>
      </button>
    </div>
  );
}
function Stage({
  icon,
  label,
  value,
  sub,
  tone,
}: {
  icon: string;
  label: string;
  value: string;
  sub: string;
  tone: string;
}) {
  return (
    <div className="stage">
      <span className={`stage-icon ${tone}`}>{icon}</span>
      <div>
        <small>{label}</small>
        <b>{value}</b>
        <p>{sub}</p>
      </div>
    </div>
  );
}
function Arrow() {
  return <span className="arrow">→</span>;
}

function Library({
  knowledge,
  onAdd,
}: {
  knowledge: KnowledgeItem[];
  onAdd: () => void;
}) {
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [source, setSource] = useState("all");
  const sources = [...new Set(knowledge.map((item) => item.source))];
  const filtered = knowledge.filter(
    (item) =>
      (status === "all" || item.status === status) &&
      (source === "all" || item.source === source) &&
      `${item.title} ${item.owner} ${item.id}`
        .toLowerCase()
        .includes(search.toLowerCase()),
  );
  return (
    <div className="page">
      <div className="page-heading compact">
        <div>
          <p className="kicker">KNOWLEDGE LIBRARY</p>
          <h1>One source of truth</h1>
          <p>
            Approved, traceable knowledge ready for every employee question.
          </p>
        </div>
        <button className="primary" onClick={onAdd}>
          + Add source
        </button>
      </div>
      <section className="panel library">
        <div className="library-tools">
          <label className="filter-input">
            <span aria-hidden="true">⌕</span>
            <input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search titles, owners, or IDs"
              aria-label="Search knowledge"
            />
          </label>
          <select
            aria-label="Filter by status"
            value={status}
            onChange={(event) => setStatus(event.target.value)}
          >
            <option value="all">All statuses</option>
            <option>Approved</option>
            <option>Needs review</option>
            <option>Rejected</option>
          </select>
          <select
            aria-label="Filter by source"
            value={source}
            onChange={(event) => setSource(event.target.value)}
          >
            <option value="all">All sources</option>
            {sources.map((value) => (
              <option key={value}>{value}</option>
            ))}
          </select>
          <span className="result-count">{filtered.length} results</span>
        </div>
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Knowledge</th>
                <th>Source</th>
                <th>Status</th>
                <th>Owner</th>
                <th>Confidence</th>
                <th>Updated</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((item) => (
                <tr key={item.id}>
                  <td>
                    <b>{item.title}</b>
                    <small>{item.id}</small>
                  </td>
                  <td>{item.source}</td>
                  <td>
                    <span
                      className={`status ${item.status === "Approved" ? "ok" : item.status === "Rejected" ? "bad" : "wait"}`}
                    >
                      {item.status}
                    </span>
                  </td>
                  <td>{item.owner}</td>
                  <td>
                    <b>{item.confidence}%</b>
                  </td>
                  <td>{item.updated}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {!filtered.length && (
            <div className="table-empty">
              No knowledge matches these filters.
            </div>
          )}
        </div>
      </section>
    </div>
  );
}

function Review({
  pending,
  reviewed,
  setReviewed,
  connection,
}: {
  pending: KnowledgeItem[];
  reviewed: string[];
  setReviewed: (items: string[]) => void;
  connection: ConnectionState;
}) {
  const [selected, setSelected] = useState(pending[0]?.id || "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState(false);
  const item =
    pending.find((candidate) => candidate.id === selected) || pending[0];
  const [draft, setDraft] = useState("");
  const decide = async (action: "approve" | "reject" | "edit") => {
    if (!item) return;
    setSaving(true);
    setError("");
    try {
      if (connection === "live") {
        const response = await fetch(
          `/api/loreline/v1/candidates/${item.id}/decision`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              action,
              content: action === "edit" ? draft : "",
              note: "Reviewed in Loreline console",
            }),
          },
        );
        const body = await response.json().catch(() => ({}));
        if (!response.ok)
          throw new Error(body.detail || "The decision could not be saved");
      }
      setReviewed([...reviewed, item.id]);
      setSelected("");
      setEditing(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Review failed");
    } finally {
      setSaving(false);
    }
  };
  if (!item)
    return (
      <div className="page empty">
        <span>✓</span>
        <h1>Queue cleared</h1>
        <p>Every candidate has a decision. Nice work.</p>
      </div>
    );
  return (
    <div className="page">
      <div className="page-heading compact">
        <div>
          <p className="kicker">HUMAN REVIEW</p>
          <h1>Make knowledge trustworthy</h1>
          <p>
            Approve, edit, or reject normalized candidates before they reach
            employees.
          </p>
        </div>
        <span className="queue-pill">{pending.length} waiting</span>
      </div>
      <div className="review-layout">
        <section className="panel candidate-list">
          <h3>Priority candidates</h3>
          {pending.map((candidate) => (
            <button
              className={item.id === candidate.id ? "selected" : ""}
              key={candidate.id}
              onClick={() => {
                setSelected(candidate.id);
                setEditing(false);
              }}
            >
              <span className="priority-dot" />
              <div>
                <b>{candidate.title}</b>
                <small>
                  {candidate.source} · {candidate.updated}
                </small>
              </div>
              <strong>{candidate.confidence}%</strong>
            </button>
          ))}
        </section>
        <section className="panel editor">
          <div className="editor-top">
            <span className="tag">{item.source}</span>
            <span>{item.id}</span>
            <span>
              {connection === "live" ? "Durable candidate" : "Sample candidate"}
            </span>
          </div>
          <h2>{item.title}</h2>
          {editing ? (
            <textarea
              className="candidate-editor"
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              aria-label="Edit candidate content"
            />
          ) : (
            <p className="answer-copy">{item.content}</p>
          )}
          <div className="evidence">
            <b>Source evidence</b>
            <p>{item.content}</p>
            <small>{item.source} · normalized source record</small>
          </div>
          <div className="editor-meta">
            <div>
              <small>Owner</small>
              <b>{item.owner}</b>
            </div>
            <div>
              <small>Model confidence</small>
              <b>{item.confidence}%</b>
            </div>
            <div>
              <small>Storage</small>
              <b>{connection === "live" ? "PostgreSQL" : "Sample data"}</b>
            </div>
          </div>
          {error && (
            <p className="form-error" role="alert">
              {error}
            </p>
          )}
          <div className="editor-actions">
            <button
              className="reject"
              disabled={saving}
              onClick={() => void decide("reject")}
            >
              Reject
            </button>
            <button
              className="outline"
              disabled={saving}
              onClick={() => {
                if (editing) {
                  void decide("edit");
                } else {
                  setDraft(item.content || "");
                  setEditing(true);
                }
              }}
            >
              {editing ? "Save & approve" : "Edit candidate"}
            </button>
            <button
              className="approve"
              disabled={saving}
              onClick={() => void decide("approve")}
            >
              {saving ? "Saving…" : "✓ Approve & index"}
            </button>
          </div>
        </section>
      </div>
    </div>
  );
}

function Assistant({
  query,
  setQuery,
  asked,
  setAsked,
  connection,
}: {
  query: string;
  setQuery: (value: string) => void;
  asked: boolean;
  setAsked: (value: boolean) => void;
  connection: ConnectionState;
}) {
  const [answer, setAnswer] = useState<LiveAnswer | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState("");
  const submit = async (value = query) => {
    const question = value.trim();
    if (!question) return;
    setQuery(question);
    setAsked(true);
    setLoading(true);
    setError("");
    setAnswer(null);
    setFeedback("");
    try {
      if (connection !== "live") {
        await new Promise((resolve) => setTimeout(resolve, 350));
        const candidate =
          demoKnowledge
            .filter((item) => item.status === "Approved")
            .find((item) =>
              question
                .toLowerCase()
                .split(/\s+/)
                .some(
                  (word) =>
                    word.length > 4 && item.title.toLowerCase().includes(word),
                ),
            ) || demoKnowledge[0];
        setAnswer({
          answer: candidate.content || "",
          confidence: candidate.confidence / 100,
          grounded: true,
          citations: [
            {
              id: candidate.id,
              title: candidate.title,
              source: candidate.source,
            },
          ],
          preview: true,
        });
        return;
      }
      const response = await fetch("/api/loreline/v1/assistant/ask", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question }),
      });
      const data = await response.json();
      if (!response.ok)
        throw new Error(data.detail || "The answer service is unavailable");
      setAnswer(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Answer failed");
    } finally {
      setLoading(false);
    }
  };
  const sendFeedback = async (rating: number) => {
    if (answer?.preview) {
      setFeedback("Feedback is disabled for sample answers.");
      return;
    }
    if (!answer?.message_id) return;
    try {
      const response = await fetch("/api/loreline/v1/feedback", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message_id: answer.message_id, rating }),
      });
      if (!response.ok) throw new Error();
      setFeedback("Thanks—your feedback was recorded.");
    } catch {
      setFeedback("Feedback could not be saved.");
    }
  };
  return (
    <div className="assistant-page">
      <div className="assistant-head">
        <Mark />
        <p className="kicker">TRUSTED ANSWERS, INSTANTLY</p>
        <h1>What would you like to know?</h1>
        <p>Ask across every approved policy, ticket, and wiki page.</p>
      </div>
      {connection !== "live" && (
        <div className="demo-chip">
          Interactive preview · answers use sample knowledge
        </div>
      )}
      <div className="ask-box">
        <textarea
          aria-label="Ask a knowledge question"
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setAsked(false);
            setAnswer(null);
            setError("");
          }}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault();
              void submit();
            }
          }}
          placeholder="Ask about policies, processes, or support issues…"
        />
        <button
          aria-label="Ask question"
          disabled={loading || !query.trim()}
          onClick={() => void submit()}
        >
          {loading ? "…" : "↑"}
        </button>
        <div>
          <span>Enter to ask · Shift+Enter for a new line</span>
          <span>Approved knowledge only</span>
        </div>
      </div>
      {!asked ? (
        <div className="suggestions">
          <span>Try asking</span>
          {[
            "How do I restore SSO access?",
            "What is our parental leave policy?",
            "Can I use my card overseas?",
          ].map((value) => (
            <button key={value} onClick={() => void submit(value)}>
              {value}
              <i>→</i>
            </button>
          ))}
        </div>
      ) : error ? (
        <div className="answer-card error-card">
          <div className="answer-label">
            <span>!</span>
            <b>Answer unavailable</b>
          </div>
          <h2>The request could not be completed.</h2>
          <p>{error}</p>
          <button className="outline" onClick={() => void submit()}>
            Try again
          </button>
        </div>
      ) : loading ? (
        <div className="answer-card loading-card" aria-live="polite">
          <div className="answer-label">
            <span>✦</span>
            <b>Retrieving approved knowledge…</b>
          </div>
        </div>
      ) : (
        answer && (
          <div className="answer-card" aria-live="polite">
            <div className="answer-label">
              <span>✦</span>
              <b>{answer.preview ? "Preview answer" : "Loreline answer"}</b>
              <small>
                {answer.citations.length} approved source
                {answer.citations.length === 1 ? "" : "s"} ·{" "}
                {Math.round(answer.confidence * 100)}% confidence
              </small>
            </div>
            <h2>{query}</h2>
            <p>{answer.answer}</p>
            <div className="citations">
              <b>Sources</b>
              {answer.citations.map((citation, index) => (
                <div className="citation" key={citation.id}>
                  <span>{index + 1}</span>
                  {citation.title}
                  <small>{citation.source}</small>
                </div>
              ))}
            </div>
            <div className="feedback">
              <span>Was this useful?</span>
              <button onClick={() => void sendFeedback(1)}>Yes</button>
              <button onClick={() => void sendFeedback(-1)}>
                Needs correction
              </button>
              <small>
                {answer.grounded
                  ? "Grounded in approved knowledge"
                  : "No supporting evidence found"}
              </small>
            </div>
            {feedback && (
              <p className="feedback-note" role="status">
                {feedback}
              </p>
            )}
          </div>
        )
      )}
    </div>
  );
}
