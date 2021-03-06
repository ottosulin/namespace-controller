package controller

import (
	"log"
	"sync"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	cacheTime = 3 * 60 * time.Second
)

// Controller contains controller variables that is needed to work in class
type Controller struct {
	nsInformer cache.SharedInformer
	kclient    *kubernetes.Clientset
	config     *Config
}

// Run starts the process for listening for event changes and acting upon those changes.
func (c *Controller) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)

	// Execute go function
	go c.nsInformer.Run(stopCh)

	// Wait till we receive a stop signal
	<-stopCh
}

// NewNamespaceWatcher creates a new nsController
func NewNamespaceWatcher(kclient *kubernetes.Clientset, configFile string) *Controller {
	watcher := &Controller{}
	nsInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kclient.CoreV1().Namespaces().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kclient.CoreV1().Namespaces().Watch(options)
			},
		},
		&v1.Namespace{},
		cacheTime,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	nsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    watcher.createNS,
		UpdateFunc: watcher.updateNS,
	})

	watcher.kclient = kclient
	watcher.nsInformer = nsInformer

	config, err := makeConfig(configFile)
	if err != nil {
		log.Fatalf("could not load config '%+v'", err)
		return nil
	}
	watcher.config = config
	return watcher
}

func (c *Controller) createNS(obj interface{}) {
	c.checkAndUpdate(obj.(*v1.Namespace))
}

func (c *Controller) updateNS(old interface{}, obj interface{}) {
	c.checkAndUpdate(obj.(*v1.Namespace))
}
