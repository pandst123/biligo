package biliticket

import (
	"encoding/base64"
	"reflect"
	"testing"
)

func TestGenerateCTokenEncodesNewSemanticLayout(t *testing.T) {
	token := generateCToken(cTokenFields{
		m1:               1,
		touchend:         2,
		m2:               3,
		visibilitychange: 4,
		m3:               5,
		m4:               6,
		openWindow:       9,
		m5:               8,
		timer:            300,
		timediff:         9.8,
		m6:               9,
		m7:               10,
		m8:               11,
		m9:               12,
		beforeunload:     7,
	})
	got := decodeCTokenForTest(t, token)
	want := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 44, 0, 9, 9, 10, 11, 12}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ctoken bytes = %#v, want %#v", got, want)
	}
}

func TestCTokenSessionEncodesPrepareAndCreatePayloads(t *testing.T) {
	session := cTokenSession{
		state: cTokenRuntimeState{
			m1:                 11,
			touchend:           0,
			m2:                 22,
			visibilitychange:   0,
			m3:                 33,
			m4:                 44,
			openWindow:         2,
			m5:                 55,
			m6:                 66,
			m7:                 77,
			m8:                 88,
			m9:                 99,
			beforeunload:       3,
			ticketCollectionMs: 100_000,
			baseTimer:          42,
			createdAtMs:        100_000,
		},
	}

	prepareBytes := decodeCTokenForTest(t, session.generatePrepareAt(100_000))
	if len(prepareBytes) != 16 {
		t.Fatalf("prepare ctoken byte length = %d, want 16", len(prepareBytes))
	}
	wantPrepare := []byte{11, 0, 22, 0, 33, 44, 3, 55, 0, 42, 0, 0, 66, 77, 88, 99}
	if !reflect.DeepEqual(prepareBytes, wantPrepare) {
		t.Fatalf("prepare ctoken bytes = %#v, want %#v", prepareBytes, wantPrepare)
	}

	createBytes := decodeCTokenForTest(t, session.generateCreateAt(103_000))
	if createBytes[0] != 11 || createBytes[2] != 22 || createBytes[4] != 33 || createBytes[5] != 44 || createBytes[7] != 55 {
		t.Fatalf("unexpected create invariant fields: %#v", createBytes)
	}
	if createBytes[1] > 2 {
		t.Fatalf("create touchend = %d, want 0..2", createBytes[1])
	}
	if createBytes[3] > 1 {
		t.Fatalf("create visibilitychange = %d, want 0..1", createBytes[3])
	}
	if createBytes[6] < 10 || createBytes[6] > 50 {
		t.Fatalf("create beforeunload = %d, want 10..50", createBytes[6])
	}
	if createBytes[8] != 0 || createBytes[9] != 45 || createBytes[10] != 0 || createBytes[11] != 3 {
		t.Fatalf("unexpected create timer fields: %#v", createBytes[8:12])
	}
}

func decodeCTokenForTest(t *testing.T, token string) []byte {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("decode ctoken: %v", err)
	}
	if len(raw)%2 != 0 {
		t.Fatalf("decoded ctoken length = %d, want even", len(raw))
	}
	result := make([]byte, 0, len(raw)/2)
	for i := 0; i < len(raw); i += 2 {
		if raw[i+1] != 0 {
			t.Fatalf("decoded ctoken byte %d high byte = %d, want 0", i/2, raw[i+1])
		}
		result = append(result, raw[i])
	}
	return result
}
