// Copyright (c) 2021  Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package rpc

import (
	"fmt"

	athrift "github.com/apache/thrift/lib/go/thrift"
	"github.com/uber/tchannel-go/thrift"
)

type BinaryFetchTaggedHandler interface {
	FetchTagged(ctx thrift.Context, req *FetchTaggedRequest, protocol athrift.TBinaryProtocol) (*FetchTaggedResult_, error)
}

func (s *tchanNodeServer) handleFetchTagged(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req NodeFetchTaggedArgs
	var res NodeFetchTaggedResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	res.Write()

	r, err :=
		s.handler.FetchTagged(ctx, req.Req, protocol)

	if err != nil {
		switch v := err.(type) {
		case *Error:
			if v == nil {
				return false, nil, fmt.Errorf("Handler for err returned non-nil error type *Error but nil value")
			}
			res.Err = v
		default:
			return false, nil, err
		}
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}