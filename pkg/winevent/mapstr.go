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
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrKeyNotFound indicates that the specified key was not found.
	ErrKeyNotFound = errors.New("key not found")
)

// MapStr is a map[string]interface{} wrapper with utility methods for
// manipulating nested data structures.
type MapStr map[string]interface{}

// Put associates the specified value with the specified key. If the map
// previously contained a mapping for the key, the old value is replaced and
// returned. The key can be expressed in dot-notation (e.g. x.y) to put a value
// into a nested map.
//
// If you need to differentiate between nil and non-existent keys, use HasKey.
func (m MapStr) Put(key string, value interface{}) (interface{}, error) {
	if key == "" {
		return nil, errors.New("key is empty")
	}

	// Split the key into path components
	keys := strings.Split(key, ".")
	current := m

	for i := 0; i < len(keys)-1; i++ {
		k := keys[i]
		val, exists := current[k]
		if !exists {
			// Create a new map for this level
			newMap := MapStr{}
			current[k] = newMap
			current = newMap
			continue
		}

		// Check if the existing value is a map
		if nextMap, ok := val.(MapStr); ok {
			current = nextMap
		} else if nextMap, ok := val.(map[string]interface{}); ok {
			// Convert map[string]interface{} to MapStr
			converted := MapStr(nextMap)
			current[k] = converted
			current = converted
		} else {
			// Path exists but is not a map
			return nil, fmt.Errorf("key %s is not a map", strings.Join(keys[:i+1], "."))
		}
	}

	// Set the final value
	finalKey := keys[len(keys)-1]
	old := current[finalKey]
	current[finalKey] = value
	return old, nil
}

// Clone creates a shallow copy of the MapStr.
func (m MapStr) Clone() MapStr {
	result := make(MapStr, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// DeepUpdate recursively updates the MapStr with the key-value pairs from
// the other map. If a key exists in both maps and both values are maps,
// the maps are merged recursively.
func (m MapStr) DeepUpdate(other MapStr) {
	for key, value := range other {
		if existingValue, exists := m[key]; exists {
			// If both values are maps, merge them recursively
			if existingMap, ok := existingValue.(MapStr); ok {
				if otherMap, ok := value.(MapStr); ok {
					existingMap.DeepUpdate(otherMap)
					continue
				}
			} else if existingMap, ok := existingValue.(map[string]interface{}); ok {
				if otherMap, ok := value.(MapStr); ok {
					converted := MapStr(existingMap)
					converted.DeepUpdate(otherMap)
					m[key] = converted
					continue
				} else if otherMap, ok := value.(map[string]interface{}); ok {
					converted := MapStr(existingMap)
					converted.DeepUpdate(MapStr(otherMap))
					m[key] = converted
					continue
				}
			}
		}
		// Otherwise just set/overwrite the value
		m[key] = value
	}
}

// GetValue gets a value from the map using a dot-notation key.
// Returns ErrKeyNotFound if the key does not exist.
func (m MapStr) GetValue(key string) (interface{}, error) {
	if key == "" {
		return nil, errors.New("key is empty")
	}

	keys := strings.Split(key, ".")
	current := m

	for i := 0; i < len(keys)-1; i++ {
		k := keys[i]
		val, exists := current[k]
		if !exists {
			return nil, ErrKeyNotFound
		}

		if nextMap, ok := val.(MapStr); ok {
			current = nextMap
		} else if nextMap, ok := val.(map[string]interface{}); ok {
			current = MapStr(nextMap)
		} else {
			return nil, fmt.Errorf("key %s is not a map", strings.Join(keys[:i+1], "."))
		}
	}

	finalKey := keys[len(keys)-1]
	val, exists := current[finalKey]
	if !exists {
		return nil, ErrKeyNotFound
	}
	return val, nil
}

// HasKey returns true if the key exists.
func (m MapStr) HasKey(key string) (bool, error) {
	_, err := m.GetValue(key)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Delete removes a key from the MapStr using dot-notation.
func (m MapStr) Delete(key string) error {
	if key == "" {
		return errors.New("key is empty")
	}

	keys := strings.Split(key, ".")
	current := m

	for i := 0; i < len(keys)-1; i++ {
		k := keys[i]
		val, exists := current[k]
		if !exists {
			return ErrKeyNotFound
		}

		if nextMap, ok := val.(MapStr); ok {
			current = nextMap
		} else if nextMap, ok := val.(map[string]interface{}); ok {
			current = MapStr(nextMap)
		} else {
			return fmt.Errorf("key %s is not a map", strings.Join(keys[:i+1], "."))
		}
	}

	finalKey := keys[len(keys)-1]
	if _, exists := current[finalKey]; !exists {
		return ErrKeyNotFound
	}
	delete(current, finalKey)
	return nil
}

// Flatten flattens the given MapStr into a flat map where nested keys
// are represented using dot notation.
func (m MapStr) Flatten() MapStr {
	result := make(MapStr)
	m.flattenRecursive("", result)
	return result
}

func (m MapStr) flattenRecursive(prefix string, result MapStr) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		if nested, ok := v.(MapStr); ok {
			nested.flattenRecursive(key, result)
		} else if nested, ok := v.(map[string]interface{}); ok {
			MapStr(nested).flattenRecursive(key, result)
		} else {
			result[key] = v
		}
	}
}
