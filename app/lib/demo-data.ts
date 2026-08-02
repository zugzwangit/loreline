import type { KnowledgeItem, SourceItem } from "./domain";

export const demoKnowledge: KnowledgeItem[] = [
  {
    id: "KB-2048",
    title: "Restoring access after an SSO lockout",
    source: "Resolved ticket",
    status: "Approved",
    owner: "IT Operations",
    updated: "8 min ago",
    confidence: 98,
    content:
      "Wait fifteen minutes, then retry the identity portal. If the account remains locked, contact IT Operations.",
  },
  {
    id: "KB-2047",
    title: "2026 parental leave policy",
    source: "Policy document",
    status: "Approved",
    owner: "People Ops",
    updated: "22 min ago",
    confidence: 96,
    content:
      "Eligible employees receive sixteen weeks of paid parental leave and may begin leave up to two weeks before the expected arrival date.",
  },
  {
    id: "KB-2046",
    title: "Corporate card: international travel",
    source: "Wiki page",
    status: "Needs review",
    owner: "Finance",
    updated: "34 min ago",
    confidence: 87,
    content:
      "Employees traveling internationally may use the corporate card for approved business expenses after notifying Finance.",
  },
  {
    id: "KB-2045",
    title: "Requesting production database access",
    source: "Resolved ticket",
    status: "Needs review",
    owner: "Engineering",
    updated: "1 hr ago",
    confidence: 91,
    content:
      "Production access requires manager approval, a time-bound access request, and an incident or change reference.",
  },
  {
    id: "KB-2044",
    title: "Home office equipment allowance",
    source: "Policy document",
    status: "Approved",
    owner: "People Ops",
    updated: "2 hrs ago",
    confidence: 99,
    content:
      "Employees may expense up to $750 in approved home office equipment every two years.",
  },
];

export const demoSources: SourceItem[] = [
  {
    id: "demo-support",
    kind: "ticketing",
    name: "Support workspace",
    status: "active",
    last_synced_at: new Date().toISOString(),
  },
  {
    id: "demo-policy",
    kind: "drive",
    name: "Policy drive",
    status: "active",
    last_synced_at: new Date().toISOString(),
  },
  {
    id: "demo-wiki",
    kind: "wiki",
    name: "Company wiki",
    status: "active",
    last_synced_at: new Date().toISOString(),
  },
];
