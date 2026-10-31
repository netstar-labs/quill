package quill

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// Every field of quill.policy carries a value no other field has, so a field dropped, swapped
// or mis-typed on either side of the mapping changes the result. The offline verbs (prune
// among them) delete on what ReadPolicy returns, and a tool of another version reads the
// same file, so the mapping is a cross-version contract (decisions 29, 68).

// distinctPolicyJSON is quill.policy as writePolicy lays it out for distinctPolicyConfig,
// pasted: a change to a key, its order, its unit or its formatting must be a decision.
const distinctPolicyJSON = `{
  "retention": "36h0m0s",
  "segment_bytes": 1048576,
  "segment_max_age": "5h0m0s",
  "max_bytes": 67108864,
  "min_free_bytes": 3145728,
  "max_records": 12345,
  "max_payload_bytes": 65536,
  "consumer_ttl": "7h0m0s",
  "protected_consumers": [
    "billing",
    "audit"
  ]
}
`

func distinctPolicyConfig() Config {
	return Config{Retention: 36 * time.Hour, SegmentBytes: 1 << 20, SegmentMaxAge: 5 * time.Hour,
		MaxBytes: 64 << 20, MinFreeBytes: 3 << 20, MaxRecords: 12345, MaxPayloadBytes: 1 << 16,
		ConsumerTTL: 7 * time.Hour, ProtectedConsumers: []string{"billing", "audit"}}
}

func TestOpenWritesEveryConfigFieldToItsOwnPolicyKey(t *testing.T) {
	dir := t.TempDir()
	want := distinctPolicyConfig()
	l := openDir(t, dir, func(c *Config) {
		c.Retention, c.SegmentBytes, c.SegmentMaxAge = want.Retention, want.SegmentBytes, want.SegmentMaxAge
		c.MaxBytes, c.MinFreeBytes, c.MaxRecords = want.MaxBytes, want.MinFreeBytes, want.MaxRecords
		c.MaxPayloadBytes, c.ConsumerTTL, c.ProtectedConsumers = want.MaxPayloadBytes, want.ConsumerTTL, want.ProtectedConsumers
	})
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "quill.policy"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != distinctPolicyJSON {
		t.Fatalf("quill.policy:\n%s\nwant\n%s", got, distinctPolicyJSON)
	}
}

func TestReadPolicyTakesEveryPolicyKeyIntoItsOwnConfigField(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "quill.policy"), []byte(distinctPolicyJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	want := distinctPolicyConfig()
	cfg, ok := ReadPolicy(dir)
	if !ok {
		t.Fatal("ReadPolicy refused the policy")
	}
	// Field by field, so a failure names the field that moved.
	for _, f := range []struct {
		name      string
		got, want any
	}{
		{"Retention", cfg.Retention, want.Retention},
		{"SegmentBytes", cfg.SegmentBytes, want.SegmentBytes},
		{"SegmentMaxAge", cfg.SegmentMaxAge, want.SegmentMaxAge},
		{"MaxBytes", cfg.MaxBytes, want.MaxBytes},
		{"MinFreeBytes", cfg.MinFreeBytes, want.MinFreeBytes},
		{"MaxRecords", cfg.MaxRecords, want.MaxRecords},
		{"MaxPayloadBytes", cfg.MaxPayloadBytes, want.MaxPayloadBytes},
		{"ConsumerTTL", cfg.ConsumerTTL, want.ConsumerTTL},
		{"ProtectedConsumers", cfg.ProtectedConsumers, want.ProtectedConsumers},
	} {
		if !reflect.DeepEqual(f.got, f.want) {
			t.Errorf("ReadPolicy %s = %v, want %v", f.name, f.got, f.want)
		}
	}
	// No other field is set: the result is only the policy's nine fields.
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("ReadPolicy = %+v, want %+v", cfg, want)
	}
}
