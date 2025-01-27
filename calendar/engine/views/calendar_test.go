// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package views

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarkdownToHTMLEntities(t *testing.T) {
	for _, testCase := range []struct {
		description    string
		inputstring    string
		expectedOutput string
	}{
		{
			description:    "with asterisk",
			inputstring:    "**bold text**",
			expectedOutput: "&#42;&#42;bold text&#42;&#42;",
		},
		{
			description:    "normal string",
			inputstring:    "normal string",
			expectedOutput: "normal string",
		},
		{
			description:    "with braces",
			inputstring:    "[square](round)",
			expectedOutput: "&#91;square&#93;&#40;round&#41;",
		},
		{
			description:    "with underscore",
			inputstring:    "text_test",
			expectedOutput: "text&#95;test",
		},
		{
			description:    "withbacktick",
			inputstring:    "`test`",
			expectedOutput: "&#96;test&#96;",
		},
		{
			description:    "with greater and less than",
			inputstring:    "<test>",
			expectedOutput: "&#60;test&#62;",
		},
		{
			description:    "with backslash",
			inputstring:    "test \\ text",
			expectedOutput: "test &#92; text",
		},
		{
			description:    "URL 1",
			inputstring:    "www.example.com",
			expectedOutput: "www&#46;example&#46;com",
		},
		{
			description:    "URL 2",
			inputstring:    "https://example.com",
			expectedOutput: "https&#58;&#47;&#47;example&#46;com",
		},
		{
			description:    "strike through",
			inputstring:    "~~strike~~",
			expectedOutput: "&#126;&#126;strike&#126;&#126;",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			res := MarkdownToHTMLEntities(testCase.inputstring)
			require.EqualValues(t, testCase.expectedOutput, res)
		})
	}
}
