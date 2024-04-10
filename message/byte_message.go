package message

// ByteMessage represents a simple byte-based message.
type ByteMessage struct {
	Data []byte
}

func (m *ByteMessage) Serialize() ([]byte, error) {
	return m.Data, nil
}

func (m *ByteMessage) Deserialize(data []byte) error {
	m.Data = data
	return nil
}

func (m *ByteMessage) Type() string {
	return "byte"
}
