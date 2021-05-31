/*


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

package v1

import (
	nginxv1alpha1 "github.com/tsuru/nginx-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WafSpec defines the desired state of Waf
type WafSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Plan is the name of the wafplan instance.
	// +optional
	WafPlanName string `json:"planName"`

	Rules Rules `json:"rules,omitempty"`

	// Bind is the app bounded to the instance
	Bind Bind `json:"bind,omitempty"`

	// Service to expose the nginx instance
	// +optional
	Service *nginxv1alpha1.NginxService `json:"service,omitempty"`

	// ExtraFiles points to a ConfigMap where the files are stored.
	// +optional
	ExtraFiles *nginxv1alpha1.FilesRef `json:"extraFiles,omitempty"`
}

type Bind struct {
	Name     string `json:"name,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

// WafStatus defines the observed state of Waf
type WafStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Revision hash calculated for the current spec.
	WantedNginxRevisionHash string `json:"wantedNginxRevisionHash,omitempty"`

	// The revision hash observed by the controller in the nginx object.
	ObservedNginxRevisionHash string `json:"observedNginxRevisionHash,omitempty"`

	// PodSelector is the NGINX's pod label selector.
	PodSelector string `json:"podSelector,omitempty"`

	// The most recent generation observed by the rpaas operator controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// CurrentReplicas is the last observed number of pods.
	CurrentReplicas int32 `json:"currentReplicas,omitempty"`

	// NginxUpdated is true if the wanted nginx revision hash equals the
	// observed nginx revision hash.
	NginxUpdated bool `json:"nginxUpdated"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.currentReplicas,selectorpath=.status.podSelector

// Waf is the Schema for the wafs API
type Waf struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WafSpec   `json:"spec,omitempty"`
	Status WafStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WafList contains a list of Waf
type WafList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Waf `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Waf{}, &WafList{})
}
