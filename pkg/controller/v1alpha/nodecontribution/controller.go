/*
Copyright 2020 Sorbonne Université

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodecontribution

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"headnode/pkg/authorization"
	appsinformer_v1 "headnode/pkg/client/informers/externalversions/apps/v1alpha"
	"headnode/pkg/node"

	log "github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// The main structure of controller
type controller struct {
	logger       *log.Entry
	queue        workqueue.RateLimitingInterface
	informer     cache.SharedIndexInformer
	nodeInformer cache.SharedIndexInformer
	handler      HandlerInterface
}

// The main structure of informerevent
type informerevent struct {
	key      string
	function string
}

// JSON structure of patch operation
type patchByBoolValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value bool   `json:"value"`
}
type patchByOwnerReferenceValue struct {
	Op    string                `json:"op"`
	Path  string                `json:"path"`
	Value []patchOwnerReference `json:"value"`
}
type patchOwnerReference struct {
	APIVersion         string `json:"apiVersion"`
	BlockOwnerDeletion bool   `json:"blockOwnerDeletion"`
	Controller         bool   `json:"controller"`
	Kind               string `json:"kind"`
	Name               string `json:"name"`
	UID                string `json:"uid"`
}

// Constant variables for events
const failure = "Failure"
const recover = "Recovering"
const success = "Successful"
const noSchedule = "NoSchedule"
const create = "create"
const update = "update"
const delete = "delete"
const trueStr = "True"
const falseStr = "False"
const unknownStr = "Unknown"

// Start function is entry point of the controller
func Start() {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	edgenetClientset, err := authorization.CreateEdgeNetClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	NCHandler := &Handler{}
	// Create the nodecontribution informer which was generated by the code generator to list and watch nodecontribution resources
	informer := appsinformer_v1.NewNodeContributionInformer(
		edgenetClientset,
		metav1.NamespaceAll,
		0,
		cache.Indexers{},
	)
	// Create a work queue which contains a key of the resource to be handled by the handler
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	var event informerevent
	// Event handlers deal with events of resources. Here, there are three types of events as Add, Update, and Delete
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Put the resource object into a key
			event.key, err = cache.MetaNamespaceKeyFunc(obj)
			event.function = create
			log.Infof("Add nodecontribution: %s", event.key)
			if err == nil {
				// Add the key to the queue
				queue.Add(event)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			event.key, err = cache.MetaNamespaceKeyFunc(newObj)
			event.function = update
			log.Infof("Update nodecontribution: %s", event.key)
			if err == nil {
				queue.Add(event)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// DeletionHandlingMetaNamsespaceKeyFunc helps to check the existence of the object while it is still contained in the index.
			// Put the resource object into a key
			event.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			event.function = delete
			log.Infof("Delete nodecontribution: %s", event.key)
			if err == nil {
				queue.Add(event)
			}
		},
	})
	// The selectivedeployment resources are reconfigured according to node events in this section
	nodeInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			// The main purpose of listing is to attach geo labels to whole nodes at the beginning
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return clientset.CoreV1().Nodes().List(options)
			},
			// This function watches all changes/updates of nodes
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return clientset.CoreV1().Nodes().Watch(options)
			},
		},
		&corev1.Node{},
		0,
		cache.Indexers{},
	)
	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			nodeObj := obj.(*corev1.Node)
			for _, owner := range nodeObj.GetOwnerReferences() {
				if owner.Kind == "NodeContribution" {
					NCRaw, err := edgenetClientset.AppsV1alpha().NodeContributions("").
						List(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", owner.Name)})
					if err != nil {
						log.Println(err.Error())
						panic(err.Error())
					}
					if len(NCRaw.Items) == 0 {
						clientset.CoreV1().Nodes().Delete(nodeObj.GetName(), &metav1.DeleteOptions{})
					}
				}
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldObj := old.(*corev1.Node)
			newObj := new.(*corev1.Node)
			oldReady := node.GetConditionReadyStatus(oldObj)
			newReady := node.GetConditionReadyStatus(newObj)
			for _, owner := range newObj.GetOwnerReferences() {
				if owner.Kind == "NodeContribution" {
					NCRaw, err := edgenetClientset.AppsV1alpha().NodeContributions("").
						List(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", owner.Name)})
					if err != nil {
						log.Println(err.Error())
						panic(err.Error())
					}
					for _, NCRow := range NCRaw.Items {
						if NCRow.GetUID() == owner.UID {
							NCCopy := NCRow.DeepCopy()
							if (oldReady == falseStr && newReady == trueStr) ||
								(oldReady == unknownStr && newReady == trueStr) {
								if NCCopy.Status.State != success {
									NCCopy.Status.State = success
									if NCCopy.Status.State == recover {
										NCCopy.Status.Message = "Node recovery successful"
									} else {
										NCCopy.Status.Message = "Node is ready"
									}
									edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
								}
							} else if (oldReady == trueStr && newReady == falseStr) ||
								(oldReady == trueStr && newReady == unknownStr) {
								if NCCopy.Status.State != failure {
									NCCopy.Status.State = failure
									NCCopy.Status.Message = "Node is not ready"
									edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
								}
							}

							if (oldObj.Spec.Unschedulable == true && newObj.Spec.Unschedulable == false) ||
								(oldObj.Spec.Unschedulable == false && newObj.Spec.Unschedulable == true) {
								if NCCopy.Spec.Enabled == newObj.Spec.Unschedulable {
									// Create a patch slice and initialize it to the label size
									nodePatchArr := make([]patchByBoolValue, 1)
									nodePatch := patchByBoolValue{}
									// Append the data existing in the label map to the slice
									nodePatch.Op = "replace"
									nodePatch.Path = "/spec/unschedulable"
									nodePatch.Value = !NCCopy.Spec.Enabled
									nodePatchArr[0] = nodePatch
									nodePatchJSON, _ := json.Marshal(nodePatchArr)
									// Patch the nodes with the arguments:
									// hostname, patch type, and patch data
									_, err = clientset.CoreV1().Nodes().Patch(newObj.GetName(), types.JSONPatchType, nodePatchJSON)
								}
							}
						}
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			nodeObj := obj.(*corev1.Node)
			for _, owner := range nodeObj.GetOwnerReferences() {
				if owner.Kind == "NodeContribution" {
					NCRaw, err := edgenetClientset.AppsV1alpha().NodeContributions("").
						List(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", owner.Name)})
					if err != nil {
						log.Println(err.Error())
						panic(err.Error())
					}
					for _, NCRow := range NCRaw.Items {
						if NCRow.GetUID() == owner.UID {
							edgenetClientset.AppsV1alpha().NodeContributions(NCRow.GetNamespace()).Delete(NCRow.GetName(), &metav1.DeleteOptions{})
						}
					}
				}
			}
		},
	})
	controller := controller{
		logger:       log.NewEntry(log.New()),
		informer:     informer,
		nodeInformer: nodeInformer,
		queue:        queue,
		handler:      NCHandler,
	}

	// A channel to terminate elegantly
	stopCh := make(chan struct{})
	defer close(stopCh)
	// Run the controller loop as a background task to start processing resources
	go controller.run(stopCh)
	// A channel to observe OS signals for smooth shut down
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)
	signal.Notify(sigTerm, syscall.SIGINT)
	<-sigTerm
}

// Run starts the controller loop
func (c *controller) run(stopCh <-chan struct{}) {
	// A Go panic which includes logging and terminating
	defer utilruntime.HandleCrash()
	// Shutdown after all goroutines have done
	defer c.queue.ShutDown()
	c.logger.Info("run: initiating")
	c.handler.Init()
	// Run the informer to list and watch resources
	go c.informer.Run(stopCh)
	go c.nodeInformer.Run(stopCh)

	// Synchronization to settle resources one
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced, c.nodeInformer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Error syncing cache"))
		return
	}
	c.logger.Info("run: cache sync complete")
	// Operate the runWorker
	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

// To process new objects added to the queue
func (c *controller) runWorker() {
	log.Info("runWorker: starting")
	// Run processNextItem for all the changes
	for c.processNextItem() {
		log.Info("runWorker: processing next item")
	}

	log.Info("runWorker: completed")
}

// This function deals with the queue and sends each item in it to the specified handler to be processed.
func (c *controller) processNextItem() bool {
	log.Info("processNextItem: start")
	// Fetch the next item of the queue
	event, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(event)
	// Get the key string
	keyRaw := event.(informerevent).key
	// Use the string key to get the object from the indexer
	item, exists, err := c.informer.GetIndexer().GetByKey(keyRaw)
	if err != nil {
		if c.queue.NumRequeues(event.(informerevent).key) < 5 {
			c.logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, retrying", event.(informerevent).key, err)
			c.queue.AddRateLimited(event.(informerevent).key)
		} else {
			c.logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, no more retries", event.(informerevent).key, err)
			c.queue.Forget(event.(informerevent).key)
			utilruntime.HandleError(err)
		}
	}

	if !exists {
		if event.(informerevent).function == delete {
			c.logger.Infof("Controller.processNextItem: object deleted detected: %s", keyRaw)
			c.handler.ObjectDeleted(item)
		}
	} else {
		if event.(informerevent).function == create {
			c.logger.Infof("Controller.processNextItem: object created detected: %s", keyRaw)
			c.handler.ObjectCreated(item)
		} else if event.(informerevent).function == update {
			c.logger.Infof("Controller.processNextItem: object updated detected: %s", keyRaw)
			c.handler.ObjectUpdated(item)
		}
	}
	c.queue.Forget(event.(informerevent).key)

	return true
}