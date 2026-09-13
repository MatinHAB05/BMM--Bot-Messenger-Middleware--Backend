package otp

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// CodeGenerator produces random OTP secrets. It's a narrow seam --
// rather than each Strategy calling crypto/rand directly -- so tests can
// substitute a deterministic generator, and so this module doesn't
// hardcode a dependency on any particular random-string package. If a
// codebase already has one (e.g. an existing pkg/random engine), adapt
// it to this interface instead of using DefaultCodeGenerator.
type CodeGenerator interface {
	// Numeric returns a decimal digit string of the given length, e.g.
	// "048213" for length 6. Suitable for codes a person types by hand.
	Numeric(length int) (string, error)

	RandBase58String(n int) (string, error)

	// URLSafe returns a URL-safe random token built from length
	// characters of a base62+ alphabet. Suitable for magic-link codes,
	// which are never typed by hand and benefit from more entropy per
	// character than a numeric code can practically offer.
	URLSafe(length int) (string, error)
}

// DefaultCodeGenerator is a crypto/rand-backed CodeGenerator with no
// dependencies beyond the standard library.
type DefaultCodeGenerator struct{}

// NewDefaultCodeGenerator builds the standard-library-only CodeGenerator.
func NewDefaultCodeGenerator() CodeGenerator { return &DefaultCodeGenerator{} }

func (DefaultCodeGenerator) Numeric(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("otp: numeric code length must be positive, got %d", length)
	}

	digits := make([]byte, length)
	for i := range digits {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("otp: generate numeric code: %w", err)
		}
		digits[i] = byte('0') + byte(n.Int64())
	}
	return string(digits), nil
}

const urlSafeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

func (DefaultCodeGenerator) URLSafe(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("otp: url-safe token length must be positive, got %d", length)
	}

	out := make([]byte, length)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(urlSafeAlphabet))))
		if err != nil {
			return "", fmt.Errorf("otp: generate url-safe token: %w", err)
		}
		out[i] = urlSafeAlphabet[n.Int64()]
	}
	return string(out), nil
}

var alphabet = [58]string{
	"1", "2", "3", "4", "5", "6", "7", "8", "9", "A",
	"B", "C", "D", "E", "F", "G", "H", "J", "K", "L",
	"M", "N", "P", "Q", "R", "S", "T", "U", "V", "W",
	"X", "Y", "Z", "a", "b", "c", "d", "e", "f", "g",
	"h", "i", "j", "k", "m", "n", "o", "p", "q", "r",
	"s", "t", "u", "v", "w", "x", "y", "z",
}

func (DefaultCodeGenerator) RandBase58String(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	var s string
	for _, j := range b {
		idx := int(j) % 58
		s = s + alphabet[idx]
	}
	return s, nil
}
