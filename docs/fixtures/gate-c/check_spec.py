"""Offline Gate C acceptance model. No network, product imports, DB or send code.
This checks the specification's decisions, not a Sprint 3 implementation.
Run: python docs/fixtures/gate-c/check_spec.py
"""
from pathlib import Path
from decimal import Decimal
from datetime import datetime
import json

ROOT = Path(__file__).resolve().parent
BOUNDARY = datetime.fromisoformat("2026-10-01T00:00:00-07:00")
def instant(value):
    result = datetime.fromisoformat(value.replace("Z", "+00:00"))
    assert result.tzinfo is not None
    return result

def evaluate(inputs, rates):
    if inputs.get("kind") == "failure":
        observed = inputs["observation"]
        if observed == "PROVEN_ZERO_REQUEST_BYTES":
            return dict(state="RETRY_ELIGIBLE_AFTER_REAUTHORIZATION", automatic_retry=False, retain_exposure=False)
        if observed == "DEFINITE_INVALID_REJECTION":
            return dict(state="FAILED", automatic_retry=False, retain_exposure=False)
        return dict(state="UNCERTAIN", automatic_retry=False, retain_exposure=True)
    if inputs.get("kind") == "idempotency":
        intents = {}
        conflicts = 0
        for org, key, payload in inputs["submissions"]:
            scoped = (org, key)
            if scoped in intents and intents[scoped] != payload:
                conflicts += 1
            else:
                intents[scoped] = payload
        return dict(intents=len(intents), reservations=len(intents), conflicts=conflicts, network_requests=0)

    age = inputs.get("age_seconds", 300)
    known = age is not None and age >= 0 and inputs.get("qualifying_user_message", True)
    window = "UNKNOWN" if not known else "EXPIRED" if age >= 86400 else "EXPIRING_SOON" if age >= 85500 else "ACTIVE"
    is_template = inputs.get("type", "text") == "template"
    category = inputs.get("category", "UTILITY") if is_template else "SERVICE"
    permission = "ALLOWED" if is_template or (known and age < 86395) else "TEMPLATE_REQUIRED"
    after_change = max(instant(inputs.get("dispatch_at", "2026-09-14T06:00:00Z")),
                       instant(inputs.get("delivery_upper_bound_at", inputs.get("dispatch_at", "2026-09-14T06:00:00Z")))) >= BOUNDARY
    currency = inputs.get("currency", "USD")
    pricing, maximum, price_error = "UNKNOWN", None, None
    if not currency:
        price_error = "CURRENCY_UNKNOWN"
    elif inputs.get("qualified_fep", False):
        pricing, maximum = "ZERO_FEP", Decimal("0")
    elif not after_change and category == "SERVICE":
        pricing, maximum = "ZERO_SERVICE_POLICY", Decimal("0")
    elif not after_change and category == "UTILITY" and known and age < 86400:
        pricing, maximum = "ZERO_UTILITY_REPLY_POLICY", Decimal("0")
    elif after_change and category == "SERVICE" and inputs.get("allowance_proof", False) and inputs.get("service_used", 1000) < 1000:
        pricing, maximum = "ZERO_PROVEN_ALLOWANCE", Decimal("0")
    elif not inputs.get("rate_coverage", True):
        price_error = "PRICING_COVERAGE_MISSING"
    elif not inputs.get("rate_review_current", True):
        price_error = "RATE_REVIEW_OVERDUE"
    elif category == "AUTHENTICATION" and not inputs.get("auth_variant_verified", True):
        price_error = "AUTH_VARIANT_UNKNOWN"
    else:
        date = "2026-10-01" if after_change else "2026-07-01"
        rate = next((r for r in rates if r["currency"] == currency and r["category"] == category and r["effective_local_date"] == date), None)
        if rate is None or rate["list_rate"] is None:
            price_error = "PRICING_COVERAGE_MISSING"
        else:
            pricing = "LIST_RATE"
            maximum = Decimal(inputs.get("synthetic_rate_override", rate["list_rate"]))
    decision = "ALLOW_SPEC_ONLY"
    if not inputs.get("actor_authorized", True):
        decision = "PERMISSION_DENIED"
    elif not inputs.get("consent", True):
        decision = "CONSENT_REQUIRED"
    elif permission != "ALLOWED":
        decision = permission
    elif is_template and inputs.get("template_status", "APPROVED") != "APPROVED":
        decision = "TEMPLATE_INELIGIBLE"
    elif is_template and not inputs.get("template_fresh", True):
        decision = "TEMPLATE_EVIDENCE_STALE"
    elif is_template and inputs.get("approved_category", category) != category:
        decision = "TEMPLATE_CHANGED"
    elif price_error:
        decision = price_error
    elif inputs.get("zero_cost_only", False) and maximum > 0:
        decision = "ZERO_COST_REQUIRED"
    elif maximum > Decimal(inputs.get("confirmed_max", "1.0000")):
        decision = "NEW_CONFIRMATION_REQUIRED"
    elif maximum > Decimal(inputs.get("budget_remaining", "1.0000")):
        decision = "BUDGET_EXCEEDED"
    return dict(window=window, permission=permission, pricing=pricing,
                maximum=str(maximum) if maximum is not None else None, decision=decision)

def main():
    spec = json.loads((ROOT / "pricing-cases.json").read_text(encoding="utf-8"))
    assert spec["gate_c"] == "CLOSED" and not spec["sprint_3_implemented"]
    rates = json.loads((ROOT / "indonesia-rates.json").read_text(encoding="utf-8"))["rows"]
    keys = set()
    for row in rates:
        key = (row["market"], row["currency"], row["category"], row["effective_local_date"])
        assert key not in keys
        keys.add(key)
        assert row["status"] == "EVIDENCE_ONLY_NOT_PUBLISHED"
        assert row["list_rate"] is None or (isinstance(row["list_rate"], str) and Decimal(row["list_rate"]) >= 0)
    tiers = json.loads((ROOT / "indonesia-tiers.json").read_text(encoding="utf-8"))["rows"]
    groups = {}
    for row in tiers:
        groups.setdefault((row["currency"], row["category"]), []).append(row)
    for rows in groups.values():
        assert rows[0]["from"] == 0 and rows[-1]["through"] is None
        for previous, current in zip(rows, rows[1:]):
            assert current["from"] == previous["through"] + 1
            assert Decimal(current["rate"]) <= Decimal(previous["rate"])
    ids = set()
    for case in spec["cases"]:
        assert case["id"] not in ids
        ids.add(case["id"])
        actual = evaluate(case["inputs"], rates)
        for key, value in case["expected"].items():
            got = actual.get(key)
            if key == "maximum" and value is not None:
                assert got is not None and Decimal(got) == Decimal(value), (case["id"], key, got, value)
            else:
                assert got == value, (case["id"], key, got, value)
    print(f"PASS: {len(ids)} offline specification scenarios; {len(rates)} rate rows; {len(tiers)} contiguous tier rows.")
    print("No network requests, database migrations, product authorization or production concurrency proof.")
if __name__ == "__main__":
    main()
