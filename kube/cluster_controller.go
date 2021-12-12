package kube

import (
	"context"
	"fmt"
	"time"

	coreV1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
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
	DefaultResyncCluster       = time.Second * 30
	ControllerAgentNameCluster = "cluster-controller"
	ClusterKind                = "Cluster"

	ClusterSuccessSynced     = "ClusterSynced"
	ErrClusterResourceExists = "ErrClusterResourceExists"
	MessageClusterSynced     = "Cluster synced successfully"
	MessageClusterExists     = "Resource %q already exists and is not managed by Cluster"
)

type ClusterClient struct {
	ctx         context.Context
	threadiness int
	stopCh      <-chan struct{}

	kubeClient       kubernetes.Interface
	customClient     clientSet.Interface
	kubeFactory      kubeInformers.SharedInformerFactory
	customFactory    customInformer.SharedInformerFactory
	clusterInformer  baetylInformer.ClusterInformer
	clusterLister    baetylLister.ClusterLister
	clusterSynced    cache.InformerSynced
	templateInformer baetylInformer.TemplateInformer
	templateLister   baetylLister.TemplateLister
	templateSynced   cache.InformerSynced
	applyInformer    baetylInformer.ApplyInformer
	applyLister      baetylLister.ApplyLister
	applySynced      cache.InformerSynced

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

func NewClusterClient(ctx context.Context, kubeClient kubernetes.Interface, customClient clientSet.Interface, threadiness int, stopCh <-chan struct{}) (*ClusterClient, error) {
	// factory
	kubeFactory := kubeInformers.NewSharedInformerFactory(kubeClient, DefaultResyncCluster)
	customFactory := customInformer.NewSharedInformerFactory(customClient, DefaultResyncCluster)
	clusterInformer := customFactory.Baetyl().V1alpha1().Clusters()
	templateInformer := customFactory.Baetyl().V1alpha1().Templates()
	applyInformer := customFactory.Baetyl().V1alpha1().Applies()

	// Create event broadcaster
	// Add my-controller types to the default Kubernetes Scheme so Events can be
	// logged for my-controller types.
	utilRuntime.Must(customScheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedCoreV1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, coreV1.EventSource{Component: ControllerAgentNameCluster})

	client := &ClusterClient{
		ctx:         ctx,
		threadiness: threadiness,
		stopCh:      stopCh,

		kubeClient:       kubeClient,
		customClient:     customClient,
		kubeFactory:      kubeFactory,
		customFactory:    customFactory,
		clusterInformer:  clusterInformer,
		clusterLister:    clusterInformer.Lister(),
		clusterSynced:    clusterInformer.Informer().HasSynced,
		templateInformer: templateInformer,
		templateLister:   templateInformer.Lister(),
		templateSynced:   templateInformer.Informer().HasSynced,
		applyInformer:    applyInformer,
		applyLister:      applyInformer.Lister(),
		applySynced:      applyInformer.Informer().HasSynced,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ClusterKind),
		recorder:  recorder,
	}
	client.registerEventHandler()
	return client, nil
}

func (c *ClusterClient) registerEventHandler() {
	c.clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.clusterHandlerCreate,
		UpdateFunc: c.clusterHandlerUpdate,
	})

	c.templateInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.clusterHandleTemplateObjectCreate,
		UpdateFunc: c.clusterHandleTemplateObjectUpdate,
		DeleteFunc: c.clusterHandleTemplateObjectDelete,
	})

	c.applyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.clusterHandleApplyObjectCreate,
		UpdateFunc: c.clusterHandleApplyObjectUpdate,
		DeleteFunc: c.clusterHandleApplyObjectDelete,
	})
}

func (c *ClusterClient) Start() {
	c.kubeFactory.Start(c.stopCh)
	c.customFactory.Start(c.stopCh)
}

// 启动controller
// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *ClusterClient) Run() error {
	defer utilRuntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Cluster controller")

	// 在worker运行之前，必须要等待状态的同步完成
	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(c.stopCh, c.clusterSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// 启动多个 worker 协程并发从 queue 中获取需要处理的 item
	// runWorker 是包含真正的业务逻辑的函数
	// Launch n workers to process Cluster resources
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
func (c *ClusterClient) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem 从 workqueue 中获取一个任务并最终调用 syncHandler 执行她
func (c *ClusterClient) processNextWorkItem() bool {
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
// converge the two. It then updates the Status block of the Cluster
// resource with the current status of the resource.
func (c *ClusterClient) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	ns, n, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	clt, err := c.clusterLister.Clusters(ns).Get(n)
	if err != nil {
		// The Cluster resource may no longer exist, in which case we stop
		// processing.
		if apiErrors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf("cluster '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	if clt.Spec.TemplateRef == nil {
		clt.Spec.TemplateRef = &coreV1.ObjectReference{
			Kind:      TemplateKind,
			Namespace: clt.Namespace,
			Name:      clt.Name,
		}
	}
	if clt.Spec.ApplyRef == nil {
		clt.Spec.ApplyRef = &coreV1.ObjectReference{
			Kind:      ApplyKind,
			Namespace: clt.Namespace,
			Name:      clt.Name,
		}
	}

	if clt.Spec.TemplateRef.Name == "" {
		clt.Spec.TemplateRef.Kind = TemplateKind
		clt.Spec.TemplateRef.Name = clt.Name
		clt.Spec.TemplateRef.Namespace = clt.Namespace
	}
	if clt.Spec.ApplyRef.Name == "" {
		clt.Spec.ApplyRef.Kind = ApplyKind
		clt.Spec.ApplyRef.Name = clt.Name
		clt.Spec.ApplyRef.Namespace = clt.Namespace
	}

	_, err = c.checkAndStartTemplate(clt)
	if err != nil {
		return err
	}

	_, err = c.checkAndStartApply(clt)
	if err != nil {
		return err
	}

	klog.Info("cluster instance name:", clt.Name, " ,version:", clt.ObjectMeta.ResourceVersion)

	// update cluster instance
	_, err = c.customClient.BaetylV1alpha1().Clusters(clt.Namespace).Update(c.ctx, clt.DeepCopy(), metaV1.UpdateOptions{})
	if err != nil {
		return err
	}

	c.recorder.Event(clt, coreV1.EventTypeNormal, ClusterSuccessSynced, MessageClusterSynced)
	return nil
}

func (c *ClusterClient) checkAndStartTemplate(cluster *v1alpha1.Cluster) (*v1alpha1.Template, error) {
	ns, name := cluster.GetNamespace(), cluster.Spec.TemplateRef.Name
	tpl, err := c.templateLister.Templates(ns).Get(name)
	if apiErrors.IsNotFound(err) {
		klog.Infof("template not exist, create a new template %s in namespace %s", name, ns)

		// new Template
		res := &v1alpha1.Template{
			TypeMeta:   cluster.TypeMeta,
			ObjectMeta: *cluster.ObjectMeta.DeepCopy(),
			Spec: v1alpha1.TemplateSpec{
				UserSpec:    *cluster.Spec.UserSpec.DeepCopy(),
				ClusterRef:  &coreV1.LocalObjectReference{Name: cluster.Name},
				ApplyRef:    &coreV1.LocalObjectReference{Name: cluster.Spec.ApplyRef.Name},
				Data:        map[string]string{},
				Description: "",
			},
		}
		res.ResourceVersion = ""
		res.UID = ""
		res.OwnerReferences = []metaV1.OwnerReference{
			*metaV1.NewControllerRef(cluster, v1alpha1.SchemeGroupVersion.WithKind(ClusterKind)),
		}
		res.Labels["controller"] = cluster.Name
		tpl, err = c.customClient.BaetylV1alpha1().Templates(ns).Create(c.ctx, res, metaV1.CreateOptions{})
	}
	if err != nil {
		return nil, err
	}
	return tpl, nil
}

func (c *ClusterClient) checkAndStartApply(cluster *v1alpha1.Cluster) (*v1alpha1.Apply, error) {
	ns, name := cluster.GetNamespace(), cluster.Spec.ApplyRef.Name
	aly, err := c.applyLister.Applies(ns).Get(name)
	if apiErrors.IsNotFound(err) {
		klog.Infof("template not exist, create a new template %s in namespace %s", name, ns)

		// new Template
		res := &v1alpha1.Apply{
			TypeMeta:   cluster.TypeMeta,
			ObjectMeta: *cluster.ObjectMeta.DeepCopy(),
			Spec: v1alpha1.ApplySpec{
				UserSpec:     *cluster.Spec.UserSpec.DeepCopy(),
				ClusterRef:   &coreV1.LocalObjectReference{Name: cluster.Name},
				TemplatesRef: &coreV1.LocalObjectReference{Name: cluster.Spec.TemplateRef.Name},
				ApplyValues: []v1alpha1.ApplyValues{
					{
						Name:        DefaultValuesKey,
						Values:      map[string]string{},
						ExpectTime:  0,
						Description: "",
					},
				},
				Description: "",
			},
		}
		res.ResourceVersion = ""
		res.UID = ""
		res.OwnerReferences = []metaV1.OwnerReference{
			*metaV1.NewControllerRef(cluster, v1alpha1.SchemeGroupVersion.WithKind(ClusterKind)),
		}
		res.Labels["controller"] = cluster.Name
		aly, err = c.customClient.BaetylV1alpha1().Applies(ns).Create(c.ctx, res, metaV1.CreateOptions{})
	}
	if err != nil {
		return nil, err
	}
	return aly, nil
}

func (c *ClusterClient) clusterHandleTemplateObjectCreate(obj interface{}) {
	if !isValidMetaObject(obj) {
		return
	}
	tpl, ok := obj.(*v1alpha1.Template)
	if !ok {
		klog.V(4).Infof("failed to parse object to %s.%s", v1alpha1.SchemeGroupVersion, TemplateKind)
		return
	}

	if tpl.Spec.ClusterRef == nil || tpl.Spec.ClusterRef.Name == "" {
		klog.V(4).Info("failed to get cluster ref from %s in namespace %s", tpl.Namespace, tpl.Name)
		return
	}

	cluster, err := c.clusterLister.Clusters(tpl.GetNamespace()).Get(tpl.Spec.ClusterRef.Name)
	if err != nil {
		klog.V(4).Info("failed to get %s.%s name: %s", v1alpha1.SchemeGroupVersion, ClusterKind, tpl.Spec.ClusterRef.Name)
		return
	}
	cluster.Spec.TemplateRef = &coreV1.ObjectReference{
		Kind:            tpl.Kind,
		Namespace:       tpl.Namespace,
		Name:            tpl.Name,
		UID:             tpl.UID,
		APIVersion:      tpl.APIVersion,
		ResourceVersion: tpl.ResourceVersion,
	}
	c.enqueueCluster(cluster)
}

func (c *ClusterClient) clusterHandleTemplateObjectUpdate(oldObj, newObj interface{}) {
	if !isValidMetaObject(oldObj) {
		return
	}
	if !isValidMetaObject(newObj) {
		return
	}
	oldTpl, ok := oldObj.(*v1alpha1.Template)
	if !ok {
		klog.V(4).Infof("failed to parse old object to %s.%s", v1alpha1.SchemeGroupVersion, TemplateKind)
		return
	}
	newTpl, ok := newObj.(*v1alpha1.Template)
	if !ok {
		klog.V(4).Infof("failed to parse new object to %s.%s", v1alpha1.SchemeGroupVersion, TemplateKind)
		return
	}
	if oldTpl.ResourceVersion == newTpl.ResourceVersion {
		return
	}

	if newTpl.Spec.ClusterRef == nil || newTpl.Spec.ClusterRef.Name == "" {
		klog.V(4).Info("failed to get cluster ref from %s in namespace %s", newTpl.Namespace, newTpl.Name)
		return
	}

	cluster, err := c.clusterLister.Clusters(newTpl.GetNamespace()).Get(newTpl.Spec.ClusterRef.Name)
	if err != nil {
		klog.V(4).Info("failed to get %s.%s name: %s", v1alpha1.SchemeGroupVersion, ClusterKind, newTpl.Spec.ClusterRef.Name)
		return
	}
	cluster.Spec.TemplateRef = &coreV1.ObjectReference{
		Kind:            newTpl.Kind,
		Namespace:       newTpl.Namespace,
		Name:            newTpl.Name,
		UID:             newTpl.UID,
		APIVersion:      newTpl.APIVersion,
		ResourceVersion: newTpl.ResourceVersion,
	}
	c.enqueueCluster(cluster)
}

func (c *ClusterClient) clusterHandleTemplateObjectDelete(obj interface{}) {
	if !isValidMetaObject(obj) {
		return
	}
	tpl, ok := obj.(*v1alpha1.Template)
	if !ok {
		klog.V(4).Infof("failed to parse object to %s.%s", v1alpha1.SchemeGroupVersion, TemplateKind)
		return
	}

	if tpl.Spec.ClusterRef == nil || tpl.Spec.ClusterRef.Name == "" {
		klog.V(4).Info("failed to get cluster ref from %s in namespace %s", tpl.Namespace, tpl.Name)
		return
	}

	cluster, err := c.clusterLister.Clusters(tpl.GetNamespace()).Get(tpl.Spec.ClusterRef.Name)
	if err != nil {
		klog.V(4).Info("failed to get %s.%s name: %s", v1alpha1.SchemeGroupVersion, ClusterKind, tpl.Spec.ClusterRef.Name)
		return
	}
	cluster.Spec.TemplateRef = nil
	c.enqueueCluster(cluster)
}

func (c *ClusterClient) clusterHandleApplyObjectCreate(obj interface{}) {
	if !isValidMetaObject(obj) {
		return
	}
	aly, ok := obj.(*v1alpha1.Apply)
	if !ok {
		klog.V(4).Infof("failed to parse object to %s.%s", v1alpha1.SchemeGroupVersion, ApplyKind)
		return
	}

	if aly.Spec.ClusterRef == nil || aly.Spec.ClusterRef.Name == "" {
		klog.V(4).Info("failed to get cluster ref from %s in namespace %s", aly.Namespace, aly.Name)
		return
	}

	cluster, err := c.clusterLister.Clusters(aly.GetNamespace()).Get(aly.Spec.ClusterRef.Name)
	if err != nil {
		klog.V(4).Info("failed to get %s.%s name: %s", v1alpha1.SchemeGroupVersion, ClusterKind, aly.Spec.ClusterRef.Name)
		return
	}
	cluster.Spec.TemplateRef = &coreV1.ObjectReference{
		Kind:            aly.Kind,
		Namespace:       aly.Namespace,
		Name:            aly.Name,
		UID:             aly.UID,
		APIVersion:      aly.APIVersion,
		ResourceVersion: aly.ResourceVersion,
	}
	c.enqueueCluster(cluster)
}

func (c *ClusterClient) clusterHandleApplyObjectUpdate(oldObj, newObj interface{}) {
	if !isValidMetaObject(oldObj) {
		return
	}
	if !isValidMetaObject(newObj) {
		return
	}
	oldAly, ok := oldObj.(*v1alpha1.Apply)
	if !ok {
		klog.V(4).Infof("failed to parse old object to %s.%s", v1alpha1.SchemeGroupVersion, ApplyKind)
		return
	}
	newAly, ok := newObj.(*v1alpha1.Apply)
	if !ok {
		klog.V(4).Infof("failed to parse new object to %s.%s", v1alpha1.SchemeGroupVersion, ApplyKind)
		return
	}
	if oldAly.ResourceVersion == newAly.ResourceVersion {
		return
	}

	if newAly.Spec.ClusterRef == nil || newAly.Spec.ClusterRef.Name == "" {
		klog.V(4).Info("failed to get cluster ref from %s in namespace %s", newAly.Namespace, newAly.Name)
		return
	}

	cluster, err := c.clusterLister.Clusters(newAly.GetNamespace()).Get(newAly.Spec.ClusterRef.Name)
	if err != nil {
		klog.V(4).Info("failed to get %s.%s name: %s", v1alpha1.SchemeGroupVersion, ClusterKind, newAly.Spec.ClusterRef.Name)
		return
	}
	cluster.Spec.TemplateRef = &coreV1.ObjectReference{
		Kind:            newAly.Kind,
		Namespace:       newAly.Namespace,
		Name:            newAly.Name,
		UID:             newAly.UID,
		APIVersion:      newAly.APIVersion,
		ResourceVersion: newAly.ResourceVersion,
	}
	c.enqueueCluster(cluster)
}

func (c *ClusterClient) clusterHandleApplyObjectDelete(obj interface{}) {
	if !isValidMetaObject(obj) {
		return
	}
	aly, ok := obj.(*v1alpha1.Apply)
	if !ok {
		klog.V(4).Infof("failed to parse object to %s.%s", v1alpha1.SchemeGroupVersion, ApplyKind)
		return
	}

	if aly.Spec.ClusterRef == nil || aly.Spec.ClusterRef.Name == "" {
		klog.V(4).Info("failed to get cluster ref from %s in namespace %s", aly.Namespace, aly.Name)
		return
	}

	cluster, err := c.clusterLister.Clusters(aly.GetNamespace()).Get(aly.Spec.ClusterRef.Name)
	if err != nil {
		klog.V(4).Info("failed to get %s.%s name: %s", v1alpha1.SchemeGroupVersion, ClusterKind, aly.Spec.ClusterRef.Name)
		return
	}
	cluster.Spec.TemplateRef = nil
	c.enqueueCluster(cluster)
}

func (c *ClusterClient) clusterHandlerCreate(obj interface{}) {
	c.enqueueCluster(obj)
}

func (c *ClusterClient) clusterHandlerUpdate(oldObj, newObj interface{}) {
	oldClu, ok := oldObj.(*v1alpha1.Cluster)
	if !ok {
		klog.V(4).Infof("failed to parse old object to %s.%s", v1alpha1.SchemeGroupVersion, ClusterKind)
		return
	}
	newClu, ok := newObj.(*v1alpha1.Cluster)
	if !ok {
		klog.V(4).Infof("failed to parse new object to %s.%s", v1alpha1.SchemeGroupVersion, ClusterKind)
		return
	}
	if oldClu.ResourceVersion == newClu.ResourceVersion {
		return
	}
	c.enqueueCluster(newObj)
}

func (c *ClusterClient) enqueueCluster(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilRuntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *ClusterClient) CreateCluster(clt *v1alpha1.Cluster) (*v1alpha1.Cluster, error) {
	return c.customClient.BaetylV1alpha1().Clusters(clt.Namespace).Create(c.ctx, clt, metaV1.CreateOptions{})
}

func (c *ClusterClient) GetCluster(ns, name string) (*v1alpha1.Cluster, error) {
	return c.clusterLister.Clusters(ns).Get(name)
}

func (c *ClusterClient) UpdateCluster(clt *v1alpha1.Cluster) (*v1alpha1.Cluster, error) {
	return c.customClient.BaetylV1alpha1().Clusters(clt.Namespace).Update(c.ctx, clt, metaV1.UpdateOptions{})
}

func (c *ClusterClient) DeleteCluster(ns, name string) error {
	return c.customClient.BaetylV1alpha1().Clusters(ns).Delete(c.ctx, name, metaV1.DeleteOptions{})
}

func (c *ClusterClient) ListClusters(ns string, selector labels.Selector) ([]*v1alpha1.Cluster, error) {
	return c.clusterLister.Clusters(ns).List(selector)
}
