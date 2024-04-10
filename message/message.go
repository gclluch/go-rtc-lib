package message

// IMessage represents a generic message interface.
type IMessage interface {
	Serialize() ([]byte, error) // Convert the message to a byte slice for sending.
	Deserialize([]byte) error   // Populate the message fields from a byte slice.
	Type() string               // Return the message type.
}
