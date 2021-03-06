// MIT License
//
// Copyright (c) 2018 Jelmer Snoeck
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/jelmersnoeck/grafana-operator/apis/grafana-operator/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeDatasources implements DatasourceInterface
type FakeDatasources struct {
	Fake *FakeGrafanaV1alpha1
	ns   string
}

var datasourcesResource = schema.GroupVersionResource{Group: "grafana.sphc.io", Version: "v1alpha1", Resource: "datasources"}

var datasourcesKind = schema.GroupVersionKind{Group: "grafana.sphc.io", Version: "v1alpha1", Kind: "Datasource"}

// Get takes name of the datasource, and returns the corresponding datasource object, and an error if there is any.
func (c *FakeDatasources) Get(name string, options v1.GetOptions) (result *v1alpha1.Datasource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(datasourcesResource, c.ns, name), &v1alpha1.Datasource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Datasource), err
}

// List takes label and field selectors, and returns the list of Datasources that match those selectors.
func (c *FakeDatasources) List(opts v1.ListOptions) (result *v1alpha1.DatasourceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(datasourcesResource, datasourcesKind, c.ns, opts), &v1alpha1.DatasourceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.DatasourceList{}
	for _, item := range obj.(*v1alpha1.DatasourceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested datasources.
func (c *FakeDatasources) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(datasourcesResource, c.ns, opts))

}

// Create takes the representation of a datasource and creates it.  Returns the server's representation of the datasource, and an error, if there is any.
func (c *FakeDatasources) Create(datasource *v1alpha1.Datasource) (result *v1alpha1.Datasource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(datasourcesResource, c.ns, datasource), &v1alpha1.Datasource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Datasource), err
}

// Update takes the representation of a datasource and updates it. Returns the server's representation of the datasource, and an error, if there is any.
func (c *FakeDatasources) Update(datasource *v1alpha1.Datasource) (result *v1alpha1.Datasource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(datasourcesResource, c.ns, datasource), &v1alpha1.Datasource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Datasource), err
}

// Delete takes name of the datasource and deletes it. Returns an error if one occurs.
func (c *FakeDatasources) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(datasourcesResource, c.ns, name), &v1alpha1.Datasource{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeDatasources) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(datasourcesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.DatasourceList{})
	return err
}

// Patch applies the patch and returns the patched datasource.
func (c *FakeDatasources) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Datasource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(datasourcesResource, c.ns, name, data, subresources...), &v1alpha1.Datasource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Datasource), err
}
