package emailverification

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"
	"edgenet/pkg/controller/v1alpha/authority"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Constant variables for events
const success = "Successful"

// The main structure of test group
type EVTestGroup struct {
	authorityObj        apps_v1alpha.Authority
	EVObj               apps_v1alpha.EmailVerification
	authorityRequestObj apps_v1alpha.AuthorityRequest
	client              kubernetes.Interface
	edgenetclient       versioned.Interface
	handler             Handler
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

// Init syncs the test group
func (g *EVTestGroup) Init() {
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
			Enabled: true,
		},
	}
	EVObj := apps_v1alpha.EmailVerification{
		TypeMeta: metav1.TypeMeta{
			Kind:       "emailVerification",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenetEV",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.EmailVerificationSpec{
			Kind:       "",
			Identifier: "",
			Verified:   false,
		},
	}
	g.authorityObj = authorityObj
	g.EVObj = EVObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	// Create Authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	//invoke ObjectCreated to create namespace
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
}

func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := EVTestGroup{}
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

func TestEVCreate(t *testing.T) {
	g := EVTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creation of Email verification obj
	t.Run("creation of Email verification", func(t *testing.T) {
		g.edgenetclient.AppsV1alpha().EmailVerifications(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.EVObj.DeepCopy())
		g.handler.ObjectCreated(g.EVObj.DeepCopy())
		// Handler will update expiration time
		EV, _ := g.edgenetclient.AppsV1alpha().EmailVerifications(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.EVObj.GetName(), metav1.GetOptions{})
		if EV.Status.Expires == nil {
			t.Error(errorDict["EV-create"])
		}
	})
	t.Run("creation of Email verification already verified", func(t *testing.T) {
		g.EVObj.Spec.Verified = true
		g.edgenetclient.AppsV1alpha().EmailVerifications(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.EVObj.DeepCopy())
		g.handler.ObjectCreated(g.EVObj.DeepCopy())
		// Handler will delete EV if verified
		EV, _ := g.edgenetclient.AppsV1alpha().EmailVerifications(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.EVObj.GetName(), metav1.GetOptions{})
		if EV != nil {
			t.Error(errorDict["EV-del-fail"])
		}
	})
}

func TestEVUpdate(t *testing.T) {
	g := EVTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creation of Email verification obj
	g.edgenetclient.AppsV1alpha().EmailVerifications(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.EVObj.DeepCopy())
	g.handler.ObjectCreated(g.EVObj.DeepCopy())
	t.Run("Update of Email verification", func(t *testing.T) {
		g.EVObj.Spec.Verified = true
		var field fields
		field.kind = true
		g.handler.ObjectUpdated(g.EVObj.DeepCopy(), field)
		// Handler will delete EV if verified
		EV, _ := g.edgenetclient.AppsV1alpha().EmailVerifications(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.EVObj.GetName(), metav1.GetOptions{})
		if EV != nil {
			t.Error(errorDict["EV-del-fail"])
		}
	})
}
