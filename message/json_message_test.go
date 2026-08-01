package message

import (
	"encoding/json"
	"testing"
)

type testStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestJSONMessage_SerializeDeserialize_Struct(t *testing.T) {
	originalData := testStruct{
		Name: "John Doe",
		Age:  30,
	}

	msg := NewJSONMessage(originalData)

	// Serialize
	serializedData, err := msg.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Deserialize into a struct of the expected type
	var deserializedData testStruct
	if err := json.Unmarshal(serializedData, &deserializedData); err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	// Directly compare the original and deserialized data
	if originalData != deserializedData {
		t.Errorf("Original and deserialized data are not equal. got = %v, want = %v", deserializedData, originalData)
	}
}

// TestJSONMessage_RoundTrip exercises the IMessage.Deserialize path itself
// (rather than reaching for encoding/json directly), since that's the path
// callers actually use when receiving a message off the wire.
func TestJSONMessage_RoundTrip(t *testing.T) {
	sent := NewJSONMessage(testStruct{Name: "Ann", Age: 5})

	data, err := sent.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	received := &JSONMessage{}
	if err := received.Deserialize(data); err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	// Content is untyped (interface{}), so json.Unmarshal decodes it as a
	// map rather than back into testStruct.
	m, ok := received.Content.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Content to be map[string]interface{}, got %T", received.Content)
	}
	if m["name"] != "Ann" {
		t.Errorf("got name = %v, want Ann", m["name"])
	}
	if m["age"] != float64(5) {
		t.Errorf("got age = %v, want 5", m["age"])
	}
}

func TestJSONMessage_Type(t *testing.T) {
	msg := &JSONMessage{}
	expectedType := "json"

	if messageType := msg.Type(); messageType != expectedType {
		t.Errorf("Type() got = %v, want = %v", messageType, expectedType)
	}
}
