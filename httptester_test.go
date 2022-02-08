package httptester_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/fiam/httptester"
)

type reporter struct {
	*testing.T
	err   error
	fatal error
}

func (r *reporter) Error(args ...interface{}) {
	r.err = args[0].(error)
}

func (r *reporter) Fatal(args ...interface{}) {
	r.fatal = args[0].(error)
}

func TestExpect(t *testing.T) {
	tt := httptester.New(t, http.DefaultServeMux)
	tt.Get("/hello", nil).Expect(200).Contains("hello").Expect("hello world").Match("\\w+ \\w+").
		ExpectHeader("X-Hello", "World").ExpectHeader("X-Number", 42).ContainsHeader("X-Hello", "Wo").
		MatchHeader("X-Hello", "W.*d")
	tt.Post("/does-not-exist", nil).Expect(404)
	echoData := []byte{1, 2, 3, 4, 5, 6}
	tt.Post("/echo", echoData).Expect(echoData)
	tt.Post("/echo", echoData).Expect(bytes.NewReader(echoData))
	tt.Post("/echo", string(echoData)).Expect(echoData)
	tt.Post("/echo", echoData).Expect(string(echoData))
	tt.Post("/echo", bytes.NewReader(echoData)).Expect(echoData)
	tt.Post("/echo", nil).Expect(200).Expect("")
	tt.Post("/echo", nil).Expect(200).Expect(nil)
	form := map[string]interface{}{"foo": 1, "bar": "baz"}
	formExpect := "bar=baz\nfoo=1\n"
	tt.Form("/echo-form", form).Expect(formExpect)
	tt.Get("/echo-form", form).Expect(formExpect)
}

func TestInvalidRegexp(t *testing.T) {
	r := &reporter{T: t}
	tt := httptester.New(r, http.DefaultServeMux)
	tt.Get("/hello", nil).Match("\\Ga+")
	if r.fatal == nil || !strings.Contains(r.fatal.Error(), "error compiling regular expression") {
		t.Errorf("expecting invalid re error, got %s", r.fatal)
	}
}

func TestInvalidWriteHeader(t *testing.T) {
	r := &reporter{T: t}
	tt := httptester.New(r, http.DefaultServeMux)
	tt.Get("/invalid-write-header", nil).Expect(nil)
	if r.err == nil || !strings.Contains(r.err.Error(), "WriteHeader() called with invalid code") {
		t.Errorf("expecting invalid WriteHeader() error, got %s", r.err)
	}
}

func TestMultipleWriteHeader(t *testing.T) {
	r := &reporter{T: t}
	tt := httptester.New(r, http.DefaultServeMux)
	err := tt.Get("/multiple-write-header", nil).Expect(nil).Err()
	if err != r.err {
		t.Errorf("bad error from Err()")
	}
	if r.err == nil || !strings.Contains(r.err.Error(), "WriteHeader() called 2 times") {
		t.Errorf("expecting multiple WriteHeader() error, got %s", r.err)
	}
}

func TestExpectErrors(t *testing.T) {
	r := &reporter{T: t}
	tt := httptester.New(r, http.DefaultServeMux)
	tt.Get("/hello", nil).Expect(400)
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Get("/hello", nil).Contains("nothing")
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Get("/hello", nil).Expect("nothing")
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Get("/hello", nil).ExpectHeader("X-Hello", 13)
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Get("/hello", nil).ExpectHeader("X-Number", 37)
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Get("/hello", nil).Expect(nil)
	if r.err == nil {
		t.Error("expecting an error")
	}
	something := []byte{1, 2, 3, 4, 5, 6}
	tt.Post("/echo", nil).Expect(something)
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Post("/echo", nil).Expect(bytes.NewReader(something))
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Post("/echo", something).Expect(nil)
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Post("/echo", something).Expect(float64(0))
	if r.err == nil {
		t.Error("expecting an error")
	}
	tt.Post("/echo", float64(0)).Expect(float64(0))
	if r.fatal == nil {
		t.Error("expecting a fatal error")
	}
}

func init() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Hello", "World")
		w.Header().Add("X-Number", "42")
		io.WriteString(w, "hello world")
	})
	http.HandleFunc("/empty", func(w http.ResponseWriter, r *http.Request) {})
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			w.Write(data)
		}
	})
	http.HandleFunc("/echo-form", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			panic(err)
		}
		var values url.Values
		if r.Method == "POST" || r.Method == "PUT" {
			values = r.PostForm
		} else {
			values = r.Form
		}
		var keys []string
		for k := range values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "%s=%s\n", k, values.Get(k))
		}
	})
	http.HandleFunc("/invalid-write-header", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(0)
	})
	http.HandleFunc("/multiple-write-header", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.WriteHeader(300)
	})
}
