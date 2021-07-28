// Copyright 2020-2021 Matt Layher
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

package system

// State is a type which can manipulate the low-level IPv6 parameters of
// a system.
type State interface {
	IPv6Autoconf(iface string) (bool, error)
	IPv6Forwarding(iface string) (bool, error)
	SetIPv6Autoconf(iface string, enable bool) error
}

// NewState creates State which directly manipulates the operating system.
func NewState() State { return systemState{} }

// A systemState directly manipulates the operating system's state.
type systemState struct{}

var _ State = systemState{}

func (systemState) IPv6Autoconf(iface string) (bool, error)   { return getIPv6Autoconf(iface) }
func (systemState) IPv6Forwarding(iface string) (bool, error) { return getIPv6Forwarding(iface) }
func (systemState) SetIPv6Autoconf(iface string, enable bool) error {
	return setIPv6Autoconf(iface, enable)
}

// A TestState is a State which is primarily useful in tests.
type TestState struct {
	// Global settings for any interface name.
	Autoconf, Forwarding bool
	Error                error

	// Alternatively, you may configure parameters individually on a
	// per-interface basis. Note that these configurations will override any
	// global configurations set above.
	Interfaces map[string]TestStateInterface
}

// A TestStateInterface sets the State configuration for a simulated network interface.
type TestStateInterface struct {
	Autoconf, Forwarding bool
}

var _ State = TestState{}

// IPv6Autoconf implements State.
func (ts TestState) IPv6Autoconf(iface string) (bool, error) {
	tsi, ok := ts.Interfaces[iface]
	if ok {
		return tsi.Autoconf, ts.Error
	}

	// Fall back to global configuration.
	return ts.Autoconf, ts.Error
}

// IPv6Forwarding implements State.
func (ts TestState) IPv6Forwarding(iface string) (bool, error) {
	tsi, ok := ts.Interfaces[iface]
	if ok {
		return tsi.Forwarding, ts.Error
	}

	// Fall back to global configuration.
	return ts.Forwarding, ts.Error
}

// SetIPv6Autoconf implements State.
func (ts TestState) SetIPv6Autoconf(iface string, _ bool) error {
	return ts.Error
}
