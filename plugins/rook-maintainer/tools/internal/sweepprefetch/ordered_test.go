package sweepprefetch

import (
	"reflect"
	"testing"
)

func TestObjectKeepsInsertionOrder(t *testing.T) {
	o := NewObject()
	o.Set("10", 1)
	o.Set("2", 2)
	o.Set("10", 3)
	if want := []string{"10", "2"}; !reflect.DeepEqual(o.Keys(), want) {
		t.Fatalf("Keys() = %v, want %v", o.Keys(), want)
	}
	if v, _ := o.Get("10"); v != 3 {
		t.Errorf("Get(\"10\") = %v, want 3", v)
	}
	if o.Len() != 2 || !o.Has("2") || o.Has("3") {
		t.Errorf("Len/Has disagree with %v", o.Keys())
	}
}

func TestEncodeMatchesTheWrittenFormat(t *testing.T) {
	o := NewObject()
	o.Set("10", map[string]string{"title": "drop the <old> guard & retry"})
	o.Set("2", []string{})
	got, err := Encode(o)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n \"10\": {\n  \"title\": \"drop the <old> guard & retry\"\n },\n \"2\": []\n}"
	if string(got) != want {
		t.Errorf("Encode() =\n%s\nwant\n%s", got, want)
	}
}

func TestDecodeObjectKeepsFileOrder(t *testing.T) {
	o, err := DecodeObject([]byte(`{"30": "Issue", "4": "PullRequest"}`))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"30", "4"}; !reflect.DeepEqual(o.Keys(), want) {
		t.Fatalf("Keys() = %v, want %v", o.Keys(), want)
	}
	o.Set("7", "Issue")
	got, err := Encode(o)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n \"30\": \"Issue\",\n \"4\": \"PullRequest\",\n \"7\": \"Issue\"\n}"
	if string(got) != want {
		t.Errorf("Encode() =\n%s\nwant\n%s", got, want)
	}
}

func TestDecodeObjectRejectsNonObjects(t *testing.T) {
	for _, in := range []string{`[]`, `"x"`, `{`, ``} {
		if _, err := DecodeObject([]byte(in)); err == nil {
			t.Errorf("DecodeObject(%q) succeeded", in)
		}
	}
}
