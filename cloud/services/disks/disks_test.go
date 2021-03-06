/*
Copyright 2020 The Kubernetes Authors.

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

package disks

import (
	"context"
	"net/http"
	"testing"

	"sigs.k8s.io/cluster-api-provider-azure/cloud/services/disks/mock_disks"

	"github.com/Azure/go-autorest/autorest"
	"github.com/golang/mock/gomock"

	network "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	infrav1 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-azure/cloud/scope"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	clusterv1.AddToScheme(scheme.Scheme)
}

func TestInvalidDiskSpec(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	disksMock := mock_disks.NewMockClient(mockCtrl)

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
	}

	client := fake.NewFakeClient(cluster)

	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		AzureClients: scope.AzureClients{
			SubscriptionID: "123",
			Authorizer:     autorest.NullAuthorizer{},
		},
		Client:  client,
		Cluster: cluster,
		AzureCluster: &infrav1.AzureCluster{
			Spec: infrav1.AzureClusterSpec{
				Location: "test-location",
				ResourceGroup: "my-rg",
				NetworkSpec: infrav1.NetworkSpec{
					Vnet: infrav1.VnetSpec{Name: "my-vnet", ResourceGroup: "my-rg"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test context: %v", err)
	}

	s := &Service{
		Scope:  clusterScope,
		Client: disksMock,
	}

	// Wrong Spec
	wrongSpec := &network.PublicIPAddress{}

	err = s.Delete(context.TODO(), &wrongSpec)
	if err == nil {
		t.Fatalf("it should fail")
	}
	if err.Error() != "invalid disk specification" {
		t.Fatalf("got an unexpected error: %v", err)
	}
}

func TestDeleteDisk(t *testing.T) {
	testcases := []struct {
		name          string
		disksSpec     Spec
		expectedError string
		expect        func(m *mock_disks.MockClientMockRecorder)
	}{
		{
			name: "delete the disk",
			disksSpec: Spec{
				Name: "my-disk",
			},
			expectedError: "",
			expect: func(m *mock_disks.MockClientMockRecorder) {
				m.Delete(context.TODO(), "my-rg", "my-disk")
			},
		},
		{
			name: "disk already deleted",
			disksSpec: Spec{
				Name: "my-disk",
			},
			expectedError: "",
			expect: func(m *mock_disks.MockClientMockRecorder) {
				m.Delete(context.TODO(), "my-rg", "my-disk").Return(autorest.NewErrorWithResponse("", "", &http.Response{StatusCode: 404}, "Not Found"))
			},
		},
		{
			name: "error while trying to delete the disk",
			disksSpec: Spec{
				Name: "my-disk",
			},
			expectedError: "failed to delete disk my-disk in resource group my-rg: #: Internal Server Error: StatusCode=500",
			expect: func(m *mock_disks.MockClientMockRecorder) {
				m.Delete(context.TODO(), "my-rg", "my-disk").Return(autorest.NewErrorWithResponse("", "", &http.Response{StatusCode: 500}, "Internal Server Error"))
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			disksMock := mock_disks.NewMockClient(mockCtrl)

			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
			}

			client := fake.NewFakeClient(cluster)

			tc.expect(disksMock.EXPECT())

			clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
				AzureClients: scope.AzureClients{
					SubscriptionID: "123",
					Authorizer:     autorest.NullAuthorizer{},
				},
				Client:  client,
				Cluster: cluster,
				AzureCluster: &infrav1.AzureCluster{
					Spec: infrav1.AzureClusterSpec{
						Location: "test-location",
						ResourceGroup: "my-rg",
					},
				},
			})
			if err != nil {
				t.Fatalf("Failed to create test context: %v", err)
			}

			s := &Service{
				Scope:  clusterScope,
				Client: disksMock,
			}

			if err := s.Delete(context.TODO(), &tc.disksSpec); err != nil {
				if tc.expectedError == "" || err.Error() != tc.expectedError {
					t.Fatalf("got an unexpected error: %v", err)
				}
			} else {
				if tc.expectedError != "" {
					t.Fatalf("expected an error: %v", tc.expectedError)

				}
			}
		})
	}
}
