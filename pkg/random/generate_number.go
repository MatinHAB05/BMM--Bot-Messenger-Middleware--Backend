package random

import (
	"crypto/rand"
	"math"
	"math/big"
)

type RandomGeneratorEngine struct{}

func NewRandomGeneratorEngine() *RandomGeneratorEngine {
	return &RandomGeneratorEngine{}
}

func (eng *RandomGeneratorEngine) GeneratePositiveIntRandomNumber(digits int) int64 {
	if digits <= 0 {
		return 0
	}

	min := int64(math.Pow10(digits - 1))
	max := int64(math.Pow10(digits)) - 1
	diff := max - min + 1

	n, err := rand.Int(rand.Reader, big.NewInt(diff))
	if err != nil {
		return 0
	}

	return n.Int64() + min
}

func (eng *RandomGeneratorEngine) GeneratePositiveIntRandomNumberInRange(min, max int64) int64 {
	diff := max - min + 1

	n, err := rand.Int(rand.Reader, big.NewInt(diff))
	if err != nil {
		return 0
	}

	return n.Int64() + min
}

var alphabet = [58]string{
	"1", "2", "3", "4", "5", "6", "7", "8", "9", "A",
	"B", "C", "D", "E", "F", "G", "H", "J", "K", "L",
	"M", "N", "P", "Q", "R", "S", "T", "U", "V", "W",
	"X", "Y", "Z", "a", "b", "c", "d", "e", "f", "g",
	"h", "i", "j", "k", "m", "n", "o", "p", "q", "r",
	"s", "t", "u", "v", "w", "x", "y", "z",
}

func (eng *RandomGeneratorEngine) GenerateRandomBase58String(len int) (string, error) {
	b := make([]byte, len)
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
