// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-1-12
// Last Modified by JamesBryce

package fake

import (
	"time"

	"k8s.io/client-go/tools/cache"
)

type SharedIndexInformer struct {
	Handler cache.ResourceEventHandler
}

// AddIndexers add indexers to the informer before it starts.
func (fs *SharedIndexInformer) AddIndexers(indexers cache.Indexers) (err error) {
	return
}
func (fs *SharedIndexInformer) GetIndexer() (index cache.Indexer) { return }

func (fs *SharedIndexInformer) AddEventHandler(handler cache.ResourceEventHandler) {
	fs.Handler = handler
}

// AddEventHandlerWithResyncPeriod adds an event handler to the shared informer using the
// specified resync period.  Events to a single handler are delivered sequentially, but there is
// no coordination between different handlers.
func (fs *SharedIndexInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) {
}

// GetStore returns the Store.
func (fs *SharedIndexInformer) GetStore() (store cache.Store) { return }

// GetController gives back a synthetic interface that "votes" to start the informer
func (fs *SharedIndexInformer) GetController() (c cache.Controller) { return }

// Run starts the shared informer, which will be stopped when stopCh is closed.
func (fs *SharedIndexInformer) Run(stopCh <-chan struct{}) {}

// HasSynced returns true if the shared informer's store has synced.
func (fs *SharedIndexInformer) HasSynced() (synced bool) { return }

// LastSyncResourceVersion is the resource version observed when last synced with the underlying
// store. The value returned is not synchronized with access to the underlying store and is not
// thread-safe.
func (fs *SharedIndexInformer) LastSyncResourceVersion() (version string) { return }
