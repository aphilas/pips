package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"os"
	"reflect"
	"testing"
)

//go:embed testdata/pip-inspect.golden
var inspectGold []byte

func TestPipInspect(t *testing.T) {
	t.Run("decodes pip inspect output", func(t *testing.T) {
		pi := new(PipInspection)
		dec := json.NewDecoder(bytes.NewReader(inspectGold))
		if err := dec.Decode(pi); err != nil {
			t.Error(err)
		}

		const want = "starlette"
		got := pi.Installed[0].Metadata.Name

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})
}

func TestSavePkgs(t *testing.T) {
	pi := new(PipInspection)
	dec := json.NewDecoder(bytes.NewReader(inspectGold))
	if err := dec.Decode(pi); err != nil {
		t.Error(err)
	}

	t.Run("saves installed package", func(t *testing.T) {
		f, err := os.CreateTemp("", "_pips_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())

		err = savePkgs(f.Name(), pi, map[string]string{"starlette": ""})
		if err != nil {
			t.Fatal(err)
		}

		buf, err := io.ReadAll(f)
		if err != nil {
			t.Fatal(err)
		}

		got := string(buf)
		want := "starlette==0.25.0\n"
		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("saves installed package with extras", func(t *testing.T) {
		f, err := os.CreateTemp("", "_pips_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())

		err = savePkgs(f.Name(), pi, map[string]string{"starlette": "[full]"})
		if err != nil {
			t.Fatal(err)
		}

		buf, err := io.ReadAll(f)
		if err != nil {
			t.Fatal(err)
		}

		got := string(buf)
		want := "starlette[full]==0.25.0\n"
		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("errors if package is not installed", func(t *testing.T) {
		f, err := os.CreateTemp("", "_pips_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())

		err = savePkgs(f.Name(), pi, map[string]string{"ezekiel": ""})
		if err == nil {
			t.Error("wanted error, got nil")
		}
	})
}

func TestParseSpecifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"parses plain name", "loki", []string{"loki", "", ""}},
		{"parses name with extras", "loki[doc,security]", []string{"loki", "[doc,security]", ""}},
		{"parses name with version", "loki==0.0.1", []string{"loki", "", "0.0.1"}},
		{"parses name with extras and version", "loki[doc,security]==0.0.1", []string{"loki", "[doc,security]", "0.0.1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, e, v := parseSpecifier(tt.input)
			got := []string{n, e, v}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v want %#v", got, tt.want)
			}
		})
	}
}
