package k8snamespace

import (
	"context"
	"fmt"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type NamespaceGetter interface {
	Get(string) (*v1.Namespace, error)
}

type NamespaceCache struct {
	indexer  cache.Indexer
	informer cache.Controller
}

func NewWatcher(ctx context.Context) (*NamespaceCache, error) {
	var config *restclient.Config
	var err error

	// config, err := rest.InClusterConfig()
	kubeconfig, found := os.LookupEnv("KUBECONFIG")
	if found {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	source := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), "namespaces", "", fields.Everything())

	eventHandler := &cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			_, isNamespace := obj.(*v1.Namespace)
			if !isNamespace {
				return false
			}
			// TODO change filter to relevant ns annotation
			return true
		},
		Handler: &namespaceLogger{},
	}

	// OnUpdate is called every resyncPeriod
	indexer, informer := cache.NewIndexerInformer(source, &v1.Namespace{}, time.Minute, eventHandler, cache.Indexers{})

	c := &NamespaceCache{
		indexer:  indexer,
		informer: informer,
	}

	go c.informer.Run(ctx.Done())

	ok := cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced)
	if !ok {
		return nil, fmt.Errorf("failed to sync cache")
	}

	return c, nil
}

// GetNamespace finds the Namespace by its name
func (c *NamespaceCache) Get(name string) (*v1.Namespace, error) {
	obj, exists, err := c.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return obj.(*v1.Namespace), nil
}

type namespaceLogger struct {
}

func (o *namespaceLogger) OnAdd(obj interface{}, isInInitialList bool) {
}

func (o *namespaceLogger) OnDelete(obj interface{}) {
}

func (o *namespaceLogger) OnUpdate(old, new interface{}) {
}
