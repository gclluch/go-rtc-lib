package message

// IMessage defines the interface for messages handled by the RTC library.
type IMessage interface {
	Encode() ([]byte, error)
	Decode([]byte) error
}
