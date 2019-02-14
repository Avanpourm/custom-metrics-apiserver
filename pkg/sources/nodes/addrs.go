// Copyright: 2018 Meitu.com Authors.
//
// Author: JamesBryce
// Author Email: zmp@meitu.com
// Date: 2018-02-01
// Last Modified by: JamesBryce

package nodes

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

var (
	// DefaultAddressTypePriority is the default node address type
	// priority list, as taken from the Kubernetes API server options.
	// In general, we prefer overrides to others, internal to external,
	// and DNS to IPs.
	DefaultAddressTypePriority = []corev1.NodeAddressType{
		// --override-hostname
		corev1.NodeHostName,

		// internal, preferring DNS if reported
		corev1.NodeInternalDNS,
		corev1.NodeInternalIP,

		// external, preferring DNS if reported
		corev1.NodeExternalDNS,
		corev1.NodeExternalIP,
	}
)

// AddressResolver knows how to find the preferred connection
// address for a given node.
type AddressResolver interface {
	// Address finds the preferred address to use to connect to
	// the given node.
	Address(node *corev1.Node) (address string, err error)
}

// prioNodeAddrResolver finds node addresses according to a list of
// priorities of types of addresses.
type prioNodeAddrResolver struct {
	addrTypePriority []corev1.NodeAddressType
}

func (r *prioNodeAddrResolver) Address(node *corev1.Node) (string, error) {
	// adapted from k8s.io/kubernetes/pkg/util/node
	for _, addrType := range r.addrTypePriority {
		for _, addr := range node.Status.Addresses {
			if addr.Type == addrType {
				return addr.Address, nil
			}
		}
	}

	return "", fmt.Errorf("node %s had no addresses that matched types %v", node.Name, r.addrTypePriority)
}

// NewPriorityAddressResolver creates a new NodeAddressResolver that resolves
// addresses first based on a list of prioritized address types, then based on
// address order (first to last) within a particular address type.
func NewPriorityAddressResolver(typePriority []corev1.NodeAddressType) AddressResolver {
	return &prioNodeAddrResolver{
		addrTypePriority: typePriority,
	}
}
