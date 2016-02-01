// Copyright 2012 The Stemmer Package Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package stemmer declares Stemmer interface.
package stemmer

type Stemmer interface {
	Stem(s string) string
}
