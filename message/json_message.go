// pkg/message/jsonmessage.go

package message

import "encoding/json"

// JSONMessage implements the IMessage interface for JSON content.
type JSONMessage struct {
	Content interface{} // Interface to hold any content.
}

func (m *JSONMessage) Serialize() ([]byte, error) {
	return json.Marshal(m.Content)
}

func (m *JSONMessage) Deserialize(data []byte) error {
	return json.Unmarshal(data, &m.Content)
}

func (m *JSONMessage) Type() string {
	return "json"
}

func NewJSONMessage(data interface{}) *JSONMessage {
	return &JSONMessage{
		Content: data,
	}
}
