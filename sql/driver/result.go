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
// Author: Peter Mattis (peter@cockroachlabs.com)

package driver

// TODO(pmattis): Currently unused, but will be needed when we support
// LastInsertId.
type result struct {
	lastInsertID int64
	rowsAffected int64
}

func (r *result) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

func (r *result) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}
