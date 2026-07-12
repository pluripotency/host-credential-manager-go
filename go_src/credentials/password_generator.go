package credentials

import (
	"crypto/rand"
	"math/big"
	"regexp"
)

type StrengthInfo struct {
	Score int    `json:"score"`
	Label string `json:"label"`
	Color string `json:"color"`
}

const (
	lowercaseCharset = "abcdefghijklmnopqrstuvwxyz"
	uppercaseCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberCharset    = "0123456789"
	symbolCharset    = "!@#$%^&*()_+-=[]{}|;:,.<>?"
)

// GeneratePassword generates a random password and evaluates its strength.
func GeneratePassword(length int, lowercase, uppercase, numbers, symbols bool) (string, StrengthInfo) {
	var charset string
	if lowercase {
		charset += lowercaseCharset
	}
	if uppercase {
		charset += uppercaseCharset
	}
	if numbers {
		charset += numberCharset
	}
	if symbols {
		charset += symbolCharset
	}

	if charset == "" || length <= 0 {
		return "Select at least one character type", StrengthInfo{
			Score: 0,
			Label: "None",
			Color: "bg-gray-200 text-gray-700",
		}
	}

	passwordBytes := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "Generation failed", StrengthInfo{
				Score: 0,
				Label: "None",
				Color: "bg-gray-200 text-gray-700",
			}
		}
		passwordBytes[i] = charset[randomIndex.Int64()]
	}

	password := string(passwordBytes)
	strength := CalculateStrength(password)

	return password, strength
}

// CalculateStrength evaluates the password strength score.
func CalculateStrength(password string) StrengthInfo {
	if password == "" {
		return StrengthInfo{
			Score: 0,
			Label: "None",
			Color: "bg-gray-200 text-gray-700",
		}
	}

	score := 0
	length := len(password)
	if length >= 8 {
		score += 1
	}
	if length >= 12 {
		score += 1
	}
	if length >= 16 {
		score += 1
	}

	varieties := 0
	if matched, _ := regexp.MatchString(`[a-z]`, password); matched {
		varieties += 1
	}
	if matched, _ := regexp.MatchString(`[A-Z]`, password); matched {
		varieties += 1
	}
	if matched, _ := regexp.MatchString(`[0-9]`, password); matched {
		varieties += 1
	}
	if matched, _ := regexp.MatchString(`[^a-zA-Z0-9]`, password); matched {
		varieties += 1
	}

	score += varieties / 2

	if score <= 2 {
		return StrengthInfo{Score: 1, Label: "Weak", Color: "bg-red-500 text-white"}
	} else if score == 3 {
		return StrengthInfo{Score: 2, Label: "Medium", Color: "bg-amber-500 text-white"}
	} else if score == 4 {
		return StrengthInfo{Score: 3, Label: "Strong", Color: "bg-green-500 text-white"}
	} else {
		return StrengthInfo{Score: 4, Label: "Excellent", Color: "bg-teal-500 text-white"}
	}
}
