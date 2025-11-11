package services

import (
    "testing"
    waTypes "go.mau.fi/whatsmeow/types"
)

func TestStripDevicePart(t *testing.T) {
    cases := []struct{
        in string
        out string
    }{
        {"62812345:12", "62812345"},
        {"62812345", "62812345"},
        {"", ""},
    }

    for _, c := range cases {
        got := stripDevicePart(c.in)
        if got != c.out {
            t.Fatalf("stripDevicePart(%q)=%q; want %q", c.in, got, c.out)
        }
    }
}

func TestNormalizePhone(t *testing.T) {
    cases := []struct{
        in string
        out string
    }{
        {"62812345@s.whatsapp.net", "62812345"},
        {"62812345:12@s.whatsapp.net", "62812345"},
        {"+62 812-345", "62812345"},
        {"  62812345  ", "62812345"},
        {"", ""},
    }

    for _, c := range cases {
        got := normalizePhone(c.in)
        if got != c.out {
            t.Fatalf("normalizePhone(%q)=%q; want %q", c.in, got, c.out)
        }
    }
}

func TestJIDFromNormalizedPhone(t *testing.T) {
    p := normalizePhone("62812345@s.whatsapp.net")
    if p != "62812345" {
        t.Fatalf("normalizePhone -> %q; want 62812345", p)
    }
    jid := waTypes.NewJID(p, waTypes.DefaultUserServer)
    if jid.String() != "62812345@s.whatsapp.net" {
        t.Fatalf("jid.String()=%q; want %q", jid.String(), "62812345@s.whatsapp.net")
    }
}