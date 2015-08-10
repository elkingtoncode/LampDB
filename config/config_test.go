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
// Author: Tamir Duberstein (tamird@gmail.com)

package config

import "testing"

func TestPermConfig(t *testing.T) {
	p := &PermConfig{
		Read:  []string{"foo", "bar", "baz"},
		Write: []string{"foo", "baz"},
	}
	for _, u := range p.Read {
		if !p.CanRead(u) {
			t.Errorf("expected read permission for %q", u)
		}
	}
	if p.CanRead("bad") {
		t.Errorf("unexpected read access for user \"bad\"")
	}
	for _, u := range p.Write {
		if !p.CanWrite(u) {
			t.Errorf("expected read permission for %q", u)
		}
	}
	if p.CanWrite("bar") {
		t.Errorf("unexpected read access for user \"bar\"")
	}
}
