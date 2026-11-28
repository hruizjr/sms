// Code generated from contracts/openapi/portal-api-v1.yaml by openapi-typescript. DO NOT EDIT.
// Regenerate with: pnpm --filter @sms/client-portal generate:api
/* eslint-disable */
export interface paths {
    "/portal/v1/api-keys": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * List API keys
         * @description The organization's API keys, active and revoked, oldest first. A key's secret and its stored
         *     hash are never returned. Requires `api_keys:manage`.
         */
        get: operations["listApiKeys"];
        put?: never;
        /**
         * Create an API key
         * @description Creates an API key of the organization with a name and the public API permissions (scopes) it
         *     grants. The owner must re-enter their current password (`current_password`, owner decision G4-7):
         *     a wrong password, or an account that is temporarily locked, receives `403`
         *     (`reauthentication-failed`) and no key is created. Failed attempts count toward the same account
         *     lockout as failed sign-ins. The password is never stored or logged. `503` (`service-busy`) means
         *     the password could not be checked now and no key was created. The response contains the full key
         *     (`key`) **once**: the platform stores only a hash of its secret and can never return it again.
         *     Requires `api_keys:manage`.
         */
        post: operations["createApiKey"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/portal/v1/api-keys/{api_key_id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        /**
         * Revoke an API key
         * @description Revokes an API key of the organization. The key is refused by the public API from the next
         *     request on (`401`). The key stays listed as `revoked`; revocation cannot be undone. Revoking a
         *     key that is already revoked returns it unchanged. A key of another organization is reported as
         *     `404`, exactly like a key that does not exist. Requires `api_keys:manage`.
         */
        delete: operations["revokeApiKey"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/portal/v1/audit-events": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * List security audit events
         * @description The organization's security audit events, newest first, optionally filtered by action, outcome
         *     and time range. Read-only: this API has no way to change or delete an event, and the application
         *     cannot change or delete them in the database either (a database superuser still can; see the
         *     Security audit log section). Requires `audit_log:read`
         *     (owners); other roles receive `403` and the refusal is itself recorded.
         */
        get: operations["listAuditEvents"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/portal/v1/message-estimates": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Estimate encoding and segments
         * @description Returns the encoding and segment count the platform would calculate for a message text, using
         *     the platform's one shared segmentation function. Nothing is stored or sent. Requires
         *     `messages:send`.
         */
        post: operations["estimateMessage"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/portal/v1/messages": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * List messages
         * @description The organization's messages, newest first. Requires `messages:read`.
         */
        get: operations["listMessages"];
        put?: never;
        /**
         * Submit one SMS
         * @description Accepts one SMS to one recipient (D-089: one recipient per portal send) through the same
         *     messaging engine as `POST /v1/messages`, with the same idempotency rules (ADR-0018): the portal
         *     generates one `Idempotency-Key` per intended send and reuses it on retries, so a repeated click
         *     or a network retry cannot create a second message. Requires `messages:send`.
         */
        post: operations["createMessage"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/portal/v1/messages/{message_id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Read a message and its status history
         * @description One message of the organization with its customer-visible status history. A message of another
         *     organization is reported as `404`, exactly like a message that does not exist. Requires
         *     `messages:read`.
         */
        get: operations["getMessage"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/portal/v1/sender-ids": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * List approved sender IDs
         * @description The organization's approved sender IDs, in order. Requires `messages:read`.
         */
        get: operations["listSenderIds"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/portal/v1/session": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Read the current session
         * @description Returns the signed-in user, the organization, the user's permissions and the CSRF token.
         *     `401` means there is no valid session (never signed in, expired, signed out or revoked).
         */
        get: operations["getSession"];
        put?: never;
        /**
         * Sign in
         * @description Verifies an email address and password and starts a session (sets the session cookie). Unknown
         *     addresses, wrong passwords and temporarily locked accounts all receive the same `401`
         *     (`invalid-credentials`). A correct password for an organization that is not active receives
         *     `403` (`account-not-active`) and no session. Too many requests or failed attempts from the
         *     caller's source receive `429` (`too-many-sign-in-attempts`) with `Retry-After`; nothing is
         *     checked. `503` (`service-busy`) means the password could not be checked now; retry after
         *     `Retry-After`.
         */
        post: operations["signIn"];
        /**
         * Sign out
         * @description Revokes the current session on the server and clears the session cookie.
         */
        delete: operations["signOut"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        /** @description An API key's metadata. Never contains the key's secret or its stored hash. */
        ApiKey: {
            /** Format: date-time */
            created_at: string;
            id: components["schemas"]["ApiKeyId"];
            /**
             * @description When the key last authenticated a public API request it was authorized for, recorded at most
             *     once a minute (so up to a minute older than its most recent request); `null` if never used.
             */
            last_used_at: string | null;
            /**
             * @description The name given at creation, to recognize the key.
             * @example Order notifications
             */
            name: string;
            /**
             * @description The key's visible, non-secret beginning (everything before the secret), to match a key in
             *     an application's configuration.
             */
            prefix: string;
            revoked_at: string | null;
            scopes: components["schemas"]["ApiKeyScope"][];
            /** @enum {string} */
            status: "active" | "revoked";
        };
        ApiKeyCreated: {
            api_key: components["schemas"]["ApiKey"];
            /**
             * @description The full API key, for the `Authorization: Bearer` header of the public API. Returned only
             *     in this response; it cannot be retrieved later.
             */
            key: string;
        };
        ApiKeyCreateRequest: {
            /**
             * Format: password
             * @description The signed-in owner's current password, to confirm the action (re-authentication). Never
             *     stored or logged.
             */
            current_password: string;
            /**
             * @description A name to recognize the key (leading and trailing spaces are removed; no control characters).
             * @example Order notifications
             */
            name: string;
            scopes: components["schemas"]["ApiKeyScope"][];
        };
        /**
         * @description Opaque API key identifier (the key ID part of the key).
         * @example 06h1wh346hsf13tppjr44p43pw
         */
        ApiKeyId: string;
        ApiKeyList: {
            data: components["schemas"]["ApiKey"][];
        };
        /**
         * @description A public API permission granted by the key (D-143).
         * @enum {string}
         */
        ApiKeyScope: "messages:send" | "messages:read";
        /**
         * @description What happened: `portal.sign_in` (a sign-in, successful or not), `portal.sign_out`,
         *     `portal.account_locked` (repeated failed attempts locked the account), `api_key.create`,
         *     `api_key.revoke`, `api_key.list` (recorded only when refused), `audit_log.list` (recorded only
         *     when refused), and, performed by the platform's administration tool, `portal_user.create`,
         *     `portal_user.unlock` (the account lockout was cleared), `portal_user.sessions_revoke` (the
         *     user's sessions were ended) and `sender_id.approve` (a sender ID was approved for the
         *     organization; `details.sender_id` names it).
         * @enum {string}
         */
        AuditAction: "portal.sign_in" | "portal.sign_out" | "portal.account_locked" | "api_key.create" | "api_key.revoke" | "api_key.list" | "audit_log.list" | "portal_user.create" | "portal_user.unlock" | "portal_user.sessions_revoke" | "sender_id.approve";
        /** @description Structured, non-secret details. Only the fields that apply are present. */
        AuditDetails: {
            /** @description The name of the API key that was created or revoked. */
            api_key_name?: string;
            api_key_scopes?: components["schemas"]["ApiKeyScope"][];
            /**
             * @description Set on a refused request: further refusals of the same user and action within the same
             *     interval of this many seconds are not recorded again.
             */
            dedup_interval_seconds?: number;
            /**
             * Format: date-time
             * @description When the account lock ends.
             */
            locked_until?: string;
            /**
             * @description Why the action failed, was refused or locked the account: `invalid_password`,
             *     `account_locked`, `organization_not_active`, `insufficient_role` or `too_many_attempts`.
             */
            reason?: string;
            required_permission?: components["schemas"]["Permission"];
            /** @description The role of a refused user, or the role given to a created user. */
            role?: components["schemas"]["Role"];
            /** @description The sender ID that was approved (`sender_id.approve` only). */
            sender_id?: string;
            /** @description The administration tool that performed a system action (`smsctl`). */
            via?: string;
        };
        /**
         * @description One security event of the organization. Never contains passwords, session tokens, API keys or
         *     their hashes, authorization headers or message text.
         */
        AuditEvent: {
            action: components["schemas"]["AuditAction"];
            actor: components["schemas"]["AuditParty"];
            details: components["schemas"]["AuditDetails"];
            /** Format: uuid */
            id: string;
            /** Format: date-time */
            occurred_at: string;
            outcome: components["schemas"]["AuditOutcome"];
            /** @description The request identifier (`X-Request-ID`), or `null` for actions outside a request. */
            request_id: string | null;
            /**
             * @description The client IP address: the direct peer, or the address reported by a configured trusted
             *     proxy. `null` for actions outside a request.
             */
            source_ip: string | null;
            target: components["schemas"]["AuditParty"] | null;
            /** @description The client's `User-Agent` (control characters removed, at most 256 characters), or `null`. */
            user_agent: string | null;
        };
        AuditEventList: {
            data: components["schemas"]["AuditEvent"][];
            /** @description Cursor for the next page, or `null` when there are no more items. */
            next_cursor: string | null;
        };
        /**
         * @description `success`; `failure` (the credentials or the confirmed password were not accepted); `denied`
         *     (refused by policy, for example a role without the permission or an organization that is not
         *     active); `enforced` (the platform applied a security control: the outcome of
         *     `portal.account_locked`, and of no other action).
         * @enum {string}
         */
        AuditOutcome: "success" | "failure" | "denied" | "enforced";
        /** @description The actor or target of an event. */
        AuditParty: {
            /** @description The portal user's email address (portal users only), or `null`. */
            email: string | null;
            /** @description The resource identifier (an API key's `id`; a UUID otherwise), or `null`. */
            id: string | null;
            /**
             * @description Actor: `portal_user`, `anonymous` (not signed in) or `system` (the platform, or its local
             *     administration tool). Target: `portal_user`, `portal_session` or `api_key`.
             * @enum {string}
             */
            type: "portal_user" | "anonymous" | "system" | "portal_session" | "api_key";
        };
        /**
         * @description Encoding chosen by the platform from the message text.
         * @enum {string}
         */
        Encoding: "gsm7" | "ucs2";
        FieldError: {
            /** @description Machine-readable error code. */
            code: string;
            /** @description Human-readable explanation. */
            detail: string;
            /** @description Name of the invalid query, path or header parameter. */
            parameter?: string;
            /** @description JSON Pointer to the invalid body field, for example `/to`. */
            pointer?: string;
        };
        /** @description A message of the organization. The message text is never returned (data minimization). */
        Message: {
            /** Format: date-time */
            accepted_at: string;
            encoding: components["schemas"]["Encoding"];
            id: components["schemas"]["MessageId"];
            segment_count: number;
            sender_id: components["schemas"]["SenderId"];
            status: components["schemas"]["MessageStatus"];
            status_category: components["schemas"]["StatusCategory"];
            status_reason: components["schemas"]["StatusReason"] | null;
            /** Format: date-time */
            status_updated_at: string;
            to: components["schemas"]["PhoneNumber"];
        };
        MessageCreateRequest: {
            /** @description The message text (1 to 1,600 characters). */
            body: string;
            sender_id: components["schemas"]["SenderId"];
            to: components["schemas"]["PhoneNumber"];
        };
        /** @description A message with its customer-visible status history. */
        MessageDetail: components["schemas"]["Message"] & {
            status_history: components["schemas"]["StatusChange"][];
        };
        MessageEstimate: {
            /** @description Number of characters (Unicode code points) in the text. */
            characters: number;
            encoding: components["schemas"]["Encoding"];
            /**
             * @description The maximum message length in characters (D-144).
             * @example 1600
             */
            max_characters: number;
            /** @description Segments the platform calculates for this text (0 for an empty text). */
            segment_count: number;
        };
        MessageEstimateRequest: {
            /** @description The message text to estimate (may be empty). */
            body: string;
        };
        /**
         * @description Opaque message identifier.
         * @example msg_01jd8x7k3p2m9q4r5s6t7v8w9y
         */
        MessageId: string;
        MessageList: {
            data: components["schemas"]["Message"][];
            /** @description Cursor for the next page, or `null` when there are no more items. */
            next_cursor: string | null;
        };
        /**
         * @description Customer-visible status; the same values and meanings as the public API `MessageStatus`.
         * @enum {string}
         */
        MessageStatus: "accepted" | "provider_accepted" | "submission_uncertain" | "delivered" | "undelivered" | "unknown" | "rejected" | "failed";
        /**
         * @description A permission of the signed-in user. Operations state the one they require (`x-required-permission`).
         * @enum {string}
         */
        Permission: "messages:read" | "messages:send" | "api_keys:manage" | "audit_log:read";
        /**
         * @description E.164 phone number. Only Philippine mobile numbers (`+639…`) are served.
         * @example +639000000001
         */
        PhoneNumber: string;
        /** @description RFC 9457 problem details; `detail` never contains internal error text. */
        Problem: {
            /** @description Human-readable explanation of this occurrence. */
            detail?: string;
            /** @description Field-level validation errors (`validation-error` only). */
            errors?: components["schemas"]["FieldError"][];
            /** @description The request identifier. */
            request_id?: string;
            /** @description HTTP status code. */
            status: number;
            /** @description Short summary of the problem type. */
            title: string;
            /**
             * Format: uri
             * @description Problem type URI. Compare it, not `title` or `detail`.
             */
            type: string;
        };
        /**
         * @description Client portal role (MVP-03). `owner` and `sender` may read and send messages; `viewer` may only
         *     read (G3-3). Only `owner` manages API keys (MVP-03) and reads the security audit log. User
         *     management is not part of this version.
         * @enum {string}
         */
        Role: "owner" | "sender" | "viewer";
        /**
         * @description An approved sender ID of the organization.
         * @example EXAMPLE
         */
        SenderId: string;
        SenderIdList: {
            data: components["schemas"]["SenderId"][];
        };
        Session: {
            /** @description Send this value in the `X-CSRF-Token` header of every state-changing request. */
            csrf_token: string;
            /**
             * Format: date-time
             * @description When the session ends at the latest (absolute lifetime).
             */
            expires_at: string;
            /**
             * Format: date-time
             * @description When the session ends if no further request is made.
             */
            idle_expires_at: string;
            organization: {
                /** @description The organization's display name. */
                name: string;
            };
            permissions: components["schemas"]["Permission"][];
            user: {
                email: string;
                role: components["schemas"]["Role"];
            };
        };
        SignInRequest: {
            /**
             * @description The user's email address (case-insensitive).
             * @example owner@example.invalid
             */
            email: string;
            /** @description The user's password. */
            password: string;
        };
        /**
         * @description Coarse grouping of `status`; the same values and meanings as the public API.
         * @enum {string}
         */
        StatusCategory: "in_progress" | "uncertain" | "final";
        /** @description One change of the customer-visible status, oldest first. */
        StatusChange: {
            /** Format: date-time */
            occurred_at: string;
            status: components["schemas"]["MessageStatus"];
            status_category: components["schemas"]["StatusCategory"];
            status_reason: components["schemas"]["StatusReason"] | null;
        };
        StatusReason: {
            /** @description Machine-readable reason (public API reason codes). */
            code: string;
            /** @description Human-readable explanation in English. */
            description: string;
        };
    };
    responses: {
        /**
         * @description The request is malformed or invalid (`malformed-request`, `validation-error` with field
         *     `errors`, `idempotency-key-missing`, `idempotency-key-invalid`).
         */
        BadRequest: {
            headers: {
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "https://api.example.invalid/problems/validation-error",
                 *       "title": "The request is invalid",
                 *       "status": 400,
                 *       "detail": "One or more fields are invalid.",
                 *       "request_id": "9b2f4c1e7d3a5b60",
                 *       "errors": [
                 *         {
                 *           "pointer": "/to",
                 *           "code": "invalid_phone_number",
                 *           "detail": "Must be an E.164 phone number, for example +639000000001."
                 *         }
                 *       ]
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /**
         * @description Refused: the CSRF token is missing or wrong (`csrf-token-invalid`), the user's role lacks the
         *     permission (`forbidden`), the organization is not active (`account-not-active`), or the current
         *     password given to confirm the action is wrong or the account is temporarily locked
         *     (`reauthentication-failed`).
         */
        Forbidden: {
            headers: {
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "https://api.example.invalid/problems/forbidden",
                 *       "title": "Permission denied",
                 *       "status": 403,
                 *       "detail": "Your role does not allow sending messages.",
                 *       "request_id": "9b2f4c1e7d3a5b63"
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description A request with the same `Idempotency-Key` and body is still being processed. */
        IdempotencyRequestInProgress: {
            headers: {
                "Retry-After": components["headers"]["RetryAfter"];
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "https://api.example.invalid/problems/idempotency-request-in-progress",
                 *       "title": "A request with this Idempotency-Key is in progress",
                 *       "status": 409,
                 *       "detail": "Retry after the indicated delay with the same key and body.",
                 *       "request_id": "9b2f4c1e7d3a5b65"
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description Unexpected server error. The outcome of a submission is unknown; retry with the same `Idempotency-Key`. */
        InternalError: {
            headers: {
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "about:blank",
                 *       "title": "Internal Server Error",
                 *       "status": 500,
                 *       "detail": "An unexpected error occurred.",
                 *       "request_id": "9b2f4c1e7d3a5b69"
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /**
         * @description The email address or password is wrong, or the account is temporarily locked
         *     (`invalid-credentials`). The cause is not distinguished.
         */
        InvalidCredentials: {
            headers: {
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "https://api.example.invalid/problems/invalid-credentials",
                 *       "title": "Sign-in failed",
                 *       "status": 401,
                 *       "detail": "The email address or password is incorrect, or the account is temporarily locked.",
                 *       "request_id": "9b2f4c1e7d3a5b61"
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description No such message or API key in this organization. */
        NotFound: {
            headers: {
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "about:blank",
                 *       "title": "Not Found",
                 *       "status": 404,
                 *       "detail": "The requested resource does not exist.",
                 *       "request_id": "9b2f4c1e7d3a5b64"
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description The request body is too large (`payload-too-large`). */
        PayloadTooLarge: {
            headers: {
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "https://api.example.invalid/problems/payload-too-large",
                 *       "title": "Request body too large",
                 *       "status": 413,
                 *       "detail": "The request body exceeds the allowed size.",
                 *       "request_id": "9b2f4c1e7d3a5b66"
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /**
         * @description The server's password-checking capacity is exhausted (`service-busy`). The password was not
         *     checked, nothing was changed and no failed attempt was counted. Retry after the indicated delay.
         */
        ServiceBusy: {
            headers: {
                "Retry-After": components["headers"]["RetryAfter"];
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "https://api.example.invalid/problems/service-busy",
                 *       "title": "Service busy",
                 *       "status": 503,
                 *       "detail": "The service is busy. Try again in a moment.",
                 *       "request_id": "9b2f4c1e7d3a5b6b"
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /**
         * @description Too many sign-in requests, or too many failed sign-in attempts, from the caller's source
         *     (`too-many-sign-in-attempts`). No credential was checked and no account was affected. Retry
         *     after the indicated delay.
         */
        TooManySignInAttempts: {
            headers: {
                "Retry-After": components["headers"]["RetryAfter"];
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "https://api.example.invalid/problems/too-many-sign-in-attempts",
                 *       "title": "Too many sign-in attempts",
                 *       "status": 429,
                 *       "detail": "Too many sign-in attempts from your network. Wait before trying again.",
                 *       "request_id": "9b2f4c1e7d3a5b6a"
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description No valid session (`unauthenticated`): never signed in, expired, signed out or revoked. */
        Unauthenticated: {
            headers: {
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "https://api.example.invalid/problems/unauthenticated",
                 *       "title": "Authentication required",
                 *       "status": 401,
                 *       "detail": "Sign in to continue.",
                 *       "request_id": "9b2f4c1e7d3a5b62"
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /**
         * @description Valid but not acceptable: `sender-id-not-approved`, `destination-not-supported`,
         *     `message-too-long`, or `idempotency-key-reused`. Nothing was sent.
         */
        UnprocessableMessage: {
            headers: {
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "https://api.example.invalid/problems/sender-id-not-approved",
                 *       "title": "Sender ID not approved",
                 *       "status": 422,
                 *       "detail": "The sender_id is not an approved sender ID for this account.",
                 *       "request_id": "9b2f4c1e7d3a5b68"
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description The request body is not `application/json` (`unsupported-media-type`). */
        UnsupportedMediaType: {
            headers: {
                "X-Request-ID": components["headers"]["XRequestID"];
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "type": "https://api.example.invalid/problems/unsupported-media-type",
                 *       "title": "Unsupported media type",
                 *       "status": 415,
                 *       "detail": "Send the request body as application/json.",
                 *       "request_id": "9b2f4c1e7d3a5b67"
                 *     }
                 */
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
    };
    parameters: {
        /**
         * @description The `csrf_token` of the current session. A missing or wrong token receives `403`
         *     (`csrf-token-invalid`) and nothing is processed.
         */
        CsrfToken: string;
        /** @description Opaque cursor from a previous page's `next_cursor`, used with the same filters. */
        Cursor: string;
        /**
         * @description A unique value per intended send (the portal uses a random UUID), reused unchanged on retries.
         *     Same rules as the public API (ADR-0018): scope is the organization + this operation; a different
         *     body with the same key receives `422` (`idempotency-key-reused`); an identical body is replayed
         *     with `Idempotent-Replayed: true`; keys are kept for seven days.
         * @example 0c9d4bf1-3b52-4f6e-9a57-3f0b6f2f9b1e
         */
        IdempotencyKey: string;
        /** @description Maximum number of items to return. */
        Limit: number;
        /** @description Optional correlation identifier, handled exactly as in the public API. */
        RequestId: string;
    };
    requestBodies: never;
    headers: {
        /** @description `true` when this response replays an earlier response for the same `Idempotency-Key`. */
        IdempotentReplayed: "true";
        /** @description Seconds to wait before retrying. */
        RetryAfter: number;
        /** @description Request identifier for support and troubleshooting. */
        XRequestID: string;
    };
    pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
    listApiKeys: {
        parameters: {
            query?: never;
            header?: {
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The organization's API keys. */
            200: {
                headers: {
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ApiKeyList"];
                };
            };
            401: components["responses"]["Unauthenticated"];
            403: components["responses"]["Forbidden"];
            500: components["responses"]["InternalError"];
        };
    };
    createApiKey: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description The `csrf_token` of the current session. A missing or wrong token receives `403`
                 *     (`csrf-token-invalid`) and nothing is processed.
                 */
                "X-CSRF-Token": components["parameters"]["CsrfToken"];
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ApiKeyCreateRequest"];
            };
        };
        responses: {
            /** @description Created. `key` is shown only in this response. */
            201: {
                headers: {
                    /** @description `no-store`: the response contains a secret and must not be cached. */
                    "Cache-Control"?: "no-store";
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ApiKeyCreated"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthenticated"];
            403: components["responses"]["Forbidden"];
            413: components["responses"]["PayloadTooLarge"];
            415: components["responses"]["UnsupportedMediaType"];
            500: components["responses"]["InternalError"];
            503: components["responses"]["ServiceBusy"];
        };
    };
    revokeApiKey: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description The `csrf_token` of the current session. A missing or wrong token receives `403`
                 *     (`csrf-token-invalid`) and nothing is processed.
                 */
                "X-CSRF-Token": components["parameters"]["CsrfToken"];
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path: {
                /** @description The API key's `id`. */
                api_key_id: components["schemas"]["ApiKeyId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The revoked key. */
            200: {
                headers: {
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ApiKey"];
                };
            };
            401: components["responses"]["Unauthenticated"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            500: components["responses"]["InternalError"];
        };
    };
    listAuditEvents: {
        parameters: {
            query?: {
                /** @description Only events with this action. */
                action?: components["schemas"]["AuditAction"];
                /** @description Opaque cursor from a previous page's `next_cursor`, used with the same filters. */
                cursor?: components["parameters"]["Cursor"];
                /** @description Maximum number of items to return. */
                limit?: components["parameters"]["Limit"];
                /** @description Only events before this time (RFC 3339); must be later than `occurred_from`. */
                occurred_before?: string;
                /** @description Only events at or after this time (RFC 3339). */
                occurred_from?: string;
                /** @description Only events with this outcome. */
                outcome?: components["schemas"]["AuditOutcome"];
            };
            header?: {
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description One page of audit events. */
            200: {
                headers: {
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AuditEventList"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthenticated"];
            403: components["responses"]["Forbidden"];
            500: components["responses"]["InternalError"];
        };
    };
    estimateMessage: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description The `csrf_token` of the current session. A missing or wrong token receives `403`
                 *     (`csrf-token-invalid`) and nothing is processed.
                 */
                "X-CSRF-Token": components["parameters"]["CsrfToken"];
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["MessageEstimateRequest"];
            };
        };
        responses: {
            /** @description The estimate. */
            200: {
                headers: {
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["MessageEstimate"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthenticated"];
            403: components["responses"]["Forbidden"];
            413: components["responses"]["PayloadTooLarge"];
            415: components["responses"]["UnsupportedMediaType"];
            500: components["responses"]["InternalError"];
        };
    };
    listMessages: {
        parameters: {
            query?: {
                /** @description Opaque cursor from a previous page's `next_cursor`, used with the same filters. */
                cursor?: components["parameters"]["Cursor"];
                /** @description Maximum number of items to return. */
                limit?: components["parameters"]["Limit"];
                /** @description Only messages with this status. */
                status?: components["schemas"]["MessageStatus"];
            };
            header?: {
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description One page of messages. */
            200: {
                headers: {
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["MessageList"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthenticated"];
            403: components["responses"]["Forbidden"];
            500: components["responses"]["InternalError"];
        };
    };
    createMessage: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description A unique value per intended send (the portal uses a random UUID), reused unchanged on retries.
                 *     Same rules as the public API (ADR-0018): scope is the organization + this operation; a different
                 *     body with the same key receives `422` (`idempotency-key-reused`); an identical body is replayed
                 *     with `Idempotent-Replayed: true`; keys are kept for seven days.
                 * @example 0c9d4bf1-3b52-4f6e-9a57-3f0b6f2f9b1e
                 */
                "Idempotency-Key": components["parameters"]["IdempotencyKey"];
                /**
                 * @description The `csrf_token` of the current session. A missing or wrong token receives `403`
                 *     (`csrf-token-invalid`) and nothing is processed.
                 */
                "X-CSRF-Token": components["parameters"]["CsrfToken"];
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["MessageCreateRequest"];
            };
        };
        responses: {
            /** @description Accepted (or replayed for the same key and body). Not a delivery report. */
            202: {
                headers: {
                    "Idempotent-Replayed": components["headers"]["IdempotentReplayed"];
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Message"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthenticated"];
            403: components["responses"]["Forbidden"];
            409: components["responses"]["IdempotencyRequestInProgress"];
            413: components["responses"]["PayloadTooLarge"];
            415: components["responses"]["UnsupportedMediaType"];
            422: components["responses"]["UnprocessableMessage"];
            500: components["responses"]["InternalError"];
        };
    };
    getMessage: {
        parameters: {
            query?: never;
            header?: {
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path: {
                /** @description The message `id` returned at acceptance. */
                message_id: components["schemas"]["MessageId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The message. */
            200: {
                headers: {
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["MessageDetail"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthenticated"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            500: components["responses"]["InternalError"];
        };
    };
    listSenderIds: {
        parameters: {
            query?: never;
            header?: {
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The approved sender IDs. */
            200: {
                headers: {
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["SenderIdList"];
                };
            };
            401: components["responses"]["Unauthenticated"];
            403: components["responses"]["Forbidden"];
            500: components["responses"]["InternalError"];
        };
    };
    getSession: {
        parameters: {
            query?: never;
            header?: {
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The current session. */
            200: {
                headers: {
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Session"];
                };
            };
            401: components["responses"]["Unauthenticated"];
            403: components["responses"]["Forbidden"];
            500: components["responses"]["InternalError"];
        };
    };
    signIn: {
        parameters: {
            query?: never;
            header?: {
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["SignInRequest"];
            };
        };
        responses: {
            /** @description Signed in. The response sets the session cookie. */
            200: {
                headers: {
                    /** @description The session cookie `__Host-sms_session` (`HttpOnly; Secure; SameSite=Strict; Path=/`). */
                    "Set-Cookie"?: string;
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Session"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["InvalidCredentials"];
            403: components["responses"]["Forbidden"];
            413: components["responses"]["PayloadTooLarge"];
            415: components["responses"]["UnsupportedMediaType"];
            429: components["responses"]["TooManySignInAttempts"];
            500: components["responses"]["InternalError"];
            503: components["responses"]["ServiceBusy"];
        };
    };
    signOut: {
        parameters: {
            query?: never;
            header: {
                /**
                 * @description The `csrf_token` of the current session. A missing or wrong token receives `403`
                 *     (`csrf-token-invalid`) and nothing is processed.
                 */
                "X-CSRF-Token": components["parameters"]["CsrfToken"];
                /** @description Optional correlation identifier, handled exactly as in the public API. */
                "X-Request-ID"?: components["parameters"]["RequestId"];
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Signed out. The response clears the session cookie. */
            204: {
                headers: {
                    /** @description Expires the session cookie. */
                    "Set-Cookie"?: string;
                    "X-Request-ID": components["headers"]["XRequestID"];
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthenticated"];
            403: components["responses"]["Forbidden"];
            500: components["responses"]["InternalError"];
        };
    };
}
