// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package winevent

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/tianlin/go-windows-eventlog/pkg/sys"
)

// AddOptional adds a key and value to the given map if the value is not the
// zero value for the type of v. It is safe to call the function with a nil map.
func AddOptional(m map[string]interface{}, key string, v interface{}) {
	if m != nil && !isZero(v) {
		setNestedField(m, key, v)
	}
}

// AddPairs adds a new dictionary to the given map. The key/value pairs are
// added to the new dictionary. If any keys are duplicates, the first key/value
// pair is added and the remaining duplicates are dropped. Pair keys are not
// expanded into dotted paths.
//
// The new dictionary is added to the given map and it is also returned for
// convenience purposes.
func AddPairs(m map[string]interface{}, key string, pairs []KeyValue) map[string]interface{} {
	if len(pairs) == 0 {
		return nil
	}

	// Explicitly use the unnamed type to prevent accidental use
	// of map path look-up methods.
	h := make(map[string]interface{}, len(pairs))

	for i, kv := range pairs {
		// Ignore empty values.
		if kv.Value == "" {
			continue
		}

		// If the key name is empty or if it the default of "Data" then
		// assign a generic name of paramN.
		k := kv.Key
		if k == "" || k == "Data" {
			k = fmt.Sprintf("param%d", i+1)
		}

		// Do not overwrite.
		_, exists := h[k]
		if exists {
			// Note: debug logging removed to keep package independent
			// Original: debugf("Dropping key/value (k=%s, v=%s) pair because key already "+
			//	"exists. event=%+v", k, kv.Value, m)
		} else {
			h[k] = sys.RemoveWindowsLineEndings(kv.Value)
		}
	}

	if len(h) == 0 {
		return nil
	}

	setNestedField(m, key, h)

	return h
}

// isZero return true if the given value is the zero value for its type.
func isZero(i interface{}) bool {
	switch i := i.(type) {
	case nil:
		return true
	case time.Time:
		return false
	default:
		return reflect.ValueOf(i).IsZero()
	}
}

// setNestedField sets a value in a nested map using dot notation.
// For example: setNestedField(m, "event.code", 123) sets m["event"]["code"] = 123
func setNestedField(m map[string]interface{}, key string, value interface{}) {
	keys := strings.Split(key, ".")
	current := m

	for i := 0; i < len(keys)-1; i++ {
		if _, exists := current[keys[i]]; !exists {
			current[keys[i]] = make(map[string]interface{})
		}
		var ok bool
		current, ok = current[keys[i]].(map[string]interface{})
		if !ok {
			// Path exists but is not a map, cannot set nested field
			return
		}
	}

	current[keys[len(keys)-1]] = value
}
