//go:build portico_conformance_fixture

package main

import (
    "encoding/json"
    "strings"
    "testing"
)

func TestLabStateRejectsForgeryAndReplayShape(t *testing.T) {
    key := []byte(strings.Repeat("k", 32))
    token := labSignedState(key, "round-1")
    phase, valid := labVerifyState(key, token)
    if !valid || phase != "round-1" {
        t.Fatal("authentic fixture state rejected")
    }
    for _, invalid := range []string{
        "", "round-1", token+"0", labSignedState([]byte(strings.Repeat("z", 32)), "round-1"),
        "round-2."+strings.Repeat("0", 64),
    } {
        if _, ok := labVerifyState(key, invalid); ok {
            t.Fatalf("forged fixture requestState accepted: %q", invalid)
        }
    }
}

func TestLabInputRequiredWireResult(t *testing.T) {
    r, err := labInputRequired("some-opaque-state", "step1", "name")
    if err != nil {
        t.Fatal(err)
    }
    if !r.NeedsInput() {
        t.Fatal("fixture did not produce a real SDK InputRequiredResult")
    }
    wire, err := json.Marshal(r)
    if err != nil {
        t.Fatal(err)
    }
    var data map[string]any
    if err := json.Unmarshal(wire, &data); err != nil {
        t.Fatal(err)
    }
    if data["resultType"] != "input_required" || data["requestState"] != "some-opaque-state" {
        t.Fatalf("incorrect input_required wire envelope: %s", wire)
    }
    req, ok := data["inputRequests"].(map[string]any)
    if !ok || req["step1"] == nil {
        t.Fatalf("missing request binding: %s", wire)
    }
}
