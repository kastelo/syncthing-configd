package main

import (
	"net/netip"
	"testing"

	"github.com/syncthing/syncthing/lib/protocol"
)

func TestVariableExpansion(t *testing.T) {
	t.Parallel()

	empty := &deviceRejectedData{}
	data := &deviceRejectedData{
		name:    "test",
		device:  protocol.LocalDeviceID,
		address: netip.MustParseAddr("127.0.0.1"),
	}

	cases := []struct {
		input string
		data  *deviceRejectedData
		want  string // empty means error
	}{
		{
			input: "hello",
			data:  empty,
			want:  "hello",
		},
		{
			input: "hello",
			data:  data,
			want:  "hello",
		},
		{
			input: "${device}/${name}/${address}",
			data:  data,
			want:  "7777777-777777N-7777777-777777N-7777777-777777N-7777777-77777Q4/test/127.0.0.1",
		},
		{
			input: "${device}/${name}/${foo}",
			data:  data,
			want:  "", // foo is missing
		},
		{
			input: "${name} test",
			data:  empty,
			want:  "", // name is missing
		},
	}

	for _, c := range cases {
		got, err := replaceVariables(c.input, c.data)
		if c.want != "" {
			if err != nil {
				t.Errorf("replaceVariables(%q, %v) returned error: %v", c.input, c.data, err)
				continue
			}
			if got != c.want {
				t.Errorf("replaceVariables(%q, %v) = %q, want %q", c.input, c.data, got, c.want)
			}
		} else if err == nil {
			t.Errorf("replaceVariables(%q, %v) = %q, want error", c.input, c.data, got)
		}
	}
}
