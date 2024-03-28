package validator

import (
	"regexp"
	"unicode/utf8"
)

var EmailRx = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type Validator struct {
	Errors map[string]string
}

func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

func (v *Validator) Check(ok bool, key, val string) {
	if !ok {
		v.AddError(key, val)
	}
}

func (v *Validator) AddError(key, val string) {
	if _, fnd := v.Errors[key]; !fnd { // do no update if value exists
		v.Errors[key] = val
	}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func In(value string, list ...string) bool {
	for i := range list {
		if value == list[i] {
			return true
		}
	}
	return false
}

func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

func Unique(values []string) bool {
	uniqueValues := make(map[string]bool)
	for _, value := range values {
		uniqueValues[value] = true
	}
	return len(values) == len(uniqueValues)
}

func NotBlank(s string) bool {
	return len(s) > 0
}

func MinChars(s string, n int) bool {
	return utf8.RuneCountInString(s) >= n
}

func MaxChars(field string, n int) bool {
	return utf8.RuneCountInString(field) <= n
}

func Min(n int, min int) bool {
	return n >= min
}

func Max(n int, max int) bool {
	return n <= max
}
