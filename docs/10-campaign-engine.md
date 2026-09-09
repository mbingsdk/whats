# Campaign engine

Campaigns schedule guarded sends; they never own a separate Graph client. State machines are in doc 04; dispatch correctness in docs 08/12. A campaign run is one immutable approved revision and sealed audience snapshot. A change of sender, template content/category/language, personalization, membership, schedule bounds or spending ceiling creates a new revision and invalidates approval.

## Audience and preflight

Dynamic segment: an allowlisted predicate over contact fields/tags/country/consent/suppression/interactions/campaign history. The preview is time-stamped and advisory. Snapshot: immutable selected identities and frozen personalized content at a declared cutoff. Recipient: one execution row per run/snapshot member. Segment membership never changes an approved campaign automatically.

Build a snapshot from a consistent PostgreSQL read transaction into staging. Freeze source version and cutoff, normalize identities, deduplicate by scoped destination, resolve variables/fallbacks and classify exclusions. At initial 100k scale use one bounded repeatable-read job; if it exceeds the proposed 5-minute limit, fail and rebuild without sealing partial membership. Worker chunks may read staged rows later without rerunning the live audience query. Seal counts and ordered membership/content digest atomically. Late-arriving historical attributes are outside that snapshot; consent/suppression and other final guards are always current.

| Preflight check | Result and consequence |
| --- | --- |
| Sender/binding/capability/health | Missing access or stale critical capability blocks; degraded quality can require supervisor review or pause |
| Template status/category/content | Exact observed sendable version/language; rejected/paused/unknown blocks |
| Recipient identity | Malformed/ambiguous target excluded with reason; no country guessed from profile |
| Variables/media | Missing fallback, invalid component or unavailable asset excludes/blocks explicitly |
| Consent/suppression | Appropriate active evidence required; revocation always excludes |
| Duplicate/frequency | One destination per snapshot; frequency policies estimate eligibility and reserve at dispatch |
| Pricing | Verified policy coverage for proposed send interval; per-market/category upper bound; unknown blocks |
| Budgets | All applicable scopes checked; preview availability is not reservation |
| Approval/confirmation | Policy result, eligible approvers, expiry and signed scope displayed separately |
| Schedule | Explicit timezone/offset, deadline and authority validity; future pricing gaps block |

Excluded recipients retain reason codes and source evidence references. Totals distinguish initial exclusions from final dispatch exclusions. A preview score, if introduced, is labeled **Internal campaign risk indicator**; do not combine arbitrary inputs into an unexplained numeric Meta-like score. Initially use concrete blocker/warning counts instead.

## Approval and execution authority

Submission freezes campaign revision/snapshot/estimate ceiling and creates reusable approval request. Threshold predicates can use cost, category, count and actor grant. Default self-approval forbidden, quorum determined by policy, approvers deduplicated. Campaign Manager cannot approve merely from its role name. Auto-approval below threshold is an explicit policy decision with evidence.

Management approval allows an exact execution scope; pricing confirmation accepts its monetary bound. Neither substitutes for consent or sender health. Campaign confirmation issues a run-bound authority with recipient membership/content digest, count limit, currency, policy versions, time bounds and maximum total exposure. Child intents can consume this authority only for their snapshot membership and remaining ceiling. Do not reuse a one-recipient token for a whole campaign.

Campaign is organization-owned with an active sponsor. Offboarding creator does not silently delete approved work. Losing sponsor/required current approval authority pauses unsent work pending reassignment/review. Permission/policy revocation invalidates queued authority. No worker impersonates a logged-out user session.

## Dispatch protocol

1. Scheduler claims due run and verifies immutable revision, sealed snapshot, approval/confirmation, deadline and current capabilities. Transition QUEUED -> RUNNING in transaction.
2. Generate recipient rows with unique `(run,snapshot_member)` and one immutable intent per recipient. Bounded batches (proposed 100 claims) allow fairness, not API batch sending.
3. Obtain current rate/throttle capacity; then inside the short canonical lock transaction recheck run epoch/state, suppression/consent, template/category/hash, window, policy, authority, frequency and all budgets. Reserve exposure and create short-lived permit.
4. Just before network work, dispatcher reacquires the canonical locks, reruns all current guards (including suppression, authority, window and price), verifies permit/run revisions and reserved coverage, and atomically consumes the permit into DISPATCHING. A changed relevant policy/resource revision requires a fresh gate result; ordinary recipient progress does not update the run or organization barrier. Pause/opt-out/role changes serialize with this final authorization point. Call Meta outside transaction. External messages already in flight cannot be stopped.
5. Classify acceptance, proven rejection or uncertainty. Apply safe backoff only for proven transient non-acceptance. Preserve attempt evidence and correlation. Recheck all controls before retry.
6. When no dispatchable work remains, enter DRAINING; outstanding attempts settle or become UNCERTAIN. Complete processing with explicit failed/excluded/uncertain counts. Late delivery/read events update analytics.

DB fencing protects local state but not already-issued network traffic. Another worker must not retry a DISPATCHING attempt after lease expiry. A stale process returning later may contribute acceptance evidence through a reconciliation transaction; it cannot authorize another attempt.

## Throttling, pause and cancellation

Separate campaign concurrency from inbox and webhook pools. Weighted fairness per organization/phone prevents a large campaign starving customer service. Start with a low internal configurable dispatch rate (set during controlled test); do not claim a fabricated Meta throughput. Enforce verified app/WABA/phone/recipient restrictions plus internal limits. Rate-limit errors reduce pace/circuit-break affected asset; do not rotate phone numbers to evade restrictions.

Pause commits a dispatch_epoch change and prevents further permits. Unconsumed reservations are released when confirmed undispatched. UI shows `pausing/in_flight_count` until authorized network attempts settle; after this, PAUSED remains. Resume reevaluates deadline, consent, prices, template, budget and authority; changed material conditions require approval/confirmation. Cancel stops remaining work permanently, preserves accepted/uncertain attempts and releases only provably undispatched exposure.

Frequency policy is organization-local marketing policy, e.g. configurable maxima per 1/7/30 days without proposed production values. Count reservations + accepted/uncertain sends across campaigns, API and automation to the same contact. Rejected-before-acceptance attempts can release count. A contact identity merge conservatively combines exposure; never evade frequency by importing aliases.

## Measurement and attribution

Show selected, initially eligible/excluded, finally excluded, pending, accepted, uncertain, failed, delivered, read and replied separately. Sent/delivered/read use source event evidence; missing read is unknown. Processing completion is not delivery completion.

Reply attribution: first use explicit message context pointing to a campaign message; otherwise optional internal heuristic within a configured attribution interval and same sender/contact. Store method/version/confidence; a reply cannot automatically count toward every overlapping campaign. Link tracking optional and source-labeled. Cost reports show frozen estimate, committed exposure, Meta pricing facts and reconciled amount with source/coverage. Per-recipient billing stays null where only aggregate invoice evidence exists.

## Freshness safety policy

Final eligibility uses committed current state, not unprocessed raw inbound bytes. Prioritize webhook ingestion/opt-out projection above bulk dispatch. As an internal proposed default, pause marketing dispatch when oldest relevant unprocessed inbound exceeds 30 seconds, or when current sender/template/pricing evidence exceeds configured freshness bounds. This reduces an otherwise invisible opt-out race but does not claim to cancel requests already authorized before an opt-out was observed. Record both provider event time and suppression commit time for review.
