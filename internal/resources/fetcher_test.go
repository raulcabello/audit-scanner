package resources

import (
	"context"
	"github.com/google/go-cmp/cmp"
	policiesv1 "github.com/kubewarden/kubewarden-controller/pkg/apis/policies/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"testing"
)

const (
	kubewardenPoliciesGroup   = "policies.kubewarden.io"
	kubewardenPoliciesVersion = "v1"
)

// policies for testing
var p1 = policiesv1.AdmissionPolicy{
	Spec: policiesv1.AdmissionPolicySpec{PolicySpec: policiesv1.PolicySpec{
		Rules: []admissionregistrationv1.RuleWithOperations{{
			Operations: nil,
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{""},
				APIVersions: []string{"v1"},
				Resources:   []string{"pods"},
			},
		},
		},
	}},
}

var p2 = policiesv1.ClusterAdmissionPolicy{
	Spec: policiesv1.ClusterAdmissionPolicySpec{PolicySpec: policiesv1.PolicySpec{
		Rules: []admissionregistrationv1.RuleWithOperations{{
			Operations: nil,
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{"", "apps"},
				APIVersions: []string{"v1", "alphav1"},
				Resources:   []string{"pods", "deployments"},
			},
		},
		},
	}},
}

var p3 = policiesv1.AdmissionPolicy{
	Spec: policiesv1.AdmissionPolicySpec{PolicySpec: policiesv1.PolicySpec{
		Rules: []admissionregistrationv1.RuleWithOperations{{
			Operations: nil,
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{"", "apps"},
				APIVersions: []string{"v1"},
				Resources:   []string{"pods", "deployments"},
			},
		},
		},
	}},
}

func TestGetResourcesForPolicies(t *testing.T) {
	pod1 := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podDefault",
			Namespace: "default",
		},
		Spec: v1.PodSpec{},
	}
	pod2 := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podKubewarden",
			Namespace: "kubewarden",
		},
		Spec: v1.PodSpec{},
	}
	deployment1 := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploymentDefault",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{},
	}
	customScheme := scheme.Scheme
	customScheme.AddKnownTypes(schema.GroupVersion{Group: kubewardenPoliciesGroup, Version: kubewardenPoliciesVersion}, &policiesv1.ClusterAdmissionPolicy{}, &policiesv1.AdmissionPolicy{}, &policiesv1.ClusterAdmissionPolicyList{}, &policiesv1.AdmissionPolicyList{})
	metav1.AddToGroupVersion(customScheme, schema.GroupVersion{Group: kubewardenPoliciesGroup, Version: kubewardenPoliciesVersion})

	dynamicClient := fake.NewSimpleDynamicClient(customScheme, &p1, &pod1, &pod2, &deployment1)

	unstructuredPod1 := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":              "podDefault",
			"namespace":         "default",
			"creationTimestamp": nil,
		},
		"spec": map[string]interface{}{
			"containers": nil,
		},
		"status": map[string]interface{}{},
	}

	expectedP1 := []AuditableResources{{
		Policies:  []policiesv1.Policy{&p1},
		Resources: []unstructured.Unstructured{{Object: unstructuredPod1}},
	}}

	fetcher := Fetcher{dynamicClient}

	tests := []struct {
		name     string
		policies []policiesv1.Policy
		expect   []AuditableResources
	}{
		{"policy1 (just pods)", []policiesv1.Policy{&p1}, expectedP1},
		{"no policies", []policiesv1.Policy{}, []AuditableResources{}},
	}

	for _, test := range tests {
		ttest := test
		t.Run(ttest.name, func(t *testing.T) {
			resources, err := fetcher.GetResourcesForPolicies(context.Background(), ttest.policies, "default")
			if err != nil {
				t.Errorf("error shouldn't have happend " + err.Error())
			}
			if !cmp.Equal(resources, ttest.expect) {
				t.Errorf("expected %v, but got %v", ttest.expect, resources)
			}
		})
	}
}

func TestCreateGVRPolicyMap(t *testing.T) {

	// all posible combination of GVR (Group, Version, Resource) for p1, p2 and p3
	gvr1 := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}
	gvr2 := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "deployments",
	}
	gvr3 := schema.GroupVersionResource{
		Group:    "",
		Version:  "alphav1",
		Resource: "pods",
	}
	gvr4 := schema.GroupVersionResource{
		Group:    "",
		Version:  "alphav1",
		Resource: "deployments",
	}
	gvr5 := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "pods",
	}
	gvr6 := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	gvr7 := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "alphav1",
		Resource: "pods",
	}
	gvr8 := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "alphav1",
		Resource: "deployments",
	}

	// expected outcome

	expectedP1andP2 := make(map[schema.GroupVersionResource][]policiesv1.Policy)

	expectedP1andP2[gvr1] = []policiesv1.Policy{&p1, &p2}
	expectedP1andP2[gvr2] = []policiesv1.Policy{&p2}
	expectedP1andP2[gvr3] = []policiesv1.Policy{&p2}
	expectedP1andP2[gvr4] = []policiesv1.Policy{&p2}
	expectedP1andP2[gvr5] = []policiesv1.Policy{&p2}
	expectedP1andP2[gvr6] = []policiesv1.Policy{&p2}
	expectedP1andP2[gvr7] = []policiesv1.Policy{&p2}
	expectedP1andP2[gvr8] = []policiesv1.Policy{&p2}

	expectedP1P2andP3 := make(map[schema.GroupVersionResource][]policiesv1.Policy)

	expectedP1P2andP3[gvr1] = []policiesv1.Policy{&p1, &p2, &p3}
	expectedP1P2andP3[gvr2] = []policiesv1.Policy{&p2, &p3}
	expectedP1P2andP3[gvr3] = []policiesv1.Policy{&p2}
	expectedP1P2andP3[gvr4] = []policiesv1.Policy{&p2}
	expectedP1P2andP3[gvr5] = []policiesv1.Policy{&p2, &p3}
	expectedP1P2andP3[gvr6] = []policiesv1.Policy{&p2, &p3}
	expectedP1P2andP3[gvr7] = []policiesv1.Policy{&p2}
	expectedP1P2andP3[gvr8] = []policiesv1.Policy{&p2}

	expectedP1andP3 := make(map[schema.GroupVersionResource][]policiesv1.Policy)

	expectedP1andP3[gvr1] = []policiesv1.Policy{&p1, &p3}
	expectedP1andP3[gvr2] = []policiesv1.Policy{&p3}
	expectedP1andP3[gvr5] = []policiesv1.Policy{&p3}
	expectedP1andP3[gvr6] = []policiesv1.Policy{&p3}

	expectedP1 := make(map[schema.GroupVersionResource][]policiesv1.Policy)

	expectedP1[gvr1] = []policiesv1.Policy{&p1}

	tests := []struct {
		name     string
		policies []policiesv1.Policy
		expect   map[schema.GroupVersionResource][]policiesv1.Policy
	}{
		{"policy1 (just pods) and policy2 (pods, deployments, v1 and alphav1)", []policiesv1.Policy{&p1, &p2}, expectedP1andP2},
		{"policy1 (just pods), policy2 (pods, deployments, v1 and alphav1) and policy3 (pods, deployments, v1)", []policiesv1.Policy{&p1, &p2, &p3}, expectedP1P2andP3},
		{"policy1 (just pods) and policy3 (pods, deployments, v1)", []policiesv1.Policy{&p1, &p3}, expectedP1andP3},
		{"policy1 (just pods)", []policiesv1.Policy{&p1}, expectedP1},
		{"empty array", []policiesv1.Policy{}, make(map[schema.GroupVersionResource][]policiesv1.Policy)},
	}

	for _, test := range tests {
		ttest := test
		t.Run(ttest.name, func(t *testing.T) {
			gvrPolicyMap := createGVRPolicyMap(ttest.policies)
			if !cmp.Equal(gvrPolicyMap, ttest.expect) {
				t.Errorf("expected %v, but got %v", ttest.expect, gvrPolicyMap)
			}
		})
	}

}
