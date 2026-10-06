// Package norm contains types and functions for manipulating
// strings according to Unicode normalization forms.
// This file is an incomplete copy of
// https://github.com/golang/go/blob/master/src/vendor/golang.org/x/text/unicode/norm/iter.go
// created solely for testing purposes.
package norm

// MaxSegmentSize is the maximum size of a byte buffer needed to consider any
// sequence of starter and non-starter runes for the purpose of normalization.
const MaxSegmentSize = 10
