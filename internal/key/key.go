package key

import (
	"math/rand"
	"regexp"
	"strconv"
	"time"
)

const (
	// IDChars represents the character set used to generate cluster IDs.
	// (does not contain 1 and l, to avoid confusion)
	idChars = "023456789abcdefghijkmnopqrstuvwxyz"
	// IDLength represents the number of characters used to create a cluster ID.
	idLength = 5
)

func GenerateID() string {
	for {
		letterRunes := []rune(idChars)
		b := make([]rune, idLength)
		rand.Seed(time.Now().UnixNano())
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}

		id := string(b)

		if _, err := strconv.Atoi(id); err == nil {
			// string is numbers only, which we want to avoid
			continue
		}

		matched, err := regexp.MatchString("^[a-z]+$", id)
		if err == nil && matched == true {
			// strings is letters only, which we also avoid
			continue
		}

		return id
	}
}
