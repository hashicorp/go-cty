// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MIT

package msgpack

type unknownType struct{}

var unknownVal = unknownType{}

// unknownValBytes is the raw bytes of the msgpack fixext1 value we
// write to represent an unknown value. It's an extension value of
// type zero whose value is irrelevant. Since it's irrelevant, we
// set it to a single byte whose value is also zero, since that's
// the most compact possible representation.
var unknownValBytes = []byte{0xd4, 0, 0}

func (uv unknownType) MarshalMsgpack() ([]byte, error) {
	return unknownValBytes, nil
}
