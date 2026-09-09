# UI and interaction specification

Design for frequent work on dense operational information. The primary surface is the inbox, not a dashboard hero. Campaign views lead with audience, eligibility, authority and cost; Meta Health leads with diagnosis; developer screens lead with request/event evidence.

## Pattern study and rationale

Documentation-based study, not an authenticated usability audit: [Intercom's inbox](https://www.intercom.com/help/en/articles/6258745-the-inbox-explained) suggests complementary thread and table views plus discoverable shortcuts; [Zendesk's context panel](https://support.zendesk.com/hc/en-us/articles/4408828503450-Configuring-the-context-panel-in-the-Zendesk-Agent-Workspace) supports keeping customer context adjacent; [Stripe webhook tools](https://docs.stripe.com/webhooks) show the value of separating event from delivery attempt; [Mailchimp's checklist](https://mailchimp.com/help/create-and-send-regular-email/?v=114) makes send completeness explicit. Adapt these task patterns; do not copy product branding, layouts wholesale or AI features.

Our choices: stable navigation and dense rows for scan speed; split inbox to reduce navigation; contextual inspector to preserve reply focus; explicit preflight table for campaign review; event timeline with separate attempts for diagnosis. Each supports an operation, not decoration.

## Visual language

Warm off-white canvas `#F5F4EF`, white work surface, charcoal ink `#202823`, muted stone separators `#D7DCD5`, dark evergreen action accent `#17634D`. Amber `#8A5400` for expiring/warning, red `#B42318` for blocked/failure and restrained blue `#235C8A` for informational/reference links. Colors are design candidates; verify contrast with actual foreground/background pairs before implementation. Status always has text/icon, never color alone.

Use a system sans stack initially, tabular numerals for costs/counters/timers, monospace only for technical IDs/code. Body 14–15 px, metadata 12–13 px, primary page title 22–24 px. Default row height 40 px with 48 px comfortable mode; controls 32–36 px desktop, touch targets at least 44 px on touch layouts. Radius 4–6 px, 1 px separators, shadows limited to overlays. No gradients, glass panels, emoji icons or oversized metric cards.

Spacing is a 4 px scale; operational areas remain compact with clear grouping and readable line length. Motion only for focus/state change, roughly 120 ms, respecting reduced motion. No entrance animation or transition delays on inbox navigation. Semantic tokens allow a later tested dark theme without forcing launch scope.

## Navigation and screen specifications

Company identity and phone/team context remain visible. Show an organization switcher only when the user has multiple active organization memberships; the normal one-PT deployment opens its company workspace directly, without an onboarding wizard. Navigation: Inbox, Contacts, Campaigns, Content (Templates/Flows), Automations, Health, Reports, Developer, Settings. Permission filters remove inaccessible routes but direct URL still checks server access. Notification tray is targeted and navigates to the relevant record, not a generic activity feed. Settings includes a permission-scoped Rate Card Registry: import/manual entry, validation errors, change diff, review and explicit publish; show source dates, reviewer, coverage and next review. Published rows are read-only; correction starts a new import. Identity invitation status shows sanitized mail failure/resend controls without exposing tokens.

| Screen | Structure and primary actions | Required states |
| --- | --- | --- |
| Inbox | Slim navigation, 300–340 px conversation list, flexible thread, collapsible 280–320 px contact inspector | Empty team/unassigned, loading, restricted, stale/disconnected, unknown media, pending/uncertain send |
| Inbox list | Contact/name, phone alias, assignee/team, latest safe snippet, unread, priority, window/SLA badge | Distinguish service-window timer from SLA deadline; filter count respects access |
| Thread header | Identity/sender, assignment, handoff, state; takeover/resolve/snooze | Assignment conflict, deactivated assignee, bot already in flight |
| Composer | Explicit Reply/Note modes, attachment/type tools, server window and price state beside Send | Reply/Note never confused; expiry blocks free-form; drafts survive correctable errors |
| Contact | Focused identifiers/fields/tags plus timeline and consent ledger | Missing/opaque identity, duplicate conflict, suppressed, erased; no fake inferred country |
| Campaign list | Dense rows: state, sender, audience size, eligible/excluded, estimated/exposed cost, schedule, sponsor | Scheduled time with timezone, paused reason, uncertainty count |
| Campaign editor | Left ordered workflow; audience/content/schedule editor; persistent preflight summary | Invalid variables, snapshot building/stale, policy gap, no approved template |
| Campaign review | Immutable revision, diff, recipient breakdown, concrete blockers, ceiling/currency, policy and approvals | Management approval and price confirmation separate controls |
| Campaign run | Progress table and breakdown, pace, queued/in-flight, exclusion/failure reasons, pause/cancel | “Pause requested” until authorized attempts settle; cancel consequence shown |
| Templates | Table by WABA/name/language/category/status; editor/preview/version diff and usage | Remote change, rejection reason absent, edit unsupported, stale sync |
| Flows | Versioned JSON, validation with line paths, official preview/publish state, responses | Preview expired, invalid JSON, unsupported schema; no fake drag-and-drop API |
| Meta Health | Asset tree, source-labeled status table, observation time, diagnosis/actions | Unknown distinct from healthy; link out for unsupported Meta actions |
| Developer | Keys/scopes, endpoints, events, deliveries, attempts and sanitized errors | Secret shown once, revoked scope, replay history, payload expired |
| Budget/report | Estimate/exposure/reconciliation columns, period/timezone/currency and provenance | Unconfigured, unknown, gap, over threshold, variance; no fabricated charts |

Desktop 1440 px supports all inbox panes with narrow nav; at 1280 px collapse inspector by default; tablet uses list+thread and inspector drawer; mobile shows list then thread with explicit back/context header. Message operations remain usable on mobile, but large campaign review defaults to a full-screen structured review, not compressed desktop tables. No horizontal scroll required for core reply controls.

## Interaction detail

Send is explicit click or Ctrl/Cmd+Enter (preference); plain Enter creates newline by default to reduce accidental sends. `/` quick template/search only outside editable fields or through an explicit command control. Command menu exposes authorized actions and shortcut help. Roving focus in lists, Escape closes overlay and returns focus, no global shortcuts while typing. Bulk selection shows exact scope: current page versus all matching contacts; bulk campaign snapshot still requires preview.

Paid-send confirmation displays recipient, sender, template/content summary, current estimate, upper bound/currency, pricing policy/time and reason confirmation is needed. Button says “Confirm up to [amount currency]” rather than ambiguous “OK”. If unknown, explain missing pricing evidence and block. A toast cannot serve as approval. Management review separately shows author, approvers/quorum and immutable revision.

Race errors are actionable: assignment changed -> show current owner and allow reload/takeover if permitted; quote expired -> rerun preflight; template changed -> inspect diff and submit new revision; uncertain send -> show attempt time and “Do not resend until reviewed”. Drafts survive page refresh through namespaced sessionStorage (doc 09). If storage is unavailable/full, show “Draft not saved locally” and warn before navigation. Organization switch saves the old draft in its namespace and removes it from the active UI/cache; returning restores only after authorization. Logout clears drafts. Attachment selections may require reattachment after refresh.

Charts appear only when real data exists and an operational comparison benefits: error rate trend, response time distribution, cost variance. Empty reports state “No observations in this interval”. Estimated and reconciled costs cannot share an unlabeled line. Table totals disclose watermark and missing coverage.

## Accessibility and validation

Target [WCAG 2.2 AA](https://www.w3.org/TR/WCAG22/): semantic headings/tables/forms, labels and error association, visible focus, contrast verification, screen-reader announcements for send status without reading every incoming event, reduced motion, zoom/reflow and keyboard-complete workflows. Virtualization must preserve focus and accessible row semantics. Touch layouts enlarge controls rather than scale down desktop.

Validate five tasks with representative employees: triage/reply, expiry/template confirmation, handoff, campaign review/pause, webhook diagnosis. Record task failures and accidental actions; iterate information density and terminology before polishing. No frontend pages or mock application were generated in this phase.
