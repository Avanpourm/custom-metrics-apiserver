// Copyright: meitu.com
// Author: JamesBryce
// Author Email: zmp@meitu.com
// Date: 2018-1-12
// Last Modified by:   JamesBryce

package sources

import "time"

type CustomMetricsPoint struct {
	// Mtrics scrapt time
	Timestamp  time.Time
	WindowTime time.Duration
	// Usage is the metrics usage rate, in cores
	Usage ResourceList
}

// Merge a metrics
func (c *CustomMetricsPoint) Merge(new *CustomMetricsPoint) (metricsNum int64) {
	if c == nil || new == nil {
		return
	}
	if c.Timestamp.After(new.Timestamp) {
		c.Timestamp = new.Timestamp
	}
	metricsNum = c.Usage.Merge(&new.Usage)
	return metricsNum
}

// RateOfSome get a rate of some metrics
func (c *CustomMetricsPoint) RateOfSome(early *CustomMetricsPoint) (new *CustomMetricsPoint, metricsNum int64) {
	if c == nil || early == nil {
		return
	}
	new = &CustomMetricsPoint{
		Timestamp: c.Timestamp,
	}
	new.WindowTime = c.Timestamp.Sub(early.Timestamp)
	new.Usage, metricsNum = c.Usage.Rate(&early.Usage, int64(new.WindowTime.Seconds()))

	return new, metricsNum
}

// Equal check is the point is the same value
func (c *CustomMetricsPoint) Equal(d *CustomMetricsPoint) bool {
	if int64(c.Timestamp.Second()) != int64(d.Timestamp.Second()) || c.WindowTime != d.WindowTime || len(c.Usage) != len(d.Usage) {
		return false
	}
	for key, value := range c.Usage {
		v, ok := d.Usage[key]
		if !ok || v.String() != value.String() {
			return false
		}
	}
	for key, value := range d.Usage {
		v, ok := c.Usage[key]
		if !ok || v.String() != value.String() {
			return false
		}
	}
	return true
}
