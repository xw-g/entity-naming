// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package entname provides entity-naming functions.
package entname

import (
	"fmt"
	"strings"

	"github.com/openconfig/entity-naming/internal/arista"
	"github.com/openconfig/entity-naming/internal/cisco"
	"github.com/openconfig/entity-naming/internal/juniper"
	"github.com/openconfig/entity-naming/internal/namer"
	"github.com/openconfig/entity-naming/internal/nokia"
	"github.com/openconfig/entity-naming/oc"
)

// Vendor is an enum of network device suppliers.
type Vendor string

// Vendor enum constants.
const (
	VendorArista  = Vendor("Arista")
	VendorCisco   = Vendor("Cisco")
	VendorJuniper = Vendor("Juniper")
	VendorNokia   = Vendor("Nokia")
)

var namerFactories = map[Vendor]func(string) namer.Namer{
	VendorArista:  func(hwm string) namer.Namer { return &arista.Namer{HardwareModel: hwm} },
	VendorCisco:   func(hwm string) namer.Namer { return &cisco.Namer{HardwareModel: hwm} },
	VendorJuniper: func(hwm string) namer.Namer { return &juniper.Namer{HardwareModel: hwm} },
	VendorNokia:   func(hwm string) namer.Namer { return &nokia.Namer{HardwareModel: hwm} },
}

// DeviceParams are parameters of a network device.
type DeviceParams struct {
	Vendor        Vendor
	HardwareModel string
}

func (dp *DeviceParams) String() string {
	if dp == nil {
		return "nil"
	}
	return fmt.Sprintf("%+v", *dp)
}

// PortChannelState indicates whether the port is channelized and channelizable.
type PortChannelState int

const (
	// Unchannelized means the port can be channelized but is not.
	Unchannelized PortChannelState = iota
	// Channelized means the port is channelized.
	Channelized
	// Unchannelizable means the port cannot be channelized.
	Unchannelizable
)

// PortParams are parameters of a network port.
//
//go:generate ./oc/generate.sh
type PortParams struct {
	SlotIndex, PICIndex, PortIndex, ChannelIndex int
	ChannelState                                 PortChannelState
	Speed                                        oc.E_IfEthernet_ETHERNET_SPEED
}

func (pp *PortParams) String() string {
	if pp == nil {
		return "nil"
	}
	return fmt.Sprintf("%+v", *pp)
}

// LoopbackInterface returns the vendor-specific name of the loopback
// interface with the given zero-based index.
func LoopbackInterface(dp *DeviceParams, index int) (string, error) {
	n, err := lookupNamer(dp)
	if err != nil {
		return "", err
	}
	if index < 0 {
		return "", fmt.Errorf("interface index cannot be negative: %d", index)
	}
	return n.LoopbackInterface(uint(index))
}

// AggregateInterface returns the vendor-specific name of the aggregate
// interface with the given zero-based index.
func AggregateInterface(dp *DeviceParams, index int) (string, error) {
	n, err := lookupNamer(dp)
	if err != nil {
		return "", err
	}
	if index < 0 {
		return "", fmt.Errorf("interface index cannot be negative: %d", index)
	}
	return n.AggregateInterface(uint(index))
}

// AggregateMemberInterface returns the vendor-specific name of the member
// interface bound to the aggregate interface with the given zero-based index.
func AggregateMemberInterface(dp *DeviceParams, index int) (string, error) {
	n, err := lookupNamer(dp)
	if err != nil {
		return "", err
	}
	if index < 0 {
		return "", fmt.Errorf("interface index cannot be negative: %d", index)
	}
	return n.AggregateMemberInterface(uint(index))
}

// Port returns the vendor-specific name of the physical interface with the
// given port parameters.
func Port(dp *DeviceParams, pp *PortParams) (string, error) {
	n, err := lookupNamer(dp)
	if err != nil {
		return "", err
	}
	npp, err := namerPortParams(pp, n.IsFixedFormFactor())
	if err != nil {
		return "", err
	}
	return n.Port(npp)
}

func namerPortParams(pp *PortParams, fixedFormFactor bool) (*namer.PortParams, error) {
	switch {
	case pp.SlotIndex < 0:
		return nil, fmt.Errorf("slot index cannot be negative: %d", pp.SlotIndex)
	case pp.PICIndex < 0:
		return nil, fmt.Errorf("pic index cannot be negative: %d", pp.PICIndex)
	case pp.PortIndex < 0:
		return nil, fmt.Errorf("port index cannot be negative: %d", pp.PortIndex)
	case pp.ChannelIndex < 0:
		return nil, fmt.Errorf("channel index cannot be negative: %d", pp.ChannelIndex)
	case pp.SlotIndex > 0 && fixedFormFactor:
		return nil, fmt.Errorf("cannot have a non-zero slot index on a fixed form factor device")
	case pp.ChannelIndex > 0 && pp.ChannelState != Channelized:
		return nil, fmt.Errorf("cannot have a non-zero channel index with an unchannelized port")
	case pp.Speed == oc.IfEthernet_ETHERNET_SPEED_UNSET || pp.Speed == oc.IfEthernet_ETHERNET_SPEED_SPEED_UNKNOWN:
		return nil, fmt.Errorf("port speed cannot be unset or unknown")
	}
	npp := &namer.PortParams{
		PICIndex:      uint(pp.PICIndex),
		PortIndex:     uint(pp.PortIndex),
		Speed:         pp.Speed,
		Channelizable: pp.ChannelState != Unchannelizable,
	}
	if !fixedFormFactor {
		slotIndex := uint(pp.SlotIndex)
		npp.SlotIndex = &slotIndex
	}
	if pp.ChannelState == Channelized {
		channelIndex := uint(pp.ChannelIndex)
		npp.ChannelIndex = &channelIndex
	}
	return npp, nil
}

// Linecard returns the vendor-specific name of the linecard with the given
// zero-based index.
func Linecard(dp *DeviceParams, index int) (string, error) {
	n, err := lookupNamer(dp)
	if err != nil {
		return "", err
	}
	if index < 0 {
		return "", fmt.Errorf("interface index cannot be negative: %d", index)
	}
	return n.Linecard(uint(index))
}

// ControllerCard returns the vendor-specific name of the controller card with
// the given zero-based index.
func ControllerCard(dp *DeviceParams, index int) (string, error) {
	n, err := lookupNamer(dp)
	if err != nil {
		return "", err
	}
	if index < 0 {
		return "", fmt.Errorf("interface index cannot be negative: %d", index)
	}
	return n.ControllerCard(uint(index))
}

// Fabric returns the vendor-specific name of the fabric with the given
// zero-based index.
func Fabric(dp *DeviceParams, index int) (string, error) {
	n, err := lookupNamer(dp)
	if err != nil {
		return "", err
	}
	if index < 0 {
		return "", fmt.Errorf("interface index cannot be negative: %d", index)
	}
	return n.Fabric(uint(index))
}

// QosForwardingGroup represents a QoS forwarding group.
// See the forwarding group definitions here:
// https://github.com/openconfig/entity-naming/blob/main/README.md#qos-forwarding-groups
type QosForwardingGroup string

const (
	// NC1 is the NC1 forwarding group.
	NC1 = QosForwardingGroup("NC1")
	// AF4 is the AF4 forwarding group.
	AF4 = QosForwardingGroup("AF4")
	// AF3 is the AF3 forwarding group.
	AF3 = QosForwardingGroup("AF3")
	// AF2 is the AF2 forwarding group.
	AF2 = QosForwardingGroup("AF2")
	// AF1 is the AF1 forwarding group.
	AF1 = QosForwardingGroup("AF1")
	// BE1 is the BE1 forwarding group.
	BE1 = QosForwardingGroup("BE1")
	// BE0 is the BE0 forwarding group.
	BE0 = QosForwardingGroup("BE0")
)

// QoSForwardingGroupNames are the names of common QoS forwarding groups.
type QoSForwardingGroupNames struct {
	nameByGroup map[QosForwardingGroup]string
}

// Name returns the name of the specified QoS forwarding group.
func (q *QoSForwardingGroupNames) Name(fg QosForwardingGroup) string {
	return q.nameByGroup[fg]
}

func (q *QoSForwardingGroupNames) String() string {
	if q == nil {
		return "nil"
	}
	var sb strings.Builder
	sb.WriteString("{\n")
	for k, v := range q.nameByGroup {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
	}
	sb.WriteString("}")
	return sb.String()
}

// QoSParams are parameters of a QoS configuration.
type QoSParams struct {
	NumStrictPriority, NumWeightedRoundRobin int
}

// QoSForwardingGroups returns the vendors-specific names of common QoS
// forwarding groups. See the forwarding group definitions here:
// https://github.com/openconfig/entity-naming/blob/main/README.md#qos-forwarding-groups
func QoSForwardingGroups(dev *DeviceParams, qos *QoSParams) (*QoSForwardingGroupNames, error) {
	n, err := lookupNamer(dev)
	if err != nil {
		return nil, err
	}
	nqp, err := namerQoSParams(qos)
	if err != nil {
		return nil, err
	}
	fgs, err := n.QoSForwardingGroups(nqp)
	if err != nil {
		return nil, err
	}
	return &QoSForwardingGroupNames{map[QosForwardingGroup]string{
		NC1: fgs.NC1,
		AF4: fgs.AF4,
		AF3: fgs.AF3,
		AF2: fgs.AF2,
		AF1: fgs.AF1,
		BE1: fgs.BE1,
		BE0: fgs.BE0,
	}}, nil
}

func namerQoSParams(qos *QoSParams) (*namer.QoSParams, error) {
	switch {
	case qos.NumStrictPriority < 0:
		return nil, fmt.Errorf("numStrictPriority cannot be negative: %d", qos.NumStrictPriority)
	case qos.NumWeightedRoundRobin < 0:
		return nil, fmt.Errorf("numWeightedRoundRobin cannot be negative: %d", qos.NumWeightedRoundRobin)
	}
	return &namer.QoSParams{
		NumStrictPriority:     uint(qos.NumStrictPriority),
		NumWeightedRoundRobin: uint(qos.NumWeightedRoundRobin),
	}, nil
}

// CommonTrafficQueueNames are the names of common traffic class queues.
type CommonTrafficQueueNames struct {
	NC1, AF4, AF3, AF2, AF1, BE1, BE0 string
}

func (qn *CommonTrafficQueueNames) String() string {
	if qn == nil {
		return "nil"
	}
	return fmt.Sprintf("%+v", *qn)
}

// CommonTrafficQueues returns the vendors-specific names of common traffic
// class queues. See the forwarding group definitions here:
// https://github.com/openconfig/entity-naming/blob/main/README.md#qos-forwarding-groups
func CommonTrafficQueues(dev *DeviceParams) (*CommonTrafficQueueNames, error) {
	n, err := lookupNamer(dev)
	if err != nil {
		return nil, err
	}
	fgs, err := n.QoSForwardingGroups(&namer.QoSParams{})
	if err != nil {
		return nil, err
	}
	return &CommonTrafficQueueNames{
		NC1: fgs.NC1,
		AF4: fgs.AF4,
		AF3: fgs.AF3,
		AF2: fgs.AF2,
		AF1: fgs.AF1,
		BE1: fgs.BE1,
		BE0: fgs.BE0,
	}, nil
}

func lookupNamer(dp *DeviceParams) (namer.Namer, error) {
	nf, ok := namerFactories[dp.Vendor]
	if !ok {
		return nil, fmt.Errorf("no Namer for vendor %v", dp.Vendor)
	}
	return nf(dp.HardwareModel), nil
}
