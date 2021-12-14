package kube

import (
	"context"
	"fmt"
	"time"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeInformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/baetyl/baetyl-controller/kube/apis/baetyl/v1alpha1"
	clientSet "github.com/baetyl/baetyl-controller/kube/client/clientset/versioned"
	customScheme "github.com/baetyl/baetyl-controller/kube/client/clientset/versioned/scheme"
	customInformer "github.com/baetyl/baetyl-controller/kube/client/informers/externalversions"
	baetylInformer "github.com/baetyl/baetyl-controller/kube/client/informers/externalversions/baetyl/v1alpha1"
	baetylLister "github.com/baetyl/baetyl-controller/kube/client/listers/baetyl/v1alpha1"
)

const (
	DefaultResyncApply       = time.Second * 30
	ControllerAgentNameApply = "apply-controller"
	ApplyKind                = "Apply"
	DefaultValuesKey         = "default"

	ApplySuccessSynced     = "ApplySynced"
	ErrApplyResourceExists = "ErrApplyResourceExists"
	MessageApplySynced     = "Apply synced successfully"
	MessageApplyExists     = "Resource %q already exists and is not managed by Apply"
)

type Apply interface {
	CreateApply(clt *v1alpha1.Apply) (*v1alpha1.Apply, error)
	GetApply(ns, name string) (*v1alpha1.Apply, error)
	UpdateApply(clt *v1alpha1.Apply) (*v1alpha1.Apply, error)
	DeleteApply(ns, name string) error
	ListApplies(ns string, selector labels.Selector) ([]*v1alpha1.Apply, error)
}

type ApplyClient struct {
	ctx         context.Context
	threadiness int
	stopCh      <-chan struct{}

	kubeClient       kubernetes.Interface
	customClient     clientSet.Interface
	kubeFactory      kubeInformers.SharedInformerFactory
	customFactory    customInformer.SharedInformerFactory
	applyInformer    baetylInformer.ApplyInformer
	applyLister      baetylLister.ApplyLister
	applySynced      cache.InformerSynced
	templateInformer baetylInformer.TemplateInformer
	templateLister   baetylLister.TemplateLister
	templateSynced   cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func NewApplyClient(ctx context.Context, kubeClient kubernetes.Interface, customClient clientSet.Interface, threadiness int, stopCh <-chan struct{}) (*ApplyClient, error) {
	// factory
	kubeFactory := kubeInformers.NewSharedInformerFactory(kubeClient, DefaultResyncApply)
	customFactory := customInformer.NewSharedInformerFactory(customClient, DefaultResyncApply)
	applyInformer := customFactory.Baetyl().V1alpha1().Applies()
	templateInformer := customFactory.Baetyl().V1alpha1().Templates()

	// Create event broadcaster
	// Add my-controller types to the default Kubernetes Scheme so Events can be
	// logged for my-controller types.
	utilRuntime.Must(customScheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedCoreV1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, coreV1.EventSource{Component: ControllerAgentNameApply})

	client := &ApplyClient{
		ctx:         ctx,
		threadiness: threadiness,
		stopCh:      stopCh,

		kubeClient:    kubeClient,
		customClient:  customClient,
		kubeFactory:   kubeFactory,
		customFactory: customFactory,

		templateInformer: templateInformer,
		templateLister:   templateInformer.Lister(),
		templateSynced:   templateInformer.Informer().HasSynced,

		applyInformer: applyInformer,
		applyLister:   applyInformer.Lister(),
		applySynced:   applyInformer.Informer().HasSynced,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ApplyKind),
		recorder:  recorder,
	}
	client.registerEventHandler()
	return client, nil
}

func (c *ApplyClient) registerEventHandler() {
	// Set up an event handler for when Apply resources change
	c.applyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueApply,
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.enqueueApply(newObj)
		},
	})
}

func (c *ApplyClient) Start() {
	c.kubeFactory.Start(c.stopCh)
	c.customFactory.Start(c.stopCh)
}

// 启动controller
// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *ApplyClient) Run() error {
	defer utilRuntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Apply controller")

	// 在worker运行之前，必须要等待状态的同步完成
	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(c.stopCh, c.applySynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// 启动多个 worker 协程并发从 queue 中获取需要处理的 item
	// runWorker 是包含真正的业务逻辑的函数
	// Launch n workers to process Apply resources
	for i := 0; i < c.threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, c.stopCh)
	}

	klog.Info("Started workers")
	<-c.stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *ApplyClient) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem 从 workqueue 中获取一个任务并最终调用 syncHandler 执行她
func (c *ApplyClient) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	// 这里写成函数形式是为了方便里面能直接调用 defer
	err := func(obj interface{}) error {
		// 通过调用 Done 方法可以通知 workqueue 完成了这个任务
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// 通过调用 Forget 方法可以避免任务被再次入队，比如调用一个任务出错后，为了避免
			// 它再次放入队列底部并在 back-off 后再次尝试，可以调用这个方法
			c.workqueue.Forget(obj)
			utilRuntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilRuntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Apply
// resource with the current status of the resource.
func (c *ApplyClient) syncHandler(key string) error {
	//// Convert the namespace/name string into a distinct namespace and name
	//ns, n, err := cache.SplitMetaNamespaceKey(key)
	//if err != nil {
	//	utilRuntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
	//	return nil
	//}
	//tpl, err := c.applyLister.Applies(ns).Get(n)
	//if err != nil {
	//	// The Apply resource may no longer exist, in which case we stop
	//	// processing.
	//	if apiErrors.IsNotFound(err) {
	//		utilRuntime.HandleError(fmt.Errorf("apply '%s' in work queue no longer exists", key))
	//		return nil
	//	}
	//	return err
	//}
	//
	//klog.Info("apply instance name:", tpl.Name, " ,version:", tpl.ObjectMeta.ResourceVersion)
	//
	//// update apply instance
	//_, err = c.customClient.BaetylV1alpha1().Applies(tpl.Namespace).Update(c.ctx, tpl.DeepCopy(), metaV1.UpdateOptions{})
	//if err != nil {
	//	return err
	//}
	//
	//c.recorder.Event(tpl, coreV1.EventTypeNormal, ApplySuccessSynced, MessageApplySynced)
	return nil
}

// enqueueApply takes a enqueueApply resource and converts
// it into a namespace/name string which is then put onto the work queue.
// This method should *not* be passed resources of any type other than enqueueApply.
//
// 它需要一个 enqueueApply 资源并将其转换为命名空间/名称字符串，然后将其放入工作队列。
// 此方法不应传递除 enqueueApply 之外的任何类型的资源。
func (c *ApplyClient) enqueueApply(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilRuntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *ApplyClient) CreateApply(tpl *v1alpha1.Apply) (*v1alpha1.Apply, error) {
	return c.customClient.BaetylV1alpha1().Applies(tpl.Namespace).Create(c.ctx, tpl, metaV1.CreateOptions{})
}

func (c *ApplyClient) GetApply(ns, name string) (*v1alpha1.Apply, error) {
	return c.applyLister.Applies(ns).Get(name)
}

func (c *ApplyClient) UpdateApply(tpl *v1alpha1.Apply) (*v1alpha1.Apply, error) {
	return c.customClient.BaetylV1alpha1().Applies(tpl.Namespace).Update(c.ctx, tpl, metaV1.UpdateOptions{})
}

func (c *ApplyClient) DeleteApply(ns, name string) error {
	return c.customClient.BaetylV1alpha1().Applies(ns).Delete(c.ctx, name, metaV1.DeleteOptions{})
}

func (c *ApplyClient) ListApplies(ns string, selector labels.Selector) ([]*v1alpha1.Apply, error) {
	return c.applyLister.Applies(ns).List(selector)
}
