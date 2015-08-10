// Copyright 2015 The Cockroach Authors.
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
// Author: Tobias Schottdorf (tobias.schottdorf@gmail.com)

// Package securitytest embeds the TLS test certificates.
package securitytest

//go:generate go-bindata -pkg securitytest -mode 0644 -modtime 1400000000 -o ./embedded.go -ignore README.md -prefix ../../resource ../../resource/test_certs/...
//go:generate gofmt -s -w embedded.go
//go:generate goimports -w embedded.go
