// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		str     string
		want    ByteSize
		wantErr bool
	}{
		// Happy path
		{"1234567890123456789", 1234567890123456789, false},
		{",,,,1,2,3,4,5,6,,,7,8,9,0,1,2,3,4,5,6,7,8,9,,", 1234567890123456789, false},
		{"1,234,567,890,123,456,789", 1234567890123456789, false},
		{"1234567890123456789b", 1234567890123456789, false},
		{"4", 4, false},
		{"4B", 4, false},
		{"1234b", 1234, false},
		{"1234.0b", 1234, false},
		{"1Kb", 1024, false},
		{"12kb", 12 * 1024, false},
		{"1.23Kb", 1259, false},
		{"1234.0kb", 1263616, false},
		{"1234Mb", 1293942784, false},
		{"1.234Mb", 1293942, false},
		{"1234Gb", 1324997410816, false},
		{"1.234Gb", 1324997410, false},
		{"1234Tb", 1356797348675584, false},
		{"1.234tb", 1356797348675, false},

		// Errors
		{"AA", 0, true},
		{"1..00kb", 0, true},
		{" 1.00b", 0, true},
		{"1AA", 0, true},
		{"1.0AA", 0, true},
		{"1/2", 0, true},
		{"0x10", 0, true},
		{"88888888888888888888888888888888888888888888888888888888888888888888888888888888888888888888", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			got, err := ParseByteSize(tt.str)
			if tt.wantErr {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestByteSizeString(t *testing.T) {
	tests := []struct {
		want string
		n    ByteSize
	}{
		{"0", 0},
		{"1b", 1},
		{"999b", 999},
		{"1,000b", 1000},
		{"1,023b", 1023},
		{"1Kb", 1024},
		{"12.1Kb", 12345},
		{"12.5Kb", 12851}, // 12.54980
		{"12.6Kb", 12852}, // 12.55078
		{"120.6Kb", 123456},
		{"1.2Mb", 1234567},
		{"11.8Mb", 12345678},
		{"117.7Mb", 123456789},
		{"1.1Gb", 1234567890},
		{"11.5Gb", 12345678900},
		{"115Gb", 123456789000},
		{"1.1Tb", 1234567890000},
		{"11.2Tb", 12345678900000},
		{"112.3Tb", 123456789000000},
		{"1,122.8Tb", 1234567890000000},
		{"11,228.3Tb", 12345678900000000},
		{"112,283.3Tb", 123456789000000000},
		{"n/a", 1234567890000000000},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.n), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.n.String())
		})
	}
}
