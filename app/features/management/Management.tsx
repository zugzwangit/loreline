"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { jsonRequest } from "../../lib/api";
import type { ConnectionState } from "../../lib/domain";

type AuditEvent = {
  id: number;
  actor_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  request_id: string;
  created_at: string;
};
type APIKey = {
  id: string;
  name: string;
  key_prefix: string;
  role: string;
  expires_at?: string | null;
  last_used_at?: string | null;
  created_at: string;
};

export function AuditLog({ connection }: { connection: ConnectionState }) {
  const [items, setItems] = useState<AuditEvent[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const load = useCallback(async () => {
    if (connection !== "live") return;
    setLoading(true);
    setError("");
    try {
      const result = await jsonRequest<{ items: AuditEvent[] }>(
        "/api/loreline/v1/audit?limit=200",
      );
      setItems(result.items || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Audit log unavailable");
    } finally {
      setLoading(false);
    }
  }, [connection]);
  useEffect(() => {
    if (connection !== "live") return;
    let active = true;
    fetch("/api/loreline/v1/audit?limit=200")
      .then(async (response) => {
        const body = await response.json();
        if (!response.ok)
          throw new Error(body.detail || "Audit log unavailable");
        if (active) setItems(body.items || []);
      })
      .catch((error) => {
        if (active)
          setError(
            error instanceof Error ? error.message : "Audit log unavailable",
          );
      });
    return () => {
      active = false;
    };
  }, [connection]);
  return (
    <div className="page">
      <div className="page-heading compact">
        <div>
          <p className="kicker">SECURITY & GOVERNANCE</p>
          <h1>Audit log</h1>
          <p>
            Trace privileged decisions, source changes, key rotation, and
            recovery actions.
          </p>
        </div>
        <button
          className="outline"
          disabled={connection !== "live" || loading}
          onClick={() => void load()}
        >
          {loading ? "Refreshing…" : "Refresh"}
        </button>
      </div>
      {connection !== "live" && (
        <div className="connection-banner">
          Audit events are available when the production gateway is connected.
        </div>
      )}
      {error && (
        <p className="form-error" role="alert">
          {error}
        </p>
      )}
      {connection === "live" && (
        <section className="panel table-scroll">
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Action</th>
                <th>Resource</th>
                <th>Actor</th>
                <th>Request</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.id}>
                  <td>{new Date(item.created_at).toLocaleString()}</td>
                  <td>
                    <b>{item.action}</b>
                  </td>
                  <td>
                    {item.resource_type} · {item.resource_id.slice(0, 12)}
                  </td>
                  <td>
                    {item.actor_id ? item.actor_id.slice(0, 12) : "system"}
                  </td>
                  <td>
                    <code>{item.request_id.slice(0, 12)}</code>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!loading && !items.length && (
            <div className="table-empty">No audit events yet.</div>
          )}
        </section>
      )}
    </div>
  );
}

export function Settings({ connection }: { connection: ConnectionState }) {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [created, setCreated] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const load = useCallback(async () => {
    if (connection !== "live") return;
    try {
      const result = await jsonRequest<{ items: APIKey[] }>(
        "/api/loreline/v1/api-keys",
      );
      setKeys(result.items || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "API keys unavailable");
    }
  }, [connection]);
  useEffect(() => {
    if (connection !== "live") return;
    let active = true;
    fetch("/api/loreline/v1/api-keys")
      .then(async (response) => {
        const body = await response.json();
        if (!response.ok)
          throw new Error(body.detail || "API keys unavailable");
        if (active) setKeys(body.items || []);
      })
      .catch((error) => {
        if (active)
          setError(
            error instanceof Error ? error.message : "API keys unavailable",
          );
      });
    return () => {
      active = false;
    };
  }, [connection]);
  const create = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSaving(true);
    setError("");
    setCreated("");
    const data = new FormData(event.currentTarget);
    try {
      const result = await jsonRequest<{ api_key: string }>(
        "/api/loreline/v1/api-keys",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            name: String(data.get("name")),
            role: String(data.get("role")),
          }),
        },
      );
      setCreated(result.api_key);
      event.currentTarget.reset();
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Key creation failed");
    } finally {
      setSaving(false);
    }
  };
  const revoke = async (id: string) => {
    if (
      !window.confirm(
        "Revoke this API key? Existing clients using it will stop immediately.",
      )
    )
      return;
    setError("");
    try {
      await jsonRequest<null>(`/api/loreline/v1/api-keys/${id}`, {
        method: "DELETE",
      });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Key revocation failed");
    }
  };
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <p className="kicker">WORKSPACE SETTINGS</p>
          <h1>Access and deployment</h1>
          <p>
            Rotate service credentials and verify the console’s operational
            connection.
          </p>
        </div>
        <span className={`connection-chip ${connection}`}>
          {connection === "live"
            ? "Connected"
            : connection === "checking"
              ? "Checking"
              : "Demo mode"}
        </span>
      </div>
      <div className="settings-grid">
        <section className="panel settings-card">
          <h2>Console connection</h2>
          <p>
            The browser never receives a service key. This console uses a
            signed-in server proxy to reach the tenant-scoped gateway.
          </p>
          <dl>
            <div>
              <dt>Gateway</dt>
              <dd>
                {connection === "live"
                  ? "Reachable and authenticated"
                  : "Not configured for this deployment"}
              </dd>
            </div>
            <div>
              <dt>Writes</dt>
              <dd>
                {connection === "live"
                  ? "Durable and audited"
                  : "Disabled in demo mode"}
              </dd>
            </div>
          </dl>
        </section>
        <section className="panel settings-card">
          <h2>Create API key</h2>
          <p>
            Secrets are shown once. Store new keys in your secret manager before
            leaving this page.
          </p>
          <form className="key-form" onSubmit={create}>
            <label>
              Key name
              <input
                name="name"
                required
                maxLength={160}
                placeholder="Production console"
              />
            </label>
            <label>
              Role
              <select name="role" defaultValue="viewer">
                <option>viewer</option>
                <option>reviewer</option>
                <option>admin</option>
                <option>owner</option>
              </select>
            </label>
            <button
              className="primary"
              disabled={connection !== "live" || saving}
            >
              {saving ? "Creating…" : "Create key"}
            </button>
          </form>
        </section>
      </div>
      {created && (
        <section className="secret-reveal" role="status">
          <div>
            <b>Copy this key now</b>
            <p>It cannot be retrieved again.</p>
          </div>
          <code>{created}</code>
          <button
            className="outline"
            onClick={() => navigator.clipboard.writeText(created)}
          >
            Copy
          </button>
        </section>
      )}
      {error && (
        <p className="form-error" role="alert">
          {error}
        </p>
      )}
      <section className="panel table-scroll">
        <div className="panel-title">
          <div>
            <h2>Active API keys</h2>
            <p>Owner access is required to manage credentials.</p>
          </div>
        </div>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Prefix</th>
              <th>Role</th>
              <th>Last used</th>
              <th>Created</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {keys.map((key) => (
              <tr key={key.id}>
                <td>
                  <b>{key.name}</b>
                </td>
                <td>
                  <code>{key.key_prefix}…</code>
                </td>
                <td>{key.role}</td>
                <td>
                  {key.last_used_at
                    ? new Date(key.last_used_at).toLocaleString()
                    : "Never"}
                </td>
                <td>{new Date(key.created_at).toLocaleDateString()}</td>
                <td>
                  <button
                    className="danger-link"
                    onClick={() => void revoke(key.id)}
                  >
                    Revoke
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!keys.length && (
          <div className="table-empty">
            {connection === "live"
              ? "No API keys were returned."
              : "Connect the production gateway to manage keys."}
          </div>
        )}
      </section>
    </div>
  );
}
