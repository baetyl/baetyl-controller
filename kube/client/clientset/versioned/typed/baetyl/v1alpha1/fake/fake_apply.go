/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/baetyl/baetyl-controller/kube/apis/baetyl/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeApplies implements ApplyInterface
type FakeApplies struct {
	Fake *FakeBaetylV1alpha1
	ns   string
}

var appliesResource = schema.GroupVersionResource{Group: "baetyl.apis", Version: "v1alpha1", Resource: "applies"}

var appliesKind = schema.GroupVersionKind{Group: "baetyl.apis", Version: "v1alpha1", Kind: "Apply"}

// Get takes name of the apply, and returns the corresponding apply object, and an error if there is any.
func (c *FakeApplies) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Apply, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(appliesResource, c.ns, name), &v1alpha1.Apply{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Apply), err
}

// List takes label and field selectors, and returns the list of Applies that match those selectors.
func (c *FakeApplies) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ApplyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(appliesResource, appliesKind, c.ns, opts), &v1alpha1.ApplyList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ApplyList{ListMeta: obj.(*v1alpha1.ApplyList).ListMeta}
	for _, item := range obj.(*v1alpha1.ApplyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested applies.
func (c *FakeApplies) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(appliesResource, c.ns, opts))

}

// Create takes the representation of a apply and creates it.  Returns the server's representation of the apply, and an error, if there is any.
func (c *FakeApplies) Create(ctx context.Context, apply *v1alpha1.Apply, opts v1.CreateOptions) (result *v1alpha1.Apply, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(appliesResource, c.ns, apply), &v1alpha1.Apply{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Apply), err
}

// Update takes the representation of a apply and updates it. Returns the server's representation of the apply, and an error, if there is any.
func (c *FakeApplies) Update(ctx context.Context, apply *v1alpha1.Apply, opts v1.UpdateOptions) (result *v1alpha1.Apply, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(appliesResource, c.ns, apply), &v1alpha1.Apply{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Apply), err
}

// Delete takes name of the apply and deletes it. Returns an error if one occurs.
func (c *FakeApplies) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(appliesResource, c.ns, name), &v1alpha1.Apply{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeApplies) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(appliesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ApplyList{})
	return err
}

// Patch applies the patch and returns the patched apply.
func (c *FakeApplies) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Apply, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(appliesResource, c.ns, name, pt, data, subresources...), &v1alpha1.Apply{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Apply), err
}