// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package main

import (
	"context"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/sandbox"
	"github.com/containerd/typeurl"
	"github.com/pkg/errors"

	"github.com/firecracker-microvm/firecracker-containerd/proto"
)

type controller struct {
	impl *local
}

var _ sandbox.Controller = &controller{}

func newController(ctx context.Context) (*controller, error) {
	l, err := newLocal(ctx)
	if err != nil {
		return nil, err
	}

	return &controller{impl: l}, nil
}

func (c *controller) Start(ctx context.Context, id string, opts *sandbox.CreateOpts) (sandbox.Descriptor, error) {
	log.G(ctx).Debug("start")

	reqAny, ok := opts.Extensions["firecracker-create-request"]
	if !ok {
		return sandbox.Descriptor{}, errors.New("missing required firecracker extension")
	}

	any, err := typeurl.UnmarshalAny(&reqAny)
	if err != nil {
		return sandbox.Descriptor{}, err
	}
	createVMRequest := any.(*proto.CreateVMRequest)

	createVMRequest.VMID = id
	vm, err := c.impl.CreateVM(ctx, createVMRequest)
	if err != nil {
		return sandbox.Descriptor{}, errors.Wrap(err, "failed to start Firecracker")
	}

	descriptor, err := typeurl.MarshalAny(vm)
	if err != nil {
		return sandbox.Descriptor{}, errors.Wrap(err, "failed to marshal descriptor")
	}

	return *descriptor, nil
}

func (c *controller) Stop(ctx context.Context, id string) error {
	log.G(ctx).Debug("stop")

	_, err := c.impl.StopVM(ctx, &proto.StopVMRequest{
		VMID:           id,
		TimeoutSeconds: 100,
	})

	return err
}

func (c *controller) Update(ctx context.Context, id string, opts *sandbox.UpdateOpts, fieldpaths ...string) error {
	log.G(ctx).Debug("update")
	return errdefs.ErrNotImplemented
}

func (c *controller) Status(ctx context.Context, id string) (sandbox.Status, error) {
	log.G(ctx).Debug("status")

	resp, err := c.impl.GetVMInfo(ctx, &proto.GetVMInfoRequest{
		VMID: id,
	})

	if err != nil {
		return sandbox.Status{}, err
	}

	return sandbox.Status{
		ID:    resp.VMID,
		State: sandbox.StateReady,
	}, nil
}

func (c *controller) Delete(ctx context.Context, id string, opts *sandbox.DeleteOpts) error {
	log.G(ctx).Debug("delete")
	return nil
}
