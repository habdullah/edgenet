/*
Copyright The Kubernetes Authors.

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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha

import (
	v1alpha "edgenet/pkg/apis/apps/v1alpha"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// UserRegistrationRequestLister helps list UserRegistrationRequests.
type UserRegistrationRequestLister interface {
	// List lists all UserRegistrationRequests in the indexer.
	List(selector labels.Selector) (ret []*v1alpha.UserRegistrationRequest, err error)
	// UserRegistrationRequests returns an object that can list and get UserRegistrationRequests.
	UserRegistrationRequests(namespace string) UserRegistrationRequestNamespaceLister
	UserRegistrationRequestListerExpansion
}

// userRegistrationRequestLister implements the UserRegistrationRequestLister interface.
type userRegistrationRequestLister struct {
	indexer cache.Indexer
}

// NewUserRegistrationRequestLister returns a new UserRegistrationRequestLister.
func NewUserRegistrationRequestLister(indexer cache.Indexer) UserRegistrationRequestLister {
	return &userRegistrationRequestLister{indexer: indexer}
}

// List lists all UserRegistrationRequests in the indexer.
func (s *userRegistrationRequestLister) List(selector labels.Selector) (ret []*v1alpha.UserRegistrationRequest, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha.UserRegistrationRequest))
	})
	return ret, err
}

// UserRegistrationRequests returns an object that can list and get UserRegistrationRequests.
func (s *userRegistrationRequestLister) UserRegistrationRequests(namespace string) UserRegistrationRequestNamespaceLister {
	return userRegistrationRequestNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// UserRegistrationRequestNamespaceLister helps list and get UserRegistrationRequests.
type UserRegistrationRequestNamespaceLister interface {
	// List lists all UserRegistrationRequests in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha.UserRegistrationRequest, err error)
	// Get retrieves the UserRegistrationRequest from the indexer for a given namespace and name.
	Get(name string) (*v1alpha.UserRegistrationRequest, error)
	UserRegistrationRequestNamespaceListerExpansion
}

// userRegistrationRequestNamespaceLister implements the UserRegistrationRequestNamespaceLister
// interface.
type userRegistrationRequestNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all UserRegistrationRequests in the indexer for a given namespace.
func (s userRegistrationRequestNamespaceLister) List(selector labels.Selector) (ret []*v1alpha.UserRegistrationRequest, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha.UserRegistrationRequest))
	})
	return ret, err
}

// Get retrieves the UserRegistrationRequest from the indexer for a given namespace and name.
func (s userRegistrationRequestNamespaceLister) Get(name string) (*v1alpha.UserRegistrationRequest, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha.Resource("userregistrationrequest"), name)
	}
	return obj.(*v1alpha.UserRegistrationRequest), nil
}
