package generator

import (
	"crypto/md5"
	"encoding/hex"
)

func GenerateAndMatch(targetHash string, maxLength int, alphabet string, partNumber int, totalParts int) []string {
	var found []string
	runes := []rune(alphabet)
	base := len(runes)

	if base == 0 || maxLength <= 0 || totalParts <= 0 || partNumber < 0 || partNumber >= totalParts {
		return found
	}

	gModT := 0

	for length := 1; length <= maxLength; length++ {
		offset := (partNumber - gModT + totalParts) % totalParts

		indices := make([]int, length)
		temp := offset
		for i := length - 1; i >= 0; i-- {
			indices[i] = temp % base
			temp /= base
		}

		if temp == 0 {
			for {
				word := make([]rune, length)
				for i, idx := range indices {
					word[i] = runes[idx]
				}
				strWord := string(word)

				hash := md5.Sum([]byte(strWord))
				hashStr := hex.EncodeToString(hash[:])

				if hashStr == targetHash {
					found = append(found, strWord)
				}

				carry := totalParts
				for pos := length - 1; pos >= 0 && carry > 0; pos-- {
					sum := indices[pos] + carry
					indices[pos] = sum % base
					carry = sum / base
				}

				if carry > 0 {
					break
				}
			}
		}

		bPowerModT := 1
		baseModT := base % totalParts
		for i := 0; i < length; i++ {
			bPowerModT = (bPowerModT * baseModT) % totalParts
		}
		gModT = (gModT + bPowerModT) % totalParts
	}

	return found
}
