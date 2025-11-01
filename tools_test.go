package toolkit

import (
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s1, s2 := testTools.RandomString(10), testTools.RandomString(10)
	if len(s1) != 10 || len(s2) != 10 {
		t.Error("the length of the random string is wrong")
	}
	if s1 == s2 {
		t.Error("the random strings are not random ")
	}
}
