// Copyright 2016-2017 The grok_exporter Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporter

import (
	configuration "github.com/fstab/grok_exporter/config/v2"
	"gopkg.in/yaml.v2"
	"strings"
	"testing"
)

func TestGrok(t *testing.T) {
	libonig, err := InitOnigurumaLib()
	if err != nil {
		t.Fatal(err)
	}
	patterns := loadPatternDir(t)
	run(t, "compile all patterns", func(t *testing.T) {
		testCompileAllPatterns(t, patterns, libonig)
	})
	run(t, "compile unknown pattern", func(t *testing.T) {
		testCompileUnknownPattern(t, patterns, libonig)
	})
	run(t, "compile invalid regexp", func(t *testing.T) {
		testCompileInvalidRegexp(t, patterns, libonig)
	})
	run(t, "verify capture group", func(t *testing.T) {
		testVerifyCaptureGroup(t, patterns, libonig)
	})
}

func testCompileAllPatterns(t *testing.T, patterns *Patterns, libonig *OnigurumaLib) {
	for pattern := range *patterns {
		_, err := Compile("%{"+pattern+"}", patterns, libonig)
		if err != nil {
			t.Errorf("%v", err.Error())
		}
	}
}

func testCompileUnknownPattern(t *testing.T, patterns *Patterns, libonig *OnigurumaLib) {
	_, err := Compile("%{USER} [a-z] %{SOME_UNKNOWN_PATTERN}.*", patterns, libonig)
	if err == nil || !strings.Contains(err.Error(), "SOME_UNKNOWN_PATTERN") {
		t.Error("expected error message saying which pattern is undefined.")
	}
}

func testCompileInvalidRegexp(t *testing.T, patterns *Patterns, libonig *OnigurumaLib) {
	_, err := Compile("%{USER} [a-z] \\", patterns, libonig) // wrong because regex cannot end with backslash
	if err == nil || !strings.Contains(err.Error(), "%{USER} [a-z] \\") {
		t.Error("expected error message saying which pattern is invalid.")
	}
}

func testVerifyCaptureGroup(t *testing.T, patterns *Patterns, libonig *OnigurumaLib) {
	regex, err := Compile("host %{HOSTNAME:host} user %{USER:user} value %{NUMBER:val}.", patterns, libonig)
	if err != nil {
		t.Fatal(err)
	}
	expectOK(t, regex, `
            name: test
            value: '{{.val}}'
            labels:
              user: '{{.user}}'
              host: '{{.host}}'`)
	expectOK(t, regex, `
            name: test`)
	expectError(t, regex, `
            name: test
            value: '{{.value}}'
            labels:
              user: '{{.user}}'`)
	expectError(t, regex, `
            name: test
            value: '{{.val}}'
            labels:
              user: '{{.user2}}'`)
	regex.Free()
}

func expectOK(t *testing.T, regex *OnigurumaRegexp, config string) {
	expect(t, regex, config, false)
}

func expectError(t *testing.T, regex *OnigurumaRegexp, config string) {
	expect(t, regex, config, true)
}

func expect(t *testing.T, regex *OnigurumaRegexp, config string, isErrorExpected bool) {
	cfg := &configuration.MetricConfig{}
	err := yaml.Unmarshal([]byte(config), cfg)
	if err != nil {
		t.Fatal(err)
	}
	err = cfg.InitTemplates()
	if err != nil {
		t.Fatal(err)
	}
	err = VerifyFieldNames(cfg, regex, nil)
	if isErrorExpected && err == nil {
		t.Fatal("Expected error, but got no error.")
	}
	if !isErrorExpected && err != nil {
		t.Fatal("Expected ok, but got error.")
	}
}
