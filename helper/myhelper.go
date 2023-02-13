package helper

import (
	"crypto/sha1"
	"encoding/hex"
	"net/mail"
	"strings"
)

func ValidFormatEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func Pass2Hash(plaintext string) string {
	pjg_char := len(plaintext)
	holderstr := []string{}
	for i := 0; i < pjg_char; i++ {
		start := i
		end := i + 1
		strnya := plaintext[start:end]
		h := sha1.New()
		h.Write([]byte(strnya))
		sha1_hash := hex.EncodeToString(h.Sum(nil))
		holderstr = append(holderstr, sha1_hash)
	}
	// fmt.Println(holderstr)
	newstring := strings.Join(holderstr, "")
	// fmt.Println("New Key : " + newstring)
	hash := sha1.New()
	hash.Write([]byte(newstring))
	hashedstring := hex.EncodeToString(hash.Sum(nil))

	return hashedstring
}
