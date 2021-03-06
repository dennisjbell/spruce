package main

import (
	"bufio"
	"fmt"
	"github.com/smallfish/simpleyaml"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v2"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestVault(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	ToYAML := func(tree map[interface{}]interface{}) string {
		y, err := yaml.Marshal(tree)
		So(err, ShouldBeNil)
		return string(y)
	}
	ReYAML := func(s string) string {
		return ToYAML(YAML(s))
	}
	RunTests := func(src string) {
		var test, input, output string
		var current *string
		testPat := regexp.MustCompile(`^##+\s+(.*)\s*$`)

		convey := func() {
			if test != "" {
				Convey(test, func() {
					ev := &Evaluator{Tree: YAML(input)}
					err := ev.RunPhase(EvalPhase)
					So(err, ShouldBeNil)
					So(ToYAML(ev.Tree), ShouldEqual, ReYAML(output))
				})
			}
		}

		s := bufio.NewScanner(strings.NewReader(src))
		for s.Scan() {
			if testPat.MatchString(s.Text()) {
				m := testPat.FindStringSubmatch(s.Text())
				convey()
				test, input, output = m[1], "", ""
				continue
			}

			if s.Text() == "---" {
				if input == "" {
					current = &input
				} else {
					current = &output
				}
				continue
			}

			if current != nil {
				*current = *current + s.Text() + "\n"
			}
		}
		convey()
	}

	RunErrorTests := func(src string) {
		var test, input, errors string
		var current *string
		testPat := regexp.MustCompile(`^##+\s+(.*)\s*$`)

		convey := func() {
			if test != "" {
				Convey(test, func() {
					ev := &Evaluator{Tree: YAML(input)}
					err := ev.RunPhase(EvalPhase)
					So(err, ShouldNotBeNil)
					So(strings.Trim(err.Error(), " \t"), ShouldEqual, errors)
				})
			}
		}

		s := bufio.NewScanner(strings.NewReader(src))
		for s.Scan() {
			if testPat.MatchString(s.Text()) {
				m := testPat.FindStringSubmatch(s.Text())
				convey()
				test, input, errors = m[1], "", ""
				continue
			}

			if s.Text() == "---" {
				if input == "" {
					current = &input
				} else {
					current = &errors
				}
				continue
			}

			if current != nil {
				*current = *current + s.Text() + "\n"
			}
		}
		convey()
	}

	Convey("Disconnected Vault", t, func() {
		os.Setenv("VAULT_ADDR", "")
		os.Setenv("VAULT_TOKEN", "")

		RunTests(`
#############################################  emits REDACTED for no VAULT_* ENV
---
secret: (( vault "secret/hand:shake" ))

---
secret: REDACTED
`)

		os.Setenv("VAULT_ADDR", "")
		os.Setenv("VAULT_TOKEN", "something")
		RunTests(`
#############################################  emits REDACTED for no VAULT_ADDR
---
secret: (( vault "secret/hand:shake" ))

---
secret: REDACTED
`)
	})

	Convey("Connected Vault", t, func() {
		mock := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-Vault-Token") != "sekrit-toekin" {
						w.WriteHeader(403)
						fmt.Fprintf(w, `{"errors":["missing client token"]}`)
						return
					}
					switch r.URL.Path {
					case "/v1/secret/hand":
						w.WriteHeader(200)
						fmt.Fprintf(w, `{"data":{"shake":"knock, knock"}}`)
					case "/v1/secret/admin":
						w.WriteHeader(200)
						fmt.Fprintf(w, `{"data":{"username":"admin","password":"x12345"}}`)
					case "/v1/secret/key":
						w.WriteHeader(200)
						fmt.Fprintf(w, `{"data":{"test":"testing"}}`)
					case "/v1/secret/malformed":
						w.WriteHeader(200)
						fmt.Fprintf(w, `wait, this isn't JSON`)
					case "/v1/secret/structure":
						w.WriteHeader(200)
						fmt.Fprintf(w, `{"data":{"data":[1,2,3]}}`)
					default:
						w.WriteHeader(404)
						fmt.Fprintf(w, `{"errors":[]}`)
					}
				},
			),
		)
		defer mock.Close()

		os.Setenv("VAULT_ADDR", mock.URL)
		os.Setenv("VAULT_TOKEN", "sekrit-toekin")
		RunTests(`
################################################  emits sensitive credentials
---
meta: secret/key:test
secret: (( vault "secret/hand:shake" ))
username: (( vault "secret/admin:username" ))
password: (( vault "secret/admin:password" ))
key: (( vault $.meta ))

---
meta: secret/key:test
secret: knock, knock
username: admin
password: x12345
key: testing
`)

		os.Setenv("VAULT_ADDR", mock.URL)
		oldhome := os.Getenv("HOME")
		os.Setenv("HOME", "assets/home/auth")
		os.Setenv("VAULT_TOKEN", "")
		RunTests(`
##########################  retrieves token transparently from ~/.vault-token
---
secret: (( vault "secret/hand:shake" ))

---
secret: knock, knock
`)

		RunErrorTests(`
#########################################  fails when missing its argument
---
secret: (( vault ))

---
1 error(s) detected:
 - $.secret: vault operator requires exactly one argument

#########################################  fails on non-existent reference
---
meta: {}
secret: (( vault $.meta.key ))

---
1 error(s) detected:
 - $.secret: Unable to resolve ` + "`" + `meta.key` + "`" + `: ` + "`" + `$.meta.key` + "`" + ` could not be found in the YAML datastructure

####################################################  fails on map reference
---
meta:
  key: secret/hand:shake
secret: (( vault $.meta ))

---
1 error(s) detected:
 - $.secret: tried to look up $.meta, which is not a string scalar

##################################################  fails on list reference
---
meta:
  - first
secret: (( vault $.meta ))

---
1 error(s) detected:
 - $.secret: tried to look up $.meta, which is not a string scalar

#########################################  fails on non-existent credentials
---
secret: (( vault "secret/e:noent" ))

---
1 error(s) detected:
 - $.secret: secret secret/e:noent not found

##############################################  fails on non-string argument
---
secret: (( vault 42 ))

---
1 error(s) detected:
 - $.secret: invalid argument 42; must be in the form path/to/secret:key

#################################################  fails on non-JSON response
---
secret: (( vault "secret/malformed:key" ))

---
1 error(s) detected:
 - $.secret: bad JSON response received from Vault: "wait, this isn't JSON"

#################################################  fails on non-string data
---
secret: (( vault "secret/structure:data" ))

---
1 error(s) detected:
 - $.secret: secret secret/structure:data is not a string

`)

		os.Setenv("VAULT_TOKEN", "incorrect")
		RunErrorTests(`
#####################################################  fails on a bad token
---
secret: (( vault "secret/hand:shake" ))

---
1 error(s) detected:
 - $.secret: failed to retrieve secret/hand:shake from Vault (` + os.Getenv("VAULT_ADDR") + `): missing client token

`)

		oldhome = os.Getenv("HOME")
		os.Setenv("HOME", "assets/home/unauth")
		os.Setenv("VAULT_TOKEN", "")
		RunErrorTests(`
################################################  fails on a missing token
---
secret: (( vault "secret/hand:shake" ))

---
1 error(s) detected:
 - $.secret: VAULT_ADDR specified, but no VAULT_TOKEN or ~/.vault-token found

`)
		os.Setenv("HOME", oldhome)
	})
}
