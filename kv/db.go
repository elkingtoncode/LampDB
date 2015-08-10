// Copyright 2014 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.
//
// Author: Spencer Kimball (spencer.kimball@gmail.com)

package kv

import (
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"github.com/cockroachdb/cockroach/base"
	"github.com/cockroachdb/cockroach/client"
	"github.com/cockroachdb/cockroach/proto"
	"github.com/cockroachdb/cockroach/rpc"
	"github.com/cockroachdb/cockroach/security"
	"github.com/cockroachdb/cockroach/util"
	gogoproto "github.com/gogo/protobuf/proto"
)

const (
	// DBPrefix is the prefix for the key-value database endpoint used
	// to interact with the key-value datastore via HTTP RPC.
	DBPrefix = client.KVDBEndpoint
)

var allowedEncodings = []util.EncodingType{util.JSONEncoding, util.ProtoEncoding}

// verifyRequest checks for illegal inputs in request proto and
// returns an error indicating which, if any, were found.
func verifyRequest(args proto.Request) error {
	switch t := args.(type) {
	case *proto.EndTransactionRequest:
		if t.InternalCommitTrigger != nil {
			return util.Errorf("EndTransaction request from public KV API contains commit trigger: %+v", t.GetInternalCommitTrigger())
		}
	case *proto.BatchRequest:
		for i := range t.Requests {
			method := t.Requests[i].GetValue().(proto.Request).Method()
			if _, ok := allPublicMethods[method.String()]; !ok {
				return util.Errorf("Batch contains a non-public request %s", method.String())
			}
		}
	}
	return nil
}

var allPublicMethods = map[string]proto.Method{
	proto.Get.String():            proto.Get,
	proto.Put.String():            proto.Put,
	proto.ConditionalPut.String(): proto.ConditionalPut,
	proto.Increment.String():      proto.Increment,
	proto.Delete.String():         proto.Delete,
	proto.DeleteRange.String():    proto.DeleteRange,
	proto.Scan.String():           proto.Scan,
	proto.ReverseScan.String():    proto.ReverseScan,
	proto.EndTransaction.String(): proto.EndTransaction,
	proto.Batch.String():          proto.Batch,
	proto.AdminSplit.String():     proto.AdminSplit,
	proto.AdminMerge.String():     proto.AdminMerge,
}

// createArgsAndReply returns allocated request and response pairs
// according to the specified method. Note that createArgsAndReply
// only knows about public methods and explicitly returns nil for
// internal methods. Do not change this behavior without also fixing
// DBServer.ServeHTTP.
func createArgsAndReply(method string) (proto.Request, proto.Response) {
	if m, ok := allPublicMethods[method]; ok {
		switch m {
		case proto.Get:
			return &proto.GetRequest{}, &proto.GetResponse{}
		case proto.Put:
			return &proto.PutRequest{}, &proto.PutResponse{}
		case proto.ConditionalPut:
			return &proto.ConditionalPutRequest{}, &proto.ConditionalPutResponse{}
		case proto.Increment:
			return &proto.IncrementRequest{}, &proto.IncrementResponse{}
		case proto.Delete:
			return &proto.DeleteRequest{}, &proto.DeleteResponse{}
		case proto.DeleteRange:
			return &proto.DeleteRangeRequest{}, &proto.DeleteRangeResponse{}
		case proto.Scan:
			return &proto.ScanRequest{}, &proto.ScanResponse{}
		case proto.ReverseScan:
			return &proto.ReverseScanRequest{}, &proto.ReverseScanResponse{}
		case proto.EndTransaction:
			return &proto.EndTransactionRequest{}, &proto.EndTransactionResponse{}
		case proto.Batch:
			return &proto.BatchRequest{}, &proto.BatchResponse{}
		case proto.AdminSplit:
			return &proto.AdminSplitRequest{}, &proto.AdminSplitResponse{}
		case proto.AdminMerge:
			return &proto.AdminMergeRequest{}, &proto.AdminMergeResponse{}
		}
	}
	return nil, nil
}

// A DBServer provides an HTTP server endpoint serving the key-value API.
// It accepts either JSON or serialized protobuf content types.
type DBServer struct {
	context *base.Context
	sender  client.Sender
}

// NewDBServer allocates and returns a new DBServer.
func NewDBServer(ctx *base.Context, sender client.Sender) *DBServer {
	return &DBServer{context: ctx, sender: sender}
}

// ServeHTTP serves the key-value API by treating the request URL path
// as the method, the request body as the arguments, and sets the
// response body as the method reply. The request body is unmarshalled
// into arguments based on the Content-Type request header. Protobuf
// and JSON-encoded requests are supported. The response body is
// encoded according the the request's Accept header, or if not
// present, in the same format as the request's incoming Content-Type
// header.
func (s *DBServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check TLS settings before anything else.
	authenticationHook, err := security.AuthenticationHook(s.context.Insecure, r.TLS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	method := r.URL.Path
	if !strings.HasPrefix(method, DBPrefix) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	method = strings.TrimPrefix(method, DBPrefix)
	args, reply := createArgsAndReply(method)
	if args == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	// Unmarshal the request.
	reqBody, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := util.UnmarshalRequest(r, reqBody, args, allowedEncodings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verify the request for public API.
	if err := verifyRequest(args); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check request user against client certificate user.
	if err := authenticationHook(args); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Create a call and invoke through sender.
	s.sender.Send(context.TODO(), proto.Call{Args: args, Reply: reply})

	// Marshal the response.
	body, contentType, err := util.MarshalResponse(r, reply, allowedEncodings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(util.ContentTypeHeader, contentType)
	w.Write(body)
}

// RegisterRPC registers the RPC endpoints.
func (s *DBServer) RegisterRPC(rpcServer *rpc.Server) error {
	requests := []proto.Request{
		&proto.GetRequest{},
		&proto.PutRequest{},
		&proto.ConditionalPutRequest{},
		&proto.IncrementRequest{},
		&proto.DeleteRequest{},
		&proto.DeleteRangeRequest{},
		&proto.ScanRequest{},
		&proto.ReverseScanRequest{},
		&proto.EndTransactionRequest{},
		&proto.BatchRequest{},
		&proto.AdminSplitRequest{},
		&proto.AdminMergeRequest{},
	}
	for _, r := range requests {
		if err := rpcServer.Register("Server."+r.Method().String(),
			s.executeCmd, r); err != nil {
			return err
		}
	}
	return nil
}

// executeCmd creates a proto.Call struct and sends it via our local sender.
func (s *DBServer) executeCmd(argsI gogoproto.Message) (gogoproto.Message, error) {
	args := argsI.(proto.Request)
	reply := args.CreateReply()
	s.sender.Send(context.TODO(), proto.Call{Args: args, Reply: reply})
	return reply, nil
}
