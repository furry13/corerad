// Copyright 2019 Matt Layher
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package corerad

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/corerad/internal/config"
	"github.com/mdlayher/ndp"
	"golang.org/x/sync/errgroup"
)

func TestAdvertiserAdvertiseUnsolicitedOneShot(t *testing.T) {
	// No configuration, bare minimum router advertisement.
	ad, c, done := testAdvertiser(t, nil)
	defer done()

	if err := c.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Fatalf("failed to set client read deadline: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var eg errgroup.Group
	eg.Go(func() error {
		if err := ad.Advertise(ctx); err != nil {
			return fmt.Errorf("failed to advertise: %v", err)
		}

		return nil
	})

	// Read a single advertisement and then ensure the advertiser can be halted.
	m, _, _, err := c.ReadFrom()
	if err != nil {
		t.Fatalf("failed to read RA: %v", err)
	}

	cancel()
	if err := eg.Wait(); err != nil {
		t.Fatalf("failed to stop advertiser: %v", err)
	}

	ra, ok := m.(*ndp.RouterAdvertisement)
	if !ok {
		t.Fatalf("did not receive an RA: %#v", m)
	}

	// There was no config specified, so assume the bare minimum for a valid RA.
	want := &ndp.RouterAdvertisement{
		Options: []ndp.Option{&ndp.LinkLayerAddress{
			Direction: ndp.Source,
			Addr:      ad.ifi.HardwareAddr,
		}},
	}

	if diff := cmp.Diff(want, ra); diff != "" {
		t.Fatalf("unexpected router advertisement (-want +got):\n%s", diff)
	}
}

func TestAdvertiserAdvertiseContextCanceled(t *testing.T) {
	ad, _, done := testAdvertiser(t, nil)
	defer done()

	timer := time.AfterFunc(5*time.Second, func() {
		panic("took too long")
	})
	defer timer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// This should not block because the context is already canceled.
	if err := ad.Advertise(ctx); err != nil {
		t.Fatalf("failed to advertise: %v", err)
	}
}

func testAdvertiser(t *testing.T, cfg *config.Interface) (*Advertiser, *ndp.Conn, func()) {
	t.Helper()

	// Allow empty config but always populate the interface name.
	// TODO: consider building veth pairs within the tests.
	if cfg == nil {
		cfg = &config.Interface{}
	}
	cfg.Name = "cradveth0"

	ad, err := NewAdvertiser(*cfg, nil)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skip("skipping, permission denied (run this test with CAP_NET_RAW)")
		}

		// Unfortunately this error isn't exposed as os.ErrNotExist.
		if strings.Contains(err.Error(), "no such network interface") {
			t.Skip("skipping, missing cradveth{0,1} veth pair")
		}

		t.Fatalf("failed to create advertiser: %v", err)
	}

	ifi, err := net.InterfaceByName("cradveth1")
	if err != nil {
		t.Skipf("skipping, failed to look up second veth: %v", err)
	}

	c, _, err := ndp.Dial(ifi, ndp.LinkLocal)
	if err != nil {
		t.Fatalf("failed to create NDP client connection: %v", err)
	}

	done := func() {
		if err := c.Close(); err != nil {
			t.Fatalf("failed to close NDP router solicitation connection: %v", err)
		}
	}

	return ad, c, done
}
