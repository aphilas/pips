package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed testdata/pip-inspect.golden
var inspectGold []byte

func TestInspect(t *testing.T) {
	t.Run("decodes json string", func(t *testing.T) {
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
