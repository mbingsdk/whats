# ADR 006: envelope encryption

Encrypt recoverable secrets with AEAD and per-secret data keys, wrapped by a root key outside PostgreSQL. Bind ciphertext to org/resource/purpose/version using associated data. API keys and session tokens use one-way digests because only verification is needed; Meta/signing/TOTP secrets must be recoverable by restricted services.

Initial deployment can use a protected systemd credential/root-key file with separate recovery custody. Managed KMS is preferable when available and operationally supported. A new self-hosted Vault service is unnecessary initially; DB-only encryption under a key stored in the same DB does not meet the separation requirement.

Residual risk: a fully compromised API/worker host may access secrets it can decrypt. Root-key recovery, permissions, rotation and host hardening matter. Keep keys out of images/builds/logs/DB backups. Retain old wrapping keys only for required ciphertext/backup retention; test decrypt after restore.

Revisit KMS/HSM-backed custody if regulatory obligations, operator separation or threat model require it. Do not describe encryption as a substitute for least privilege.
