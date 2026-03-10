// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MIT

// Package gocty deals with converting between cty Values and native go
// values.
//
// It operates under a similar principle to the encoding/json and
// encoding/xml packages in the standard library, using reflection to
// populate native Go data structures from cty values and vice-versa.
package gocty
