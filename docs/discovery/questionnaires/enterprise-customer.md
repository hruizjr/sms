# Questionnaire — Prospective Enterprise Customers

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — not yet used | Project owner | 2026-11-24 |

**Purpose:** learn what real launch customers send, how they want to send it, and what they expect —
evidence for traffic sizing (workstream B), MVP scope (E), and service levels (F).

**Respondents:** a business owner of the SMS use case **and**, for Part C, someone from the customer's IT or
integration team. Aim for 3–5 prospective customers across different segments (e.g. a bank or e-wallet, a
retailer, a logistics company, a government or education body).

**Do not** ask customers to share real phone numbers, real message content, or personal data. Ask for
counts, ranges, and redacted or synthetic examples. Capture numbers in the
[traffic evidence worksheet](../traffic-evidence-worksheet.md).

## Part A — Use cases and volumes

| # | Question | Register IDs | P |
| - | -------- | ------------ | - |
| CU-01 | What do you use SMS for today (OTP, transaction alerts, reminders, marketing, advisories)? Roughly what share is each? | D-061, D-084 | P0 |
| CU-02 | How many messages do you send per month today, and through which provider? Can you share monthly volume reports? | D-060, D-004 | P0 |
| CU-03 | How many would you send through us in the first month, by month 6, and by month 12? Is that a commitment, an estimate, or an aspiration? | D-060, D-074 | P0 |
| CU-04 | What is your busiest hour or day? How many messages do you send in your largest single campaign, and how quickly must it finish? | D-062 | P1 |
| CU-05 | For OTPs: how quickly must the message reach the phone to be useful? | D-062, D-067 | P1 |
| CU-06 | How long are your messages typically? Do you use emoji or non-English characters? (Redacted samples help.) | D-075 | P1 |
| CU-07 | Which networks are your recipients on (if known)? Must you reach DITO subscribers? Any international recipients? | D-069, D-043 | P1 |
| CU-08 | Do recipients reply to your messages? Do you need keywords, short codes, or two-way conversations? | D-063, D-042 | P1 |

## Part B — Product needs

| # | Question | Register IDs | P |
| - | -------- | ------------ | - |
| CU-10 | Would you send through an API, a web portal, file uploads, or a mix? | D-089 | P0 |
| CU-11 | Which features are must-haves on day one (e.g. scheduling, contact groups, CSV import, delivery reports, webhooks, sub-accounts, user roles)? Which can wait? | D-089, D-064 | P0 |
| CU-12 | Do you need separate sub-accounts (departments, branches, or your own clients)? | D-064, D-088 | P1 |
| CU-13 | What delivery information do you need, how quickly, and for how long must it remain available? | D-099, D-072, D-073 | P1 |
| CU-14 | Which sender IDs (brand names) do you use? Are they registered with the telcos already? | D-103, D-022 | P1 |
| CU-15 | How do you collect consent and handle opt-outs today? Would you expect us to manage opt-out lists? | D-024, D-122 | P1 |
| CU-16 | Do you need reports or integrations with other systems (accounting, CRM)? | D-065, D-072 | P2 |
| CU-17 | Is an English-only interface acceptable? | D-046 | P2 |

## Part C — Technical integration (customer IT)

| # | Question | Register IDs | P |
| - | -------- | ------------ | - |
| CU-20 | How does your system handle timeouts when calling an SMS API — does it retry automatically? Could it send an idempotency key with each request? | D-094 | P1 |
| CU-21 | Do you need webhooks (we call your system when a status changes), or will you poll? | D-099 | P1 |
| CU-22 | Do you need IP allowlisting, specific authentication methods, or security reviews before integrating? | D-067 | P1 |

## Part D — Commercial and service expectations

| # | Question | Register IDs | P |
| - | -------- | ------------ | - |
| CU-30 | How do you pay today: prepaid credits or monthly invoice? What would you expect from us? | D-082, D-081 | P1 |
| CU-31 | What do you pay today (range is fine), and do you pay for undelivered messages? | D-084, D-080 | P0 |
| CU-32 | What uptime and support hours do you require? Any contractual penalties? | D-067, D-079 | P1 |
| CU-33 | Do you require your data to be hosted in the Philippines, or any certifications or audits? | D-068, D-125 | P1 |
| CU-34 | What onboarding documents could you provide (business registration, sender ID authorization letters)? | D-085 | P1 |
| CU-35 | What would make you switch from your current provider? | D-162 | P1 |

**Evidence to request:** written volume estimate (signed or on letterhead if possible); historical volume
reports; redacted message samples; feature priority list.
