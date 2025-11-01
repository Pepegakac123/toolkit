package toolkit

import "crypto/rand"

const randomStringSource = "abcdefghijklmnouprstvxyzABCDEFGHIJKLMNOUPRSTVXYZ123456789_"

// Tools is the type used to instantiate this module. Every variable of type Tools will have access to all the methods with
// the receiver tools
type Tools struct {
}

// RandomString is a function that generates a random string of the length of n. The random string is using randomStringSource
// as a source for the character
func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSource)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}
	return string(s)
}
