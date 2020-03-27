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
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/containerd/containerd/api/services/sandbox/v1"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/sandbox/proxy"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var (
	unixAddr string
	debug    bool
)

func init() {
	flag.StringVar(&unixAddr, "address", "./sandbox.sock", "RPC server unix address (default: ./sandbox.sock)")
	flag.BoolVar(&debug, "debug", false, "Debug mode")
}

func main() {
	if !flag.Parsed() {
		flag.Parse()
	}

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, ctx := errgroup.WithContext(ctx)

	// Create `Controller` instance
	ctrl, err := newController(ctx)
	if err != nil {
		log.G(ctx).WithError(err).Fatal("failed to create controller service")
	}

	service := proxy.FromService(ctrl)

	// Create GRPC server
	server := grpc.NewServer()
	sandbox.RegisterControllerServer(server, service)

	listener, err := net.Listen("unix", unixAddr)
	if err != nil {
		log.G(ctx).WithError(err).Fatalf("failed to listen socket at %s", unixAddr)
	}

	group.Go(func() error {
		return server.Serve(listener)
	})

	group.Go(func() error {
		defer func() {
			log.G(ctx).Info("stopping server")
			server.Stop()
		}()

		for {
			select {
			case <-stop:
				cancel()
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})

	if err := group.Wait(); err != nil {
		log.G(ctx).WithError(err).Warn("controller error")
	}

	log.G(ctx).Info("done")
}
