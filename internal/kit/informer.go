package kit

import "k8s.io/client-go/tools/cache"

// NamedInformer is used to give a SharedIndexInformer a name which we can use
// for later references.
type NamedInformer struct {
	Name     string
	Informer cache.SharedIndexInformer
}

// NewNamedInformer sets up a new NamedInformer struct.
func NewNamedInformer(name string, informer cache.SharedIndexInformer) NamedInformer {
	return NamedInformer{Name: name, Informer: informer}
}
