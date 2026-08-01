package message

import (
	"reflect"
	"testing"
)

func TestByteMessage_SerializeDeserialize(t *testing.T) {
	originalMessage := &ByteMessage{
		Data: []byte("hello"),
	}

	serializedData, err := originalMessage.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	var deserializedMessage ByteMessage
	if err := deserializedMessage.Deserialize(serializedData); err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if !reflect.DeepEqual(originalMessage, &deserializedMessage) {
		t.Errorf("Original and deserialized messages are not equal. got = %v, want = %v", &deserializedMessage, originalMessage)
	}
}
