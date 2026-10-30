// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package plugin

import "errors"

// Registration and lookup errors. Returned errors wrap these sentinels, so
// callers inspect them with [errors.Is].
var (
	// ErrUnsupportedPlugin reports a value that implements no plugin contract.
	ErrUnsupportedPlugin = errors.New("unsupported plugin type")
	// ErrAmbiguousPlugin reports a value that implements several plugin contracts.
	ErrAmbiguousPlugin = errors.New("plugin implements several plugin contracts")
	// ErrDuplicateKey reports a tag key already owned by another plugin.
	ErrDuplicateKey = errors.New("duplicate plugin key")
	// ErrDuplicateName reports a plugin name already used by another plugin.
	ErrDuplicateName = errors.New("duplicate plugin name")
	// ErrUnknownPlugin reports a plugin name that is not registered.
	ErrUnknownPlugin = errors.New("unknown plugin")
	// ErrUnknownCheck reports an optcheck key that no plugin provides.
	ErrUnknownCheck = errors.New("unknown optcheck key")
	// ErrPluginDisabled reports a lookup that matched a disabled plugin.
	ErrPluginDisabled = errors.New("plugin is disabled")
	// ErrNotApplicable reports a check declared for an unsupported field type.
	ErrNotApplicable = errors.New("check is not applicable to the field type")
	// ErrInvalidCheckValue reports an optcheck value the check cannot use.
	ErrInvalidCheckValue = errors.New("invalid optcheck value")
)
