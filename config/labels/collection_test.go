// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package labels_test

import (
	"testing"

	"istio.io/pkg/config/labels"
)

func TestCollection_HasSubsetOf(t *testing.T) {
	a := labels.Instance{"app": "a"}
	b := labels.Instance{"app": "b"}
	a1 := labels.Instance{"app": "a", "prod": "env"}
	ab := labels.Collection{a, b}
	a1b := labels.Collection{a1, b}
	none := labels.Collection{}

	// equivalent to empty tag
	singleton := labels.Collection{nil}

	matching := []struct {
		instance   labels.Instance
		collection labels.Collection
	}{
		{
			instance:   a,
			collection: ab},
		{b, ab},
		{a, none},
		{a, nil},
		{a, singleton},
		{a1, ab},
		{b, a1b},
	}

	if (labels.Collection{a}).HasSubsetOf(b) {
		t.Errorf("{a}.HasSubsetOf(b) => Got true")
	}

	for _, pair := range matching {
		if !pair.collection.HasSubsetOf(pair.instance) {
			t.Errorf("%v.HasSubsetOf(%v) => Got false", pair.collection, pair.instance)
		}
	}
}
