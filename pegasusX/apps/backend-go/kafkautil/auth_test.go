package kafkautil

import (
	"reflect"
	"testing"
)

func TestSplitBrokers(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", []string{}},
		{"localhost:9092", []string{"localhost:9092"}},
		{"b1:9092, b2:9092,b3:9092", []string{"b1:9092", "b2:9092", "b3:9092"}},
		{"  ,  b1:9092 , , b2:9092 , ", []string{"b1:9092", "b2:9092"}},
	}

	for _, tc := range tests {
		got := SplitBrokers(tc.input)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("SplitBrokers(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestDialerAndTransport_Plaintext(t *testing.T) {
	modes := []string{"", AuthModeNone, "PLAINTEXT"}
	for _, m := range modes {
		dialer, err := Dialer(ClientAuth{Mode: m})
		if err != nil {
			t.Fatalf("Dialer failed for mode %q: %v", m, err)
		}
		if dialer.SASLMechanism != nil {
			t.Errorf("expected nil SASL mechanism for mode %q", m)
		}
		if dialer.TLS != nil {
			t.Errorf("expected nil TLS for mode %q", m)
		}

		transport, err := Transport(ClientAuth{Mode: m})
		if err != nil {
			t.Fatalf("Transport failed for mode %q: %v", m, err)
		}
		if transport.SASL != nil {
			t.Errorf("expected nil SASL for transport mode %q", m)
		}
		if transport.TLS != nil {
			t.Errorf("expected nil TLS for transport mode %q", m)
		}
	}
}

func TestDialerAndTransport_InvalidMode(t *testing.T) {
	_, err := Dialer(ClientAuth{Mode: "UNKNOWN_MODE"})
	if err == nil {
		t.Fatalf("expected error for unknown auth mode")
	}

	_, err = Transport(ClientAuth{Mode: "UNKNOWN_MODE"})
	if err == nil {
		t.Fatalf("expected error for unknown auth mode")
	}
}
