"use client";

import { useEffect, useMemo, useState } from "react";
import { Operations, Sources, type SourceItem } from "./Operations";

type View = "overview" | "library" | "review" | "assistant" | "sources" | "operations";

type KnowledgeItem = { id:string; title:string; source:string; status:string; owner:string; updated:string; confidence:number; content?:string };
type LiveAnswer = { answer:string; confidence:number; grounded:boolean; citations:{id:string;title:string;source:string}[]; message_id?:string };

const demoKnowledge: KnowledgeItem[] = [
  { id: "KB-2048", title: "Restoring access after an SSO lockout", source: "Resolved ticket", status: "Approved", owner: "IT Operations", updated: "8 min ago", confidence: 98 },
  { id: "KB-2047", title: "2026 parental leave policy", source: "Policy document", status: "Approved", owner: "People Ops", updated: "22 min ago", confidence: 96 },
  { id: "KB-2046", title: "Corporate card: international travel", source: "Wiki page", status: "Needs review", owner: "Finance", updated: "34 min ago", confidence: 87 },
  { id: "KB-2045", title: "Requesting production database access", source: "Resolved ticket", status: "Needs review", owner: "Engineering", updated: "1 hr ago", confidence: 91 },
  { id: "KB-2044", title: "Home office equipment allowance", source: "Policy document", status: "Approved", owner: "People Ops", updated: "2 hrs ago", confidence: 99 },
];

const activity = [
  ["12:42", "Answer delivered", "SSO lockout · 3 sources cited"],
  ["12:37", "Knowledge approved", "Parental leave policy · Morgan L."],
  ["12:31", "Source synchronized", "Support workspace · 84 records"],
  ["12:18", "Correction applied", "Travel card policy · version 7"],
];

function Mark() { return <span className="mark" aria-hidden="true"><i /><i /><i /></span>; }
function Icon({ children }: { children: React.ReactNode }) { return <span className="nav-icon" aria-hidden="true">{children}</span>; }

export default function Home() {
  const [view, setView] = useState<View>("overview");
  const [reviewed, setReviewed] = useState<string[]>([]);
  const [query, setQuery] = useState("");
  const [asked, setAsked] = useState(false);
  const [knowledge,setKnowledge]=useState<KnowledgeItem[]>(demoKnowledge);
  const [connection,setConnection]=useState<"checking"|"live"|"offline">("checking");
  const [dashboard,setDashboard]=useState<Record<string,number>>({});
  const [sources,setSources]=useState<SourceItem[]>([{id:"demo-support",kind:"ticketing",name:"Support workspace",status:"active",last_synced_at:new Date().toISOString()},{id:"demo-policy",kind:"drive",name:"Policy drive",status:"active",last_synced_at:new Date().toISOString()},{id:"demo-wiki",kind:"wiki",name:"Company wiki",status:"active",last_synced_at:new Date().toISOString()}]);
  useEffect(()=>{let active=true;(async()=>{try{const [d,c,s]=await Promise.all([fetch("/api/loreline/v1/dashboard"),fetch("/api/loreline/v1/candidates?limit=100"),fetch("/api/loreline/v1/sources")]);if(!d.ok||!c.ok||!s.ok)throw new Error("api unavailable");const stats=await d.json();const candidates=await c.json();const sourceData=await s.json();if(!active)return;setDashboard(stats);const live:KnowledgeItem[]=(candidates.items||[]).map((item:{id:string;title:string;source:string;status:string;confidence:number;content:string;created_at:string})=>({id:item.id,title:item.title,source:item.source,status:item.status==="pending"?"Needs review":"Approved",owner:"Knowledge Ops",updated:new Date(item.created_at).toLocaleString(),confidence:Math.round(item.confidence*100),content:item.content}));if(live.length)setKnowledge(live);setSources(sourceData.items||[]);setConnection("live")}catch{if(active)setConnection("offline")}})();return()=>{active=false}},[]);
  const pending = useMemo(() => knowledge.filter(k => k.status === "Needs review" && !reviewed.includes(k.id)), [knowledge,reviewed]);

  const nav: [View, string, string][] = [
    ["overview", "Overview", "⌂"], ["library", "Knowledge", "◇"], ["review", "Review queue", "✓"], ["assistant", "Ask Loreline", "✦"],
  ];

  return <main className="shell">
    <aside className="sidebar">
      <div className="brand"><Mark /><span>Loreline</span></div>
      <div className="workspace"><span className="workspace-logo">A</span><div><b>Acme Labs</b><small>Operations workspace</small></div><span className="chevron">⌄</span></div>
      <nav>
        <p className="eyebrow">Workspace</p>
        {nav.map(([id, label, icon]) => <button key={id} className={view === id ? "active" : ""} onClick={() => setView(id)}><Icon>{icon}</Icon>{label}{id === "review" && pending.length > 0 && <span className="count">{pending.length}</span>}</button>)}
        <p className="eyebrow manage">Manage</p>
        <button className={view==="sources"?"active":""} onClick={()=>setView("sources")}><Icon>⌁</Icon>Sources</button><button className={view==="operations"?"active":""} onClick={()=>setView("operations")}><Icon>◎</Icon>Operations</button><button><Icon>⚙</Icon>Settings</button>
      </nav>
      <div className="sidebar-foot"><div className={`health-dot ${connection}`} /><div><b>{connection==="live"?"Production API connected":connection==="checking"?"Checking services":"Demo data · API offline"}</b><small>{connection==="live"?"Durable data and workers active":"Configure the gateway connection"}</small></div></div>
      <div className="profile"><span>ML</span><div><b>Morgan Lee</b><small>Administrator</small></div><button aria-label="More profile options">•••</button></div>
    </aside>

    <section className="content">
      <header className="topbar"><div className="breadcrumbs">Acme Labs <span>/</span> {nav.find(n => n[0] === view)?.[1]??(view==="sources"?"Sources":"Operations")}</div><div className="top-actions"><button className="search" onClick={() => setView("assistant")}>⌕ <span>Search anything</span><kbd>⌘ K</kbd></button><button className="round" aria-label="Notifications">♢<i /></button><button className="primary" onClick={() => setView("assistant")}>✦ Ask Loreline</button></div></header>

      {view === "overview" && <Overview setView={setView} knowledge={knowledge} dashboard={dashboard} />}
      {view === "library" && <Library knowledge={knowledge} />}
      {view === "review" && <Review pending={pending} reviewed={reviewed} setReviewed={setReviewed} connection={connection} />}
      {view === "assistant" && <Assistant query={query} setQuery={setQuery} asked={asked} setAsked={setAsked} />}
      {view === "sources" && <Sources items={sources} connection={connection} onCreated={item=>setSources([item,...sources])} />}
      {view === "operations" && <Operations connection={connection} />}
    </section>
  </main>;
}

function Overview({ setView,knowledge,dashboard }: { setView: (v: View) => void; knowledge:KnowledgeItem[]; dashboard:Record<string,number> }) {
  return <div className="page">
    <div className="page-heading"><div><p className="kicker">SATURDAY, AUGUST 1</p><h1>Good afternoon, Morgan.</h1><p>Your knowledge is healthy and your team is moving fast.</p></div><button className="outline" onClick={() => setView("review")}>Open review queue <span>→</span></button></div>
    <div className="metric-grid">
      <Metric label="Approved knowledge" value={(dashboard.approved_knowledge??2048).toLocaleString()} delta="Reviewed and indexed" tone="violet" spark={[20,28,25,39,42,55,58,69,76,82]} />
      <Metric label="Questions resolved" value="1,284" delta="+18.4% this month" tone="green" spark={[26,34,31,46,43,59,55,70,68,87]} />
      <Metric label="Answer quality" value={`${(dashboard.answer_quality??96.8).toFixed(1)}%`} delta="From employee feedback" tone="blue" spark={[44,46,51,50,57,61,66,69,72,78]} />
      <Metric label="Needs review" value={(dashboard.needs_review??12).toLocaleString()} delta={`${dashboard.active_jobs??4} active jobs`} tone="amber" spark={[72,65,66,55,59,46,41,35,30,24]} />
    </div>
    <div className="main-grid">
      <section className="panel pipeline"><PanelTitle title="Knowledge pipeline" subtitle="Content moving from source to trusted answer" action="View all" />
        <div className="pipeline-flow"><Stage icon="↧" label="Ingested" value="2,184" sub="3 connected sources" tone="slate" /><Arrow /><Stage icon="≋" label="Normalized" value="2,076" sub="108 duplicates removed" tone="blue" /><Arrow /><Stage icon="✓" label="Approved" value="2,048" sub="12 await review" tone="violet" /><Arrow /><Stage icon="⌕" label="Indexed" value="2,036" sub="Ready for answers" tone="green" /></div>
        <div className="freshness"><div><span>Knowledge freshness</span><b>94%</b></div><div className="bar"><i /></div><p>94% of approved knowledge has been verified in the last 90 days.</p></div>
      </section>
      <section className="panel coverage"><PanelTitle title="Source coverage" subtitle="Health across connected systems" action="Manage" />
        {[ ["S", "Support workspace", "1,462 records", "Synced 6 min ago", 98, "violet"], ["D", "Policy drive", "418 documents", "Synced 18 min ago", 91, "blue"], ["W", "Company wiki", "304 pages", "Synced 34 min ago", 87, "green"] ].map(x => <div className="source" key={x[1]}><span className={`source-icon ${x[5]}`}>{x[0]}</span><div className="source-info"><b>{x[1]}</b><small>{x[2]}</small></div><div className="source-health"><span>{x[3]}</span><div><i style={{width: `${x[4]}%`}} /></div></div></div>)}
      </section>
    </div>
    <div className="bottom-grid">
      <section className="panel review-preview"><PanelTitle title="Review queue" subtitle="Candidates that need a human decision" action="Review all" /><table><thead><tr><th>Knowledge candidate</th><th>Source</th><th>Confidence</th><th>Age</th></tr></thead><tbody>{knowledge.filter(k => k.status === "Needs review").map(k => <tr key={k.id} onClick={() => setView("review")}><td><b>{k.title}</b><small>{k.owner} · {k.id}</small></td><td><span className="tag">{k.source}</span></td><td><span className="confidence"><i style={{width: `${k.confidence}%`}} /></span>{k.confidence}%</td><td>{k.updated}</td></tr>)}</tbody></table></section>
      <section className="panel activity"><PanelTitle title="Live activity" subtitle="What Loreline is doing now" action="Audit log" />{activity.map(a => <div className="activity-row" key={a[0]}><time>{a[0]}</time><span className="activity-dot" /><div><b>{a[1]}</b><small>{a[2]}</small></div></div>)}</section>
    </div>
  </div>;
}

function Metric({ label, value, delta, tone, spark }: { label:string; value:string; delta:string; tone:string; spark:number[] }) { return <article className="metric"><div><span>{label}</span><b>{value}</b><small className={tone}>{delta}</small></div><div className={`spark ${tone}`}>{spark.map((h,i) => <i key={i} style={{height:`${h}%`}} />)}</div></article>; }
function PanelTitle({title,subtitle,action}:{title:string;subtitle:string;action:string}) { return <div className="panel-title"><div><h2>{title}</h2><p>{subtitle}</p></div><button>{action} <span>→</span></button></div>; }
function Stage({icon,label,value,sub,tone}:{icon:string;label:string;value:string;sub:string;tone:string}) { return <div className="stage"><span className={`stage-icon ${tone}`}>{icon}</span><div><small>{label}</small><b>{value}</b><p>{sub}</p></div></div>; }
function Arrow(){return <span className="arrow">→</span>}

function Library({knowledge}:{knowledge:KnowledgeItem[]}){ return <div className="page"><div className="page-heading compact"><div><p className="kicker">KNOWLEDGE LIBRARY</p><h1>One source of truth.</h1><p>Approved, traceable knowledge ready for every employee question.</p></div><button className="primary">+ Add source</button></div><section className="panel library"><div className="library-tools"><div className="filter">⌕ Search titles, owners, or IDs</div><button>All status ⌄</button><button>All sources ⌄</button></div><table><thead><tr><th>Knowledge</th><th>Source</th><th>Status</th><th>Owner</th><th>Confidence</th><th>Updated</th></tr></thead><tbody>{knowledge.map(k => <tr key={k.id}><td><b>{k.title}</b><small>{k.id}</small></td><td>{k.source}</td><td><span className={`status ${k.status === "Approved" ? "ok" : "wait"}`}>{k.status}</span></td><td>{k.owner}</td><td><b>{k.confidence}%</b></td><td>{k.updated}</td></tr>)}</tbody></table></section></div> }

function Review({pending,reviewed,setReviewed,connection}:{pending:KnowledgeItem[];reviewed:string[];setReviewed:(x:string[])=>void;connection:"checking"|"live"|"offline"}){
 const [selected,setSelected]=useState(pending[0]?.id||"");const [saving,setSaving]=useState(false);const [error,setError]=useState("");const item=pending.find(x=>x.id===selected)||pending[0];
 const decide=async(action:"approve"|"reject")=>{if(!item)return;setSaving(true);setError("");try{if(connection==="live"){const response=await fetch(`/api/loreline/v1/candidates/${item.id}/decision`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({action,note:"Reviewed in Loreline console"})});if(!response.ok)throw new Error("The decision could not be saved")};setReviewed([...reviewed,item.id]);setSelected("")}catch(e){setError(e instanceof Error?e.message:"Review failed")}finally{setSaving(false)}};
 if(!item)return <div className="page empty"><span>✓</span><h1>Queue cleared.</h1><p>Every candidate has a decision. Nice work.</p></div>;
 return <div className="page"><div className="page-heading compact"><div><p className="kicker">HUMAN REVIEW</p><h1>Make knowledge trustworthy.</h1><p>Approve, edit, or reject normalized candidates before they reach employees.</p></div><span className="queue-pill">{pending.length} waiting</span></div><div className="review-layout"><section className="panel candidate-list"><h3>Priority candidates</h3>{pending.map(k=><button className={item.id===k.id?"selected":""} key={k.id} onClick={()=>setSelected(k.id)}><span className="priority-dot"/><div><b>{k.title}</b><small>{k.source} · {k.updated}</small></div><strong>{k.confidence}%</strong></button>)}</section><section className="panel editor"><div className="editor-top"><span className="tag">{item.source}</span><span>{item.id}</span><span>{connection==="live"?"Durable candidate":"Demo candidate"}</span></div><h2>{item.title}</h2><p className="answer-copy">{item.content||"Employees traveling internationally may use the corporate card for approved business expenses. Before travel, notify Finance through the expense portal and confirm the card has international transactions enabled."}</p><div className="evidence"><b>Source evidence</b><p>{item.content||"Cardholders must notify Finance before international business travel."}</p><small>{item.source} · normalized source record</small></div><div className="editor-meta"><div><small>Owner</small><b>{item.owner}</b></div><div><small>Model confidence</small><b>{item.confidence}%</b></div><div><small>Storage</small><b>{connection==="live"?"PostgreSQL":"Demo"}</b></div></div>{error&&<p className="form-error">{error}</p>}<div className="editor-actions"><button className="reject" disabled={saving} onClick={()=>decide("reject")}>Reject</button><button className="outline">Edit candidate</button><button className="approve" disabled={saving} onClick={()=>decide("approve")}>{saving?"Saving…":"✓ Approve & index"}</button></div></section></div></div>
}

function Assistant({query,setQuery,asked,setAsked}:{query:string;setQuery:(x:string)=>void;asked:boolean;setAsked:(x:boolean)=>void}){
 const [answer,setAnswer]=useState<LiveAnswer|null>(null);const [loading,setLoading]=useState(false);const [error,setError]=useState("");
 const submit=async(value=query)=>{const question=value.trim();if(!question)return;setQuery(question);setAsked(true);setLoading(true);setError("");setAnswer(null);try{const response=await fetch("/api/loreline/v1/assistant/ask",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({question})});const data=await response.json();if(!response.ok)throw new Error(data.detail||"The answer service is unavailable");setAnswer(data)}catch(e){setError(e instanceof Error?e.message:"Answer failed")}finally{setLoading(false)}};
 const feedback=async(rating:number)=>{if(!answer?.message_id)return;await fetch("/api/loreline/v1/feedback",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({message_id:answer.message_id,rating})})};
 return <div className="assistant-page"><div className="assistant-head"><Mark/><p className="kicker">TRUSTED ANSWERS, INSTANTLY</p><h1>What would you like to know?</h1><p>Ask across every approved policy, ticket, and wiki page.</p></div><div className="ask-box"><textarea aria-label="Ask a knowledge question" value={query} onChange={e=>{setQuery(e.target.value);setAsked(false);setAnswer(null);setError("")}} onKeyDown={e=>{if(e.key==="Enter"&&!e.shiftKey){e.preventDefault();submit()}}} placeholder="Ask about policies, processes, or support issues…"/><button disabled={loading} onClick={()=>submit()}>{loading?"…":"↑"}</button><div><span>⌘ Enter to ask</span><span>Answers use approved knowledge only</span></div></div>{!asked?<div className="suggestions"><span>Try asking</span>{["How do I restore SSO access?","What is our parental leave policy?","Can I use my card overseas?"].map(x=><button key={x} onClick={()=>submit(x)}>{x}<i>→</i></button>)}</div>:error?<div className="answer-card error-card"><div className="answer-label"><span>!</span><b>Operational service unavailable</b></div><h2>The request could not be completed.</h2><p>{error}</p></div>:loading?<div className="answer-card loading-card"><div className="answer-label"><span>✦</span><b>Retrieving approved knowledge…</b></div></div>:answer&&<div className="answer-card"><div className="answer-label"><span>✦</span><b>Loreline answer</b><small>{answer.citations.length} approved source{answer.citations.length===1?"":"s"} · {Math.round(answer.confidence*100)}% confidence</small></div><h2>{query}</h2><p>{answer.answer}</p><div className="citations"><b>Sources</b>{answer.citations.map((c,i)=><button key={c.id}><span>{i+1}</span>{c.title}<small>{c.source}</small></button>)}</div><div className="feedback"><span>Was this useful?</span><button onClick={()=>feedback(1)}>Yes</button><button onClick={()=>feedback(-1)}>Needs correction</button><small>{answer.grounded?"Grounded in approved knowledge":"No supporting evidence found"}</small></div></div>}</div>
}
