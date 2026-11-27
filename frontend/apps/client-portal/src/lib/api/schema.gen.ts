// Code generated from contracts/openapi/portal-api-v1.yaml by openapi-typescript. DO NOT EDIT.
// Regenerate with: pnpm --filter @sms/client-portal generate:api
/* eslint-disable */
export interface paths {
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
         *     `403` (`account-not-active`) and no session.
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
        Permission: "messages:read" | "messages:send";
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
         *     read (G3-3). Owner management functions are not part of this version.
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
         *     permission (`forbidden`), or the organization is not active (`account-not-active`).
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
        /** @description No such message for this organization. */
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
            500: components["responses"]["InternalError"];
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
