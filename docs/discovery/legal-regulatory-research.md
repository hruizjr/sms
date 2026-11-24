# Legal and Regulatory Research Log

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — preliminary desk research; **not legal advice** | Project owner | 2026-11-24 |

> **Important.** This log records what public texts say and where they were read. It is **not a legal
> opinion**. AI-assisted research can misread, miss amendments, or rely on unofficial copies. Every item
> must be confirmed by qualified Philippine counsel before anyone relies on it. Questions for counsel are
> in the [legal counsel questionnaire](questionnaires/legal-counsel.md).

## How to read this log

- **Source type:** *Primary (official)* = regulator or government site; *Primary text (unofficial copy)*
  = full text of the law reproduced by a third-party legal library (here lawphil.net), not the Official
  Gazette; *Secondary* = commentary, news, or summaries — **leads only**.
- **Access status:** *Read* = text actually read; *Partly read*; *Not accessed* = the page refused
  automated access (official NTC, NPC, and Supreme Court E-Library pages returned HTTP 403 on
  2026-11-24, and one legal library rate-limited requests).
- **Date accessed** for all entries: **2026-11-24**.
- Nothing here changes any [discovery register](../requirements/discovery-register.md) status.

---

## A. Telecommunications licensing and registration

### LR-01 — Public Telecommunications Policy Act (RA 7925)

| Field | Detail |
| ----- | ------ |
| Question | Does an SMS aggregator need a franchise or other authorization as a value-added service (VAS) provider? |
| Source | https://lawphil.net/statutes/repacts/ra1995/ra_7925_1995.html — Primary text (unofficial copy), Read |
| Date | Approved 1 March 1995; Sec. 27: effective 15 days after publication |
| What it establishes | Sec. 3(h) defines a VAS provider as an entity that, "relying on the transmission, switching and local distribution facilities" of carriers, "offers enhanced services beyond those ordinarily provided for by such carriers." Sec. 11: "Provided that it does not put up its own network, a VAS provider need not secure a franchise." Sec. 11 also sets conditions for *telecommunications entities* that provide VAS (prior NTC approval, non-discrimination, separate books). Sec. 16 requires a franchise for public telecommunications entities. |
| What it does not establish | Whether an A2P SMS aggregator is a VAS provider; any registration requirement for standalone VAS providers (Sec. 11 does not state one); later changes (see LR-02, LR-03, LR-04). |
| Needs counsel | Classification of our operating model; whether any part of our setup (e.g. VPN links, own infrastructure) counts as "putting up a network". |
| Register items | D-120, D-160 |

### LR-02 — Konektadong Pinoy Act (RA 12234)

| Field | Detail |
| ----- | ------ |
| Question | Did the 2025 connectivity law change how VAS providers are regulated? |
| Source | https://lawphil.net/statutes/repacts/ra2025/ra_12234_2025.html — Primary text (unofficial copy), Read (in two parts) |
| Date | Lapsed into law 24 August 2025; Sec. 35: effective 15 days after publication (publication date not shown in the copy read) |
| What it establishes | Sec. 3(i): public telecommunications entities and VAS providers as defined under RA 7925 "are considered DTIPs [data transmission industry participants] to the extent of their businesses engaged in data transmission services." Sec. 3(h) defines data transmission (includes VoIP; excludes basic telephone services). Sec. 8(a): "All DTIPs shall be required to register with the NTC." Sec. 34 repeals, "insofar as they apply to DTIPs", several provisions of RA 7925, including "the first sentence of the first paragraph of Section 11" and "the first paragraph of Section 16". Sec. 30(b): earlier NTC registrations for data transmission remain valid. Sec. 31: implementing rules due within 90 days of effectivity. The Act mentions SMS only as a complaint channel (Sec. 25(f)). |
| What it does not establish | Whether A2P SMS aggregation is "data transmission" making us a DTIP; the content of the implementing rules (IRR) and NTC eligibility criteria. Secondary sources report the IRR has been issued, with inconsistent dates — **not verified**. |
| Needs counsel | Whether we are a DTIP, a VAS provider, both, or neither; which NTC registration(s) apply; current IRR status. |
| Register items | D-120 |

### LR-03 — NTC Memorandum Circular No. 02-05-2008 (Value Added Services)

| Field | Detail |
| ----- | ------ |
| Question | What NTC registration applies to VAS providers? |
| Source | https://elibrary.judiciary.gov.ph/thebookshelf/showdocs/10/56045 — Primary (official reproduction), **Not accessed** (HTTP 403) |
| Date | 2008 (exact dates not verified) |
| What it establishes | **Nothing verified.** Search-result snippets (secondary) state that no entity may offer VAS without a Certificate of Registration from the NTC, with a term of up to five years, and that SMS is treated as a VAS. |
| Needs counsel | Whether this circular is current, how it interacts with LR-02 and LR-04, and whether it covers our model. |
| Register items | D-120 |

### LR-04 — NTC Memorandum Order No. 005-06-2025 (VAS registration)

| Field | Detail |
| ----- | ------ |
| Question | Are there newer VAS registration rules? |
| Source | Official text **not found** (ntc.gov.ph returned HTTP 403). Secondary: https://apsaylaw.com/understanding-ntc-memorandum-order-no-005-06-2025/ (law-firm article), Read |
| Date | 2025 per the article (not verified) |
| What it establishes | **Nothing verified.** The article *claims* the order requires a Certificate of Registration for any VAS, centralizes filing at the NTC Central Office, and lists SMS services among VAS. |
| Needs counsel | Existence, text, and effect of the order. |
| Register items | D-120 |

### LR-05 — Implementing Rules and Regulations of RA 12234

| Field | Detail |
| ----- | ------ |
| Question | How does the NTC register DTIPs under the Konektadong Pinoy Act, and does it cover VAS such as SMS? |
| Source | Official text **not accessed**. Secondary: law-firm alerts (e.g. https://www.quisumbingtorres.com/en/alerts/2026/01/irr-law-removing-congressional-franchise-requirement) and a Supreme Court E-Library listing seen only in search results |
| Date | Reported as issued in late 2025 / effective early 2026 — **inconsistent across secondary sources; not verified** |
| What it establishes | **Nothing verified.** Secondary sources report that entities providing data transmission services obtain DTIP registration from the NTC through an administrative procedure. |
| Needs counsel | IRR text, effectivity, eligibility criteria, and whether they apply to our model. |
| Register items | D-120 |

## B. Messaging consent, opt-out, and broadcast rules

### LR-06 — NTC MC No. 03-03-2005 (Rules and Regulations on Broadcast Messaging Service)

| Field | Detail |
| ----- | ------ |
| Question | What consent, opt-out, and timing rules apply to bulk/promotional SMS? |
| Source | E-Library copy (HTTP 403) and jur.ph reproduction (HTTP 429) — **Not accessed** |
| Date | 2005 (not verified) |
| What it establishes | **Nothing verified.** Search-result summaries (secondary) describe prior consent/opt-in for commercial or promotional broadcast messages, an opt-out mechanism, sender contact details and opt-out instructions in messages, and no broadcast messages between 9:00 PM and 7:00 AM except for subscribed services. |
| Needs counsel | Current text and amendments; whether it applies to A2P messages sent by an aggregator on behalf of business customers; transactional vs. promotional distinctions. |
| Register items | D-122, D-123, D-024 |

### LR-07 — NTC MC No. 04-07-2009 and Memorandum Order 08-07-2009

| Field | Detail |
| ----- | ------ |
| Question | Were "push messages" prohibited? |
| Source | jur.ph reproductions — **Not accessed** (HTTP 429) |
| What it establishes | **Nothing verified directly.** The 2018 circular (LR-08) states in its whereas clause that MC 04-07-2009 was issued "to address concerns on disappearing and/or vanishing loads by, among others, prohibiting the sending or initiating of push messages." |
| Needs counsel | Text, scope of "push messages", and relevance to A2P traffic. |
| Register items | D-122 |

### LR-08 — NTC MC No. 07-08-2018 (Further amending MC 03-03-2005)

| Field | Detail |
| ----- | ------ |
| Question | What opt-in/opt-out mechanics are required? |
| Source | https://dev.lawyerly.ph/laws/91581 — **Secondary reproduction**, Read in full |
| Date | Adopted 30 August 2018; filed 31 August 2018; effective 15 days after publication (publication date not stated) |
| What it establishes (per the reproduction) | Addressed to "Content and/or Information providers and/or PTEs", in the context of content/information services and prepaid load deductions: opt-in confirmation with service details and opt-out instructions; "at least double opt-in confirmation process"; opt-in/opt-out records kept "for at least six (6) months" after opt-out; universal opt-out keywords "STOP, END, CANCEL, UNSUBSCRIBE, and QUIT"; opting in/out free of charge. |
| Needs counsel | Whether it applies to A2P business messaging (vs. paid content subscriptions); confirm against an official copy. |
| Register items | D-122, D-024 |

## C. Data privacy

### LR-09 — Data Privacy Act of 2012 (RA 10173)

| Field | Detail |
| ----- | ------ |
| Question | What are our privacy roles and core obligations? |
| Source | https://lawphil.net/statutes/repacts/ra2012/ra_10173_2012.html — Primary text (unofficial copy), Partly read (verbatim through Sec. 27); official page https://privacy.gov.ph/data-privacy-act/ returned HTTP 403 |
| Date | 2012 (approval and effectivity dates not verified from verbatim text) |
| What it establishes | Sec. 3(h) personal information controller (PIC): controls collection, holding, processing, or use, including instructing another to process on its behalf; excludes one who "performs such functions as instructed by another person or organization". Sec. 3(i) personal information processor (PIP): one "to whom a personal information controller may outsource the processing". Sec. 11(e): retain "only for as long as necessary". Sec. 12: lawful processing criteria (consent, contract, legal obligation, vital interests, public functions, legitimate interests). Sec. 14: a PIC may subcontract processing and remains responsible for safeguards; the processor "shall comply with all the requirements of this Act". Sec. 16: data subject rights. Sec. 20(d): PICs ensure third-party processors implement security measures. Sec. 20(f): prompt notification of the NPC and data subjects for certain breaches likely to cause "real risk of serious harm" (no fixed hour count in the Act text). Sec. 21: PIC accountability includes information transferred to third parties "whether domestically or internationally". |
| Needs counsel | Our role for each data set (customer-uploaded contacts and messages: likely processor; our own account and billing data: controller — **to be confirmed**); data processing agreement terms; cross-border hosting (D-129). |
| Register items | D-121, D-124, D-129, D-073 |

### LR-10 — Implementing Rules and Regulations of RA 10173 (2016)

| Field | Detail |
| ----- | ------ |
| Source | https://privacy.gov.ph/implementing-rules-regulations-data-privacy-act-2012/ — **Not accessed** (HTTP 403) |
| What it establishes | **Nothing verified.** Breach-notification timing, outsourcing, and retention details in the IRR remain unconfirmed. |
| Needs counsel | Breach notification timing and content; outsourcing agreement requirements. |
| Register items | D-121 |

### LR-11 — NPC Circular No. 2022-04 (Registration of data processing systems and DPOs)

| Field | Detail |
| ----- | ------ |
| Source | https://privacy.gov.ph/wp-content/uploads/2023/05/Circular-2022-04.pdf — **Not accessed** (HTTP 403); no copy found elsewhere |
| What it establishes | **Nothing verified.** Search-result snippets (secondary) report mandatory registration for PICs and PIPs meeting thresholds (e.g. 250 or more employees, or processing sensitive personal information of 1,000 or more individuals), via an online system, effective in early 2023. |
| Needs counsel | Whether we must register, when, and the DPO requirement. |
| Register items | D-121 |

## D. Fraud, identity, and numbering

### LR-12 — SIM Registration Act (RA 11934)

| Field | Detail |
| ----- | ------ |
| Source | https://lawphil.net/statutes/repacts/ra2022/ra_11934_2022.html — Primary text (unofficial copy), Read |
| Date | Approved 10 October 2022; Sec. 17: effective 15 days after publication |
| What it establishes | End users must register SIMs with public telecommunications entities (PTEs) (Sec. 4). PTEs must provide "user-friendly reporting mechanisms" for fraudulent texts and may deactivate the SIM used (Sec. 6). Sec. 3(g) defines "spoofing" as transmitting misleading or inaccurate information about the source of a call or text message with intent to defraud; Sec. 11(e) penalizes spoofing a registered SIM. The text does not mention aggregators, alphanumeric sender IDs, or business messaging. |
| Needs counsel | Any implications for sender ID practices and our liability for customers' spoofing. |
| Register items | D-128, D-123 |

### LR-13 — Anti-Financial Account Scamming Act (RA 12010)

| Field | Detail |
| ----- | ------ |
| Source | https://lawphil.net/statutes/repacts/ra2024/ra_12010_2024.html — Primary text (unofficial copy), Read |
| Date | Approved 20 July 2024; Sec. 26: effective 15 days after publication |
| What it establishes | Duties (risk controls, holds, verification) fall on "Institutions" — BSP-supervised banks, non-banks, and payment providers (Sec. 3(f), 6–10). Sec. 3(b) "electronic communications" includes SMS. Sec. 3(g) defines a "mass mailer" as a service or software used to send electronic communications to 50 or more recipients. Sec. 4 prohibited acts apply to any person. The Act does not mention telecommunications entities or impose duties on them. Enforcement: BSP, with IRR due within one year (Sec. 23). |
| Needs counsel | Whether any provision referring to mass mailers creates obligations or liability exposure for an SMS platform; what banking customers will require of us contractually. |
| Register items | D-131, D-123 |

### LR-14 — Mobile Number Portability Act (RA 11202)

| Field | Detail |
| ----- | ------ |
| Source | https://lawphil.net/statutes/repacts/ra2019/ra_11202_2019.html — Primary text (unofficial copy), Read |
| Date | Approved 8 February 2019; Sec. 14: effective 15 days after publication |
| What it establishes | Subscribers may retain their mobile number when changing provider (Sec. 4(d)), free of charge (Sec. 6(a)); porting within 48 hours (Sec. 4(i)). No provision on message routing or number databases available to third parties. |
| Implication (engineering, not legal) | A number's prefix does not reliably identify the recipient's current network (D-070). |
| Register items | D-070 |

## E. Corporate relationships

### LR-15 — PLDT Enterprise and Smart

| Field | Detail |
| ----- | ------ |
| Source | https://pldtenterprise.com/about-us — company website, **Partly read**; https://smart.com.ph pages returned no content; https://www.pldt.com redirected to a consumer site (not followed) |
| What it establishes | **Unverified lead only:** a summary of the PLDT Enterprise page describes PLDT Enterprise as "the corporate business arm of PLDT". This wording was not read verbatim. No primary statement of the Smart–PLDT relationship was obtained. |
| Needs confirmation | Ask PLDT Enterprise and Smart directly (D-112); corporate filings may confirm group structure. |
| Register items | D-112 |

---

## Sources that could not be accessed (to obtain through counsel or manually)

| Item | Why |
| ---- | --- |
| NTC MC 02-05-2008; NTC MO 005-06-2025; RA 12234 IRR | Official NTC and E-Library pages refused automated access |
| NTC MC 03-03-2005; MC 04-07-2009; MO 08-07-2009 | E-Library refused; legal library rate-limited |
| RA 10173 IRR; NPC Circular 2022-04; any NPC consent guidelines | privacy.gov.ph refused automated access |
| Smart corporate profile | Pages returned no content |

The owner or counsel can obtain these directly; record any copies received in the
[evidence register](evidence-register.md) (reference only).
