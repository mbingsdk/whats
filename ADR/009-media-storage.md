# ADR 009: private external media storage

Download verified incoming media promptly into private S3-compatible storage after validation/scan; serve authorized short-lived application links. Preserve Meta media ID and lifecycle evidence separately. Choose S3/R2 provider based on approved region, retention, cost and operational access; provider is undecided, not provisioned.

Keeping all media in PostgreSQL complicates backups and bloats storage. VPS-local files couple message evidence to one host failure. Indefinite proxying from Meta assumes availability/retention we cannot guarantee. External storage separates capacity and recovery while retaining explicit reference/erasure management.

Consequences: monitor download expiry, object orphan cleanup, malware state and credential scope. Deduplicate only within tenant. Signed URLs have a bounded revocation lag; use authenticated streaming for stricter needs. Erasure covers previews/versions/backups per approved policy.

Revisit storage classes/replication after actual media volume and retention requirements are known. Do not choose a vendor from unverified pricing claims.
