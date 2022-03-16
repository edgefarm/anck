/*
Copyright © 2021 Ci4Rail GmbH <engineering@ci4rail.com>
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

	v1alpha1 "github.com/edgefarm/anck/apis/network/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeParticipantses implements ParticipantsInterface
type FakeParticipantses struct {
	Fake *FakeNetworkV1alpha1
	ns   string
}

var participantsesResource = schema.GroupVersionResource{Group: "network", Version: "v1alpha1", Resource: "participantses"}

var participantsesKind = schema.GroupVersionKind{Group: "network", Version: "v1alpha1", Kind: "Participants"}

// Get takes name of the participants, and returns the corresponding participants object, and an error if there is any.
func (c *FakeParticipantses) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Participants, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(participantsesResource, c.ns, name), &v1alpha1.Participants{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Participants), err
}

// List takes label and field selectors, and returns the list of Participantses that match those selectors.
func (c *FakeParticipantses) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ParticipantsList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(participantsesResource, participantsesKind, c.ns, opts), &v1alpha1.ParticipantsList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ParticipantsList{ListMeta: obj.(*v1alpha1.ParticipantsList).ListMeta}
	for _, item := range obj.(*v1alpha1.ParticipantsList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested participantses.
func (c *FakeParticipantses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(participantsesResource, c.ns, opts))

}

// Create takes the representation of a participants and creates it.  Returns the server's representation of the participants, and an error, if there is any.
func (c *FakeParticipantses) Create(ctx context.Context, participants *v1alpha1.Participants, opts v1.CreateOptions) (result *v1alpha1.Participants, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(participantsesResource, c.ns, participants), &v1alpha1.Participants{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Participants), err
}

// Update takes the representation of a participants and updates it. Returns the server's representation of the participants, and an error, if there is any.
func (c *FakeParticipantses) Update(ctx context.Context, participants *v1alpha1.Participants, opts v1.UpdateOptions) (result *v1alpha1.Participants, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(participantsesResource, c.ns, participants), &v1alpha1.Participants{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Participants), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeParticipantses) UpdateStatus(ctx context.Context, participants *v1alpha1.Participants, opts v1.UpdateOptions) (*v1alpha1.Participants, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(participantsesResource, "status", c.ns, participants), &v1alpha1.Participants{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Participants), err
}

// Delete takes name of the participants and deletes it. Returns an error if one occurs.
func (c *FakeParticipantses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(participantsesResource, c.ns, name, opts), &v1alpha1.Participants{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeParticipantses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(participantsesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ParticipantsList{})
	return err
}

// Patch applies the patch and returns the patched participants.
func (c *FakeParticipantses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Participants, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(participantsesResource, c.ns, name, pt, data, subresources...), &v1alpha1.Participants{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Participants), err
}
