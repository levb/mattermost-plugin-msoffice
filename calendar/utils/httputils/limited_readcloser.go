// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package httputils

import (
	"io"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils"
)

type LimitReadCloser struct {
	ReadCloser io.ReadCloser
	OnClose    func(*LimitReadCloser) error
	TotalRead  utils.ByteSize
	Limit      utils.ByteSize
}

func (r *LimitReadCloser) Read(data []byte) (int, error) {
	if r.Limit >= 0 {
		remain := r.Limit - r.TotalRead
		if remain <= 0 {
			return 0, io.EOF
		}
		if len(data) > int(remain) {
			data = data[0:remain]
		}
	}
	n, err := r.ReadCloser.Read(data)
	r.TotalRead += utils.ByteSize(n)
	return n, err
}

func (r *LimitReadCloser) Close() error {
	if r.OnClose != nil {
		err := r.OnClose(r)
		if err != nil {
			return err
		}
	}
	return r.ReadCloser.Close()
}
