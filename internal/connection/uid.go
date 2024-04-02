package connection

import "github.com/google/uuid"

// UIDGeneratorFunc defines the signature for custom UID generation functions.
type UIDGeneratorFunc func() string

// defaultUIDGenerator is the default function used to generate UIDs.
var defaultUIDGenerator UIDGeneratorFunc = func() string {
	return uuid.New().String() // Example using Google's UUID library.
}

// SetUIDGenerator allows users of the library to specify a custom function for UID generation.
func SetUIDGenerator(generator UIDGeneratorFunc) {
	defaultUIDGenerator = generator
}
