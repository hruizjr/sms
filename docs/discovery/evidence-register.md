# Evidence Register

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Template — **no evidence recorded yet** | Project owner | 2026-11-24 |

This register lists every piece of evidence used to resolve a [discovery register](../requirements/discovery-register.md)
item. It records **references and non-confidential summaries only**.

## Rules

1. **Never store confidential documents or secrets here.** Telco agreements, rate cards, technical
   specifications under NDA, customer volume letters, legal opinions, and any credentials stay in the
   owner's secure document store. Record only the document's title, version, date, and storage location
   label (e.g. "Owner drive / Telco / Smart / 2026-12-01 term sheet v2").
2. Summaries must not reveal confidential commercial terms (e.g. write "rate card received; precision
   4 decimal places" rather than the rates themselves, if the rates are confidential).
3. Each entry is reviewed by the owner before any discovery item's status changes.
4. Secondary sources (news, blogs, AI summaries) may be logged as **leads** but can never be the sole
   evidence for confirming an item.
5. Evidence is never deleted. Superseded evidence is marked `Superseded by EV-xxxx`.

## Fields

| Field | Meaning |
| ----- | ------- |
| **Evidence ID** | `EV-0001`, `EV-0002`, … (sequential, never reused) |
| **Discovery item ID(s)** | One or more register IDs, e.g. `D-101, D-108` |
| **Source / organization** | Who provided it (organization and the person's role, not personal contact details) |
| **Document or meeting reference** | Title, version, date, and storage location label; or meeting date and attendees' roles |
| **Date received** | `YYYY-MM-DD` |
| **Summary** | Non-confidential summary of what it states |
| **Evidence type** | Signed/official document · Written statement · Official public source · Meeting notes · Secondary/lead |
| **Verification status** | `Unreviewed` · `Reviewed — accepted` · `Reviewed — insufficient` · `Contradicted by EV-xxxx` · `Superseded by EV-xxxx` |
| **Confidentiality** | `Public` · `Internal` · `Confidential (stored outside Git)` · `Restricted (legal/privileged, stored outside Git)` |
| **Owner reviewer** | Who reviewed it, and the review date |
| **Related ADR / requirement** | e.g. `ADR-0019`, `MVP-17`, exit criterion `EX-2` |

## Register

| Evidence ID | Discovery item ID(s) | Source / organization | Document or meeting reference | Date received | Summary | Evidence type | Verification status | Confidentiality | Owner reviewer | Related ADR / requirement |
| ----------- | -------------------- | --------------------- | ----------------------------- | ------------- | ------- | ------------- | ------------------- | --------------- | -------------- | ------------------------- |
| — | — | — | — | — | *No evidence recorded yet.* | — | — | — | — | — |

## Example (illustrative only — not real evidence)

| Evidence ID | Discovery item ID(s) | Source / organization | Document or meeting reference | Date received | Summary | Evidence type | Verification status | Confidentiality | Owner reviewer | Related ADR / requirement |
| ----------- | -------------------- | --------------------- | ----------------------------- | ------------- | ------- | ------------- | ------------------- | --------------- | -------------- | ------------------------- |
| EV-XXXX | D-100, D-101 | *Telco X — integration team lead* | *"A2P interface specification v1.2", stored at Owner drive / Telco X / specs* | *YYYY-MM-DD* | *States SMPP version and bind limits (values kept in the stored document)* | *Signed/official document* | *Unreviewed* | *Confidential (stored outside Git)* | *—* | *ADR-0012* |

## Public research references

Public legal and regulatory texts consulted during Phase 2A are logged in the
[legal and regulatory research log](legal-regulatory-research.md) (`LR-xx` IDs). They establish what a text
says, not how it applies to us; counsel's opinion is the evidence for regulatory items.
