// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package testcases

// All contains all test cases, grouped by category.
// The category name is used as a prefix in reference image filenames.
var All = map[string][]TestCase{
	"fill":      fillCases,
	"stroke":    strokeCases,
	"curve":     curveCases,
	"dash":      dashCases,
	"precision": precisionCases,
	"complex":   complexCases,
	"subpath":   subpathCases,
	"ctm":       ctmCases,
	"large":     largeCases,
}
