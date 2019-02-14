// Copyright: 2018 Meitu.com Authors.
//
// Author: JamesBryce
// Author Email: zmp@meitu.com
// Date: 2018-02-01
// Last Modified by: JamesBryce

package nodes

// Info contains the information needed to identify and connect to a particular node
// (node name and preferred address).
type Info struct {
	Name           string
	ConnectAddress string
}
