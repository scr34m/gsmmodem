package gsmmodem

import (
	"fmt"
	"io"
	"log"
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

// A logging ReadWriteCloser for debugging
type LogReadWriteCloser struct {
	f io.ReadWriteCloser
	l *log.Logger
}

func (self LogReadWriteCloser) Read(b []byte) (int, error) {
	n, err := self.f.Read(b)
	self.l.Printf("Read(%#v) = (%d, %v)\n", string(b[:n]), n, err)
	return n, err
}

func (self LogReadWriteCloser) Write(b []byte) (int, error) {
	n, err := self.f.Write(b)
	self.l.Printf("Write(%#v) = (%d, %v)\n", string(b), n, err)
	return n, err
}

func (self LogReadWriteCloser) Close() error {
	err := self.f.Close()
	self.l.Printf("Close() = %v\n", err)
	return err
}
