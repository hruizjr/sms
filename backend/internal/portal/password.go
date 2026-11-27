package portal

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Password hashing (owner decision G3-2, decision log §20): Argon2id with the OWASP Password
// Storage Cheat Sheet's first parameter set — 19 MiB of memory, 2 iterations, 1 degree of
// parallelism — a 16-byte random salt and a 32-byte hash, stored in the PHC string format
//
//	$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
//
// (unpadded standard base64). Verification reads the parameters from the stored string, so they can
// be raised later without invalidating existing hashes.
const (
	argonMemoryKiB   = 19 * 1024
	argonIterations  = 2
	argonParallelism = 1
	argonSaltBytes   = 16
	argonKeyBytes    = 32

	// Upper bounds accepted when verifying a stored hash, so a corrupted or tampered value cannot
	// make one verification use unbounded memory or time.
	maxArgonMemoryKiB   = 256 * 1024
	maxArgonIterations  = 10
	maxArgonParallelism = 16
)

// Password rules (owner decision G3-3): at least 12 characters. The upper bound only limits the
// work done per request.
const (
	MinPasswordCharacters = 12
	MaxPasswordBytes      = 1024
)

// ErrWeakPassword: the password does not meet the password rules.
var ErrWeakPassword = fmt.Errorf("the password must be %d to %d characters", MinPasswordCharacters, MaxPasswordBytes)

var errMalformedHash = errors.New("malformed password hash")

// hasher computes and verifies Argon2id hashes, with a bound on how many run at once: each one
// holds argonMemoryKiB of memory (AGENTS.md §7.4 bounded concurrency).
type hasher struct {
	slots chan struct{}
}

func newHasher(concurrency int) *hasher {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &hasher{slots: make(chan struct{}, concurrency)}
}

func (h *hasher) acquire(ctx context.Context) error {
	select {
	case h.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *hasher) release() { <-h.slots }

// validatePassword applies the password rules.
func validatePassword(password string) error {
	if len(password) > MaxPasswordBytes || utf8.RuneCountInString(password) < MinPasswordCharacters || !utf8.ValidString(password) {
		return ErrWeakPassword
	}
	return nil
}

// hash returns the PHC string of password with a new random salt.
func (h *hasher) hash(ctx context.Context, password string) (string, error) {
	salt := make([]byte, argonSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer h.release()
	key := argon2.IDKey([]byte(password), salt, argonIterations, argonMemoryKiB, argonParallelism, argonKeyBytes)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argonMemoryKiB, argonIterations, argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// verify reports whether password matches the PHC string encoded, in constant time with respect to
// the hash value.
func (h *hasher) verify(ctx context.Context, password, encoded string) (bool, error) {
	p, err := parseHash(encoded)
	if err != nil {
		return false, err
	}
	if err := h.acquire(ctx); err != nil {
		return false, err
	}
	defer h.release()
	key := argon2.IDKey([]byte(password), p.salt, p.iterations, p.memory, p.parallelism, uint32(len(p.key))) //nolint:gosec // len(p.key) is bounded by parseHash
	return subtle.ConstantTimeCompare(key, p.key) == 1, nil
}

type argonParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	salt, key   []byte
}

// parseHash parses "$argon2id$v=19$m=..,t=..,p=..$salt$hash" and checks the parameter bounds.
func parseHash(encoded string) (argonParams, error) {
	var p argonParams
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v="+strconv.Itoa(argon2.Version) {
		return p, errMalformedHash
	}
	for _, kv := range strings.Split(parts[3], ",") {
		name, value, ok := strings.Cut(kv, "=")
		if !ok {
			return p, errMalformedHash
		}
		n, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return p, errMalformedHash
		}
		switch name {
		case "m":
			p.memory = uint32(n)
		case "t":
			p.iterations = uint32(n)
		case "p":
			if n > maxArgonParallelism {
				return p, errMalformedHash
			}
			p.parallelism = uint8(n)
		default:
			return p, errMalformedHash
		}
	}
	var err1, err2 error
	p.salt, err1 = base64.RawStdEncoding.DecodeString(parts[4])
	p.key, err2 = base64.RawStdEncoding.DecodeString(parts[5])
	switch {
	case err1 != nil, err2 != nil,
		p.memory < 8*uint32(max(p.parallelism, 1)), p.memory > maxArgonMemoryKiB,
		p.iterations < 1, p.iterations > maxArgonIterations,
		p.parallelism < 1,
		len(p.salt) < 8, len(p.salt) > 64,
		len(p.key) < 16, len(p.key) > 64:
		return p, errMalformedHash
	}
	return p, nil
}
