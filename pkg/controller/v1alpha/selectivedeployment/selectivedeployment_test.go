package selectivedeployment

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Dictionary for status messages
var errorDict = map[string]string{
	"k8-sync":     "Kubernetes clientset sync problem",
	"edgnet-sync": "EdgeNet clientset sync problem",
}

type SDTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	client        kubernetes.Interface
	nodeObj       corev1.Node
	userObj       apps_v1alpha.User
	sdObj         apps_v1alpha.SelectiveDeployment
	edgenetclient versioned.Interface
	handler       SDHandler
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

// Init syncs the test group
func (g *SDTestGroup) Init() {
	sdObj := apps_v1alpha.SelectiveDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SelectiveDeployment",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			ClusterName: "Edgenet",
			Name:        "edgenetSD",
			Namespace:   "authority-edgenet",
		},
		Spec: apps_v1alpha.SelectiveDeploymentSpec{
			Controllers: apps_v1alpha.Controllers{
				Deployment: []v1.Deployment{v1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "authority-edgenet",
						Name:      "deploymentTest",
						UID:       "UIDTest",
					},
					Status: v1.DeploymentStatus{
						Conditions: []v1.DeploymentCondition{
							v1.DeploymentCondition{
								Type:   success,
								Status: "true",
							},
						},
					},
					Spec: v1.DeploymentSpec{
						// Selector: &metav1.LabelSelector{
						// 	MatchLabels: map[string]string{"testLabel": "testLabel"},
						// },
					},
				},
				},
				// DaemonSet: v1.DaemonSet{
				// TypeMeta: metav1.TypeMeta{
				// 	Kind:       "DeamonSet",
				// 	APIVersion: "apps/v1",
				// },
				// ObjectMeta: metav1.ObjectMeta{
				// 	Namespace: "authority-edgenet",
				// 	Name:      "TestDeamonSet",
				// },
				// Spec: v1.DaemonSetSpec{
				// },
				// Status: v1.DaemonSetStatus{
				// },
				// },
			},

			Selector: []apps_v1alpha.Selector{
				apps_v1alpha.Selector{
					Name:     "city",
					Value:    []string{"Islamabad"},
					Operator: "In",
				},
			},
		},
	}
	nodeObj := corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenetNode",
			// Namespace: "authority-edgnet",
			Labels: map[string]string{"kubernetes.io/hostname": "nodeHostName", "edge-net.io/city": "Islamabad", "edge-net.io/lon": "11.11", "edge-net.io/lat": "22.22"},
		},
		Spec: corev1.NodeSpec{
			Unschedulable: false,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				corev1.NodeCondition{
					Type:   "Ready",
					Status: "True",
				},
			},
		},
	}
	authorityObj := apps_v1alpha.Authority{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Authority",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: apps_v1alpha.AuthoritySpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: apps_v1alpha.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: apps_v1alpha.Contact{
				Email:     "unittest@edge-net.org",
				FirstName: "unit",
				LastName:  "testing",
				Phone:     "+33NUMBER",
				Username:  "unittesting",
			},
			Enabled: false,
		},
	}
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "unittesting",
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "EdgeNet",
			LastName:  "EdgeNet",
			Email:     "unittest@edge-net.org",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			Type:  "Admin",
			State: success,
		},
	}
	g.nodeObj = nodeObj
	g.authorityObj = authorityObj
	g.userObj = userObj
	g.sdObj = sdObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	// Create Authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	g.authorityObj.Status.State = success
	g.authorityObj.Spec.Enabled = true
	// Update Authority status
	g.edgenetclient.AppsV1alpha().Authorities().UpdateStatus(g.authorityObj.DeepCopy())
	authorityChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", g.authorityObj.GetName())}}
	// Create Authority child namepace
	g.client.CoreV1().Namespaces().Create(authorityChildNamespace)
}

// TestHandlerInit for handler initialization
func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := SDTestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)
	if g.handler.clientset != g.client {
		t.Error(errorDict["k8-sync"])
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error(errorDict["edgenet-sync"])
	}
}

func TestObjectCreated(t *testing.T) {
	g := SDTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	g.client.CoreV1().Nodes().Create(g.nodeObj.DeepCopy())
	t.Run("creation of SD", func(t *testing.T) {
		g.handler.ObjectCreated(g.sdObj.DeepCopy())
	})
}
