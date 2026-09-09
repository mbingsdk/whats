# Product vision

WABA Control is the company's operational home for WhatsApp communication. An agent must know which business number they represent, who owns the conversation, whether a reply is allowed and what sending may cost. A supervisor must be able to stop a campaign, explain an exclusion and trace an incident without accessing Meta credentials.

The user confirmed an **existing Meta Cloud API WABA**. Connect existing assets first. One PT with one company Organization is the normal initial deployment. Organization-first protects data ownership and future expansion; it does not introduce public registration, SaaS subscriptions, a tenant marketplace or a generic onboarding wizard. Show organization switching only to members of more than one active organization. Users can belong to multiple organizations and teams; a WABA has one internal organization owner at a time.

The product combines shared inbox, focused contact records, campaigns, Meta asset management and an internal developer gateway. Operational control is the priority: one send gate, explainable consent and cost decisions, recoverable ingestion and visible uncertainty.

## Users and scope

| Persona | Primary work | Constraint |
| --- | --- | --- |
| Service agent | Triage, reply, notes, resolve | Authorized teams and numbers only |
| Supervisor | Assign, handoff, SLA, review | Approval authority comes from policy |
| Campaign operator | Audience, template, schedule, progress | Approved content and recipient snapshot are immutable |
| Administrator | Members, assets, credentials, budgets | Sensitive changes require reauthentication and audit |
| Developer | Scoped gateway and diagnosis | No distribution of raw Meta tokens |
| Auditor/viewer | Evidence and reports | Read access does not imply PII export permission |

First delivery: membership/RBAC, existing WABA connection, reliable ingestion, shared inbox, consent, pricing controls and controlled template sends. Campaigns follow after the send gate is proven. Flows, bounded automation and gateway consumers reuse that foundation.

Non-goals: general CRM, omnichannel helpdesk, BSP resale, arbitrary code execution, an AI chatbot product, or replacing every Meta Business Manager action. Calling, payment initiation, coexistence, dynamic Flow endpoints, groups and commercial multi-company onboarding are DEFERRED pending demand and evidence.

## Outcomes and capacity assumptions

Measure first-response time, unresolved workload, failed/uncertain sends, recipient exclusions, estimated versus reconciled spend and recovery time. Establish baselines from actual pilot activity; no fabricated dashboard data or improvement percentages.

Provisional test workload: 50 members, 25 simultaneous agents, 5 WABAs, 20 numbers, 300k inbound/outbound messages per month, bursts of 50 webhook requests/second for one minute, and a 100k-recipient campaign snapshot. These are engineering assumptions, not company requirements or Meta limits. Completion time depends on measured capacity and account eligibility.

Unknown cost prevents dispatch by default. Uncertain external sends are quarantined. Staff departure removes access while retaining attributable history. These constraints take precedence over maximizing outbound throughput.
