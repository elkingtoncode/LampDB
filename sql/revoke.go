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
// Author: Marc Berhault (marc@cockroachlabs.com)

package sql

import (
	"fmt"

	"github.com/cockroachdb/cockroach/sql/parser"
	"github.com/cockroachdb/cockroach/structured"
)

// Revoke removes privileges from users.
// Current status:
// - Target: DATABASE X only
// - Privileges: ALL, or one or more of READ, WRITE.
// TODO(marc): open questions:
// - should we have root always allowed and not present in the permissions list?
// - should we make users case-insensitive?
// Privileges: WRITE on database.
//   Notes: postgres requires the object owner.
//          mysql requires the "grant option" and the same privileges, and sometimes superuser.
func (p *planner) Revoke(n *parser.Revoke) (planNode, error) {
	descriptor, err := p.getDescriptorFromTargetList(n.Targets)
	if err != nil {
		return nil, err
	}

	if !descriptor.HasPrivilege(p.user, parser.PrivilegeWrite) {
		return nil, fmt.Errorf("user %s does not have %s privilege on %s %s",
			p.user, parser.PrivilegeWrite, descriptor.TypeName(), descriptor.GetName())
	}

	if err := descriptor.Revoke(n); err != nil {
		return nil, err
	}

	// Now update the descriptor.
	// TODO(marc): do this inside a transaction. This will be needed
	// when modifying multiple descriptors in the same op.
	descKey := structured.MakeDescMetadataKey(descriptor.GetID())
	if err := p.db.Put(descKey, descriptor); err != nil {
		return nil, err
	}

	return &valuesNode{}, nil
}
