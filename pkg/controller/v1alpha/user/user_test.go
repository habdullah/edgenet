package user

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"
	"edgenet/pkg/controller/v1alpha/authority"

	"github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

//The main structure of test group
type UserTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	teamList      apps_v1alpha.TeamList
	sliceList     apps_v1alpha.SliceList
	userObj       apps_v1alpha.User
	client        kubernetes.Interface
	edgenetclient versioned.Interface
	handler       Handler
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

//Init syncs the test group
func (g *UserTestGroup) Init() {

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
	teamList := apps_v1alpha.TeamList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TeamList",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ListMeta: metav1.ListMeta{
			SelfLink:        "teamSelfLink",
			ResourceVersion: "1",
		},
		Items: []apps_v1alpha.Team{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Team",
					APIVersion: "apps.edgenet.io/v1alpha",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "edgenetTeam",
				},
				Spec: apps_v1alpha.TeamSpec{
					Users: []apps_v1alpha.TeamUsers{
						{
							Authority: "authority-edgenet",
							Username:  "unittestingTeamObj",
						},
					},
					Description: "This is a Teamtest description",
					Enabled:     false,
				},
			},
		},
	}
	sliceList := apps_v1alpha.SliceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SliceList",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ListMeta: metav1.ListMeta{
			SelfLink:        "sliceSelfLink",
			ResourceVersion: "1",
		},
		Items: []apps_v1alpha.Slice{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Slice",
					APIVersion: "apps.edgenet.io/v1alpha",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "edgenetSlice",
				},
				Spec: apps_v1alpha.SliceSpec{
					Users: []apps_v1alpha.SliceUsers{
						{
							Authority: "authority-edgenet",
							Username:  "unittestingSliceObj",
						},
					},
					Description: "This is a Slicetest description",
					Renew:       false,
				},
			},
		},
	}

	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "unittestingObj",
			Namespace:  "authority-edgenet",
			UID:        "TestUID",
			Generation: 1,
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "EdgeNetFirstName",
			LastName:  "EdgeNetLastName",
			Email:     "userObj@email.com",
		},
		Status: apps_v1alpha.UserStatus{
			State: success,
			AUP:   true,
		},
	}
	g.authorityObj = authorityObj
	g.teamList = teamList
	g.sliceList = sliceList
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	// Invoke authority ObjectCreated to create namespace
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
}

// TestHandlerInit for handler initialization
func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := UserTestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)

	if g.handler.clientset != g.client {
		t.Error("Kubernetes clientset sync problem")
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error("EdgeNet clientset sync problem")
	}
}

func TestUserCreate(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	t.Run("Creation of user from authority", func(t *testing.T) {
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
		if user == nil {
			t.Error("\nUser generation failed when an authority created\n")
		}
	})
	t.Run("Creation of user\n", func(t *testing.T) {
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
		g.handler.ObjectCreated(g.userObj.DeepCopy())
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.GetName(), metav1.GetOptions{})
		user.Spec.Active = true
		var field fields
		field.active = true
		g.handler.ObjectUpdated(user.DeepCopy(), field)
		currentUserRole, _ := g.handler.clientset.RbacV1().Roles(user.GetNamespace()).Get(fmt.Sprintf("user-%s", user.GetName()), metav1.GetOptions{})
		if currentUserRole == nil {
			t.Error("User role cannot be created")
		}
		roleBinding, _ := g.handler.clientset.RbacV1().RoleBindings(user.GetNamespace()).Get(fmt.Sprintf("%s-user-%s", user.GetNamespace(), user.GetName()), metav1.GetOptions{})
		if roleBinding == nil {
			t.Error("RoleBinding Creation failed")
		}
	})
	t.Run("Check dublicate object", func(t *testing.T) {
		// Change the user object name to make comparison with the user-created above
		g.userObj.Name = "different"
		g.userObj.UID = "differentUID"
		// Create user
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
		// Invoke the ObjectCreated()
		g.handler.ObjectCreated(g.userObj.DeepCopy())
		// Check if the user created successfully or not
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.Name, metav1.GetOptions{})
		if user.Status.Message == nil {
			t.Error("Duplicate value cannot be detected")
		}
	})

}

func TestUserUpdate(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	t.Run("Updating Email and Checking the Duplication", func(t *testing.T) {
		// Creating a user
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
		g.handler.ObjectCreated(g.userObj.DeepCopy())
		// Changing the user email as the same as the authority default created user
		g.userObj.Spec.Email = "unittest@edge-net.org"
		var field fields
		g.handler.ObjectUpdated(g.userObj.DeepCopy(), field)
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.Name, metav1.GetOptions{})
		if user.Status.Message == nil {
			t.Error("Duplicate value cannot be detected")
		}
		if user.Spec.Active {
			t.Error("User cannot be deactivated")
		}
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(g.userObj.Name, &metav1.DeleteOptions{})
	})
	t.Run("Updating Email", func(t *testing.T) {
		// Creating a user
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
		g.handler.ObjectCreated(g.userObj.DeepCopy())
		// Updateing the email
		g.userObj.Spec.Email = "NewUserObj@email.com"
		var field fields
		field.email = true
		g.handler.ObjectUpdated(g.userObj.DeepCopy(), field)
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.Name, metav1.GetOptions{})
		if user.Spec.Email != "NewUserObj@email.com" {
			t.Error("Updating Email Failed")
		}
		if user.Spec.Active != false {
			t.Error("User is still Active after changing its email!")
		}
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(g.userObj.Name, &metav1.DeleteOptions{})
	})
}

// func TestSetEmailVerification(t *testing.T) {
// 	g := UserTestGroup{}
// 	g.Init()
// 	g.handler.Init(g.client, g.edgenetclient)

// 	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
// 	result := g.handler.setEmailVerification(user, fmt.Sprintf("authority-%s", g.authorityObj.GetName()))
// 	if result != nil {
// 		t.Error("user-email-verification-update-malfunction")
// 	}
// }

func TestCreateRoleBindings(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creating User from userObj
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
	g.handler.ObjectCreated(g.userObj.DeepCopy())
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.Name, metav1.GetOptions{})
	// Invoking createRoleBindings
	g.handler.createRoleBindings(user.DeepCopy(), g.sliceList.DeepCopy(), g.teamList.DeepCopy(), fmt.Sprintf("authority-%s", g.authorityObj.GetName()))
	// Check the creation of use role Binding
	roleBindings, _ := g.handler.clientset.RbacV1().RoleBindings(user.GetNamespace()).Get(fmt.Sprintf("%s-user-%s", user.GetNamespace(), user.GetName()), metav1.GetOptions{})
	if roleBindings == nil {
		t.Error("RoleBinding Creation failed")
	}
}

func TestDeleteRoleBindings(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	// Creating User from userObj
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
	g.handler.ObjectCreated(g.userObj.DeepCopy())
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.Name, metav1.GetOptions{})
	// Invoking createRoleBindings
	g.handler.createRoleBindings(user, g.sliceList.DeepCopy(), g.teamList.DeepCopy(), fmt.Sprintf("authority-%s", g.authorityObj.GetName()))
	// Check the creation of use role Binding
	roleBindings, _ := g.handler.clientset.RbacV1().RoleBindings(user.GetNamespace()).Get(fmt.Sprintf("%s-user-%s", user.GetNamespace(), user.GetName()), metav1.GetOptions{})
	if roleBindings == nil {
		t.Error("RoleBinding Creation failed")
	}
	g.handler.deleteRoleBindings(user, g.sliceList.DeepCopy(), g.teamList.DeepCopy())
	roleBindingsResult, _ := g.handler.clientset.RbacV1().RoleBindings(user.GetNamespace()).Get(fmt.Sprintf("%s-user-%s", user.GetNamespace(), user.GetName()), metav1.GetOptions{})
	if roleBindingsResult != nil {
		t.Error("RoleBinding Deletion failed")
	}
}

func TestCreateAUPRoleBinding(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
	g.handler.ObjectCreated(g.userObj.DeepCopy())
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.GetName(), metav1.GetOptions{})
	err := g.handler.createAUPRoleBinding(user)
	if err != nil {
		t.Error("Create AUPRoleBinding failed")
	} else {
		t.Logf("\nPassed result=%v\n", err)
	}
}

// func TestGenerateRandomString(t *testing.T) {
// 	for i := 1; i < 5; i++ {
// 		origin := generateRandomString(16)
// 		time.Sleep(1 * time.Microsecond)
// 		test := generateRandomString(16)
// 		if origin == test {
// 			t.Error("User GenerateRadnomString failed")
// 		}
// 	}
// }