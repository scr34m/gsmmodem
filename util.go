package gsmmodem

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Quote a value
func quoteValue(quote bool, s interface{}) string {
	if !quote {
		return fmt.Sprintf("%s", s.(string))
	}

	switch v := s.(type) {
	case string:
		if v == "?" {
			return v
		}
		return fmt.Sprintf(`"%s"`, v)
	case int, int64:
		return fmt.Sprint(v)
	default:
		panic(fmt.Sprintf("Unsupported argument type: %T", v))
	}
}

// Join list of values with out quote
func join(quote bool, args []interface{}) string {
	ret := make([]string, len(args))
	for i, arg := range args {
		ret[i] = quoteValue(quote, arg)
	}
	return strings.Join(ret, ",")
}

// Unquote a string to a value (string or int)
func unquote(s string) interface{} {
	if strings.HasPrefix(s, `"`) {
		return strings.Trim(s, `"`)
	}
	if i, err := strconv.Atoi(s); err == nil {
		// number
		return i
	}
	return s
}

var RegexQuote = regexp.MustCompile(`"[^"]*"|[^,]*`)

// Unquote a parameter list to values
func unquotes(s string) []interface{} {
	vs := RegexQuote.FindAllString(s, -1)
	args := make([]interface{}, len(vs))
	for i, v := range vs {
		args[i] = unquote(v)
	}
	return args
}
