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
	"testing"
)

func TestMapStrPut(t *testing.T) {
	tests := []struct {
		name      string
		initial   MapStr
		key       string
		value     interface{}
		wantErr   bool
		wantValue interface{}
		checkKey  string
	}{
		{
			name:      "simple key",
			initial:   MapStr{},
			key:       "key",
			value:     "value",
			wantErr:   false,
			checkKey:  "key",
			wantValue: "value",
		},
		{
			name:      "nested key",
			initial:   MapStr{},
			key:       "a.b.c",
			value:     123,
			wantErr:   false,
			checkKey:  "a.b.c",
			wantValue: 123,
		},
		{
			name:      "overwrite existing",
			initial:   MapStr{"key": "old"},
			key:       "key",
			value:     "new",
			wantErr:   false,
			checkKey:  "key",
			wantValue: "new",
		},
		{
			name:    "empty key",
			initial: MapStr{},
			key:     "",
			value:   "value",
			wantErr: true,
		},
		{
			name:      "nested with existing parent",
			initial:   MapStr{"a": MapStr{"b": "old"}},
			key:       "a.b",
			value:     "new",
			wantErr:   false,
			checkKey:  "a.b",
			wantValue: "new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.initial.Put(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Put() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				got, err := tt.initial.GetValue(tt.checkKey)
				if err != nil {
					t.Errorf("GetValue() error = %v", err)
					return
				}
				if got != tt.wantValue {
					t.Errorf("GetValue() = %v, want %v", got, tt.wantValue)
				}
			}
		})
	}
}

func TestMapStrClone(t *testing.T) {
	original := MapStr{
		"key1": "value1",
		"key2": 123,
		"key3": MapStr{"nested": "value"},
	}

	clone := original.Clone()

	// Verify all keys are copied
	if len(clone) != len(original) {
		t.Errorf("Clone() length = %v, want %v", len(clone), len(original))
	}

	// Check simple values
	if clone["key1"] != "value1" {
		t.Errorf("Clone()[key1] = %v, want %v", clone["key1"], "value1")
	}
	if clone["key2"] != 123 {
		t.Errorf("Clone()[key2] = %v, want %v", clone["key2"], 123)
	}

	// Verify it's a shallow copy (nested maps are shared)
	clone["key1"] = "modified"
	if original["key1"] == "modified" {
		t.Error("Clone() should not modify original simple values")
	}

	// Nested maps are shared in shallow copy
	if nested, ok := clone["key3"].(MapStr); ok {
		nested["nested"] = "modified"
		if origNested, ok := original["key3"].(MapStr); ok {
			if origNested["nested"] != "modified" {
				t.Error("Clone() is shallow, nested maps should be shared")
			}
		}
	}
}

func TestMapStrDeepUpdate(t *testing.T) {
	tests := []struct {
		name     string
		initial  MapStr
		update   MapStr
		expected MapStr
	}{
		{
			name:    "simple merge",
			initial: MapStr{"a": 1},
			update:  MapStr{"b": 2},
			expected: MapStr{
				"a": 1,
				"b": 2,
			},
		},
		{
			name:    "overwrite value",
			initial: MapStr{"a": 1},
			update:  MapStr{"a": 2},
			expected: MapStr{
				"a": 2,
			},
		},
		{
			name: "nested merge",
			initial: MapStr{
				"a": MapStr{"b": 1},
			},
			update: MapStr{
				"a": MapStr{"c": 2},
			},
			expected: MapStr{
				"a": MapStr{
					"b": 1,
					"c": 2,
				},
			},
		},
		{
			name: "deep nested merge",
			initial: MapStr{
				"a": MapStr{
					"b": MapStr{"c": 1},
				},
			},
			update: MapStr{
				"a": MapStr{
					"b": MapStr{"d": 2},
				},
			},
			expected: MapStr{
				"a": MapStr{
					"b": MapStr{
						"c": 1,
						"d": 2,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.DeepUpdate(tt.update)
			if !deepEqual(tt.initial, tt.expected) {
				t.Errorf("DeepUpdate() = %v, want %v", tt.initial, tt.expected)
			}
		})
	}
}

func TestMapStrGetValue(t *testing.T) {
	m := MapStr{
		"simple": "value",
		"nested": MapStr{
			"key": "nested_value",
			"deep": MapStr{
				"key": "deep_value",
			},
		},
	}

	tests := []struct {
		name      string
		key       string
		wantValue interface{}
		wantErr   bool
		errType   error
	}{
		{
			name:      "simple key",
			key:       "simple",
			wantValue: "value",
			wantErr:   false,
		},
		{
			name:      "nested key",
			key:       "nested.key",
			wantValue: "nested_value",
			wantErr:   false,
		},
		{
			name:      "deep nested key",
			key:       "nested.deep.key",
			wantValue: "deep_value",
			wantErr:   false,
		},
		{
			name:    "non-existent key",
			key:     "nonexistent",
			wantErr: true,
			errType: ErrKeyNotFound,
		},
		{
			name:    "non-existent nested key",
			key:     "nested.nonexistent",
			wantErr: true,
			errType: ErrKeyNotFound,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.GetValue(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("GetValue() error = %v, want %v", err, tt.errType)
			}
			if !tt.wantErr && got != tt.wantValue {
				t.Errorf("GetValue() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestMapStrHasKey(t *testing.T) {
	m := MapStr{
		"simple": "value",
		"nested": MapStr{
			"key": "value",
		},
	}

	tests := []struct {
		name    string
		key     string
		want    bool
		wantErr bool
	}{
		{
			name: "existing simple key",
			key:  "simple",
			want: true,
		},
		{
			name: "existing nested key",
			key:  "nested.key",
			want: true,
		},
		{
			name: "non-existent key",
			key:  "nonexistent",
			want: false,
		},
		{
			name: "non-existent nested key",
			key:  "nested.nonexistent",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.HasKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HasKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapStrDelete(t *testing.T) {
	tests := []struct {
		name    string
		initial MapStr
		key     string
		wantErr bool
		errType error
	}{
		{
			name:    "delete simple key",
			initial: MapStr{"key": "value"},
			key:     "key",
			wantErr: false,
		},
		{
			name: "delete nested key",
			initial: MapStr{
				"a": MapStr{"b": "value"},
			},
			key:     "a.b",
			wantErr: false,
		},
		{
			name:    "delete non-existent key",
			initial: MapStr{},
			key:     "nonexistent",
			wantErr: true,
			errType: ErrKeyNotFound,
		},
		{
			name:    "empty key",
			initial: MapStr{},
			key:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initial.Delete(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("Delete() error = %v, want %v", err, tt.errType)
			}
			if !tt.wantErr {
				has, _ := tt.initial.HasKey(tt.key)
				if has {
					t.Errorf("Delete() key still exists after deletion")
				}
			}
		})
	}
}

func TestMapStrFlatten(t *testing.T) {
	tests := []struct {
		name     string
		input    MapStr
		expected MapStr
	}{
		{
			name: "simple map",
			input: MapStr{
				"a": 1,
				"b": 2,
			},
			expected: MapStr{
				"a": 1,
				"b": 2,
			},
		},
		{
			name: "nested map",
			input: MapStr{
				"a": MapStr{
					"b": 1,
					"c": 2,
				},
			},
			expected: MapStr{
				"a.b": 1,
				"a.c": 2,
			},
		},
		{
			name: "deep nested map",
			input: MapStr{
				"a": MapStr{
					"b": MapStr{
						"c": 1,
					},
				},
			},
			expected: MapStr{
				"a.b.c": 1,
			},
		},
		{
			name: "mixed levels",
			input: MapStr{
				"a": 1,
				"b": MapStr{
					"c": 2,
					"d": MapStr{
						"e": 3,
					},
				},
			},
			expected: MapStr{
				"a":     1,
				"b.c":   2,
				"b.d.e": 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.Flatten()
			if !deepEqual(got, tt.expected) {
				t.Errorf("Flatten() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function to compare MapStr deeply
func deepEqual(a, b MapStr) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if aMap, ok := v.(MapStr); ok {
			if bMap, ok := bv.(MapStr); ok {
				if !deepEqual(aMap, bMap) {
					return false
				}
			} else {
				return false
			}
		} else if v != bv {
			return false
		}
	}
	return true
}
