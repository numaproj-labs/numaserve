/*
Copyright 2023 sarabala1979.

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

package v1alpha1

import (
	"github.com/numaproj/numaflow/pkg/apis/numaflow/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Step struct {
	Name         string `json:"name"`
	v1alpha1.UDF `json:",inline"`
	Condition    string `json:"condition,omitempty"`
}

type S3ModelStore struct {
	Host            string               `json:"host"`
	AccessKeyID     v1.SecretKeySelector `json:"accessKeyID,omitempty"`
	SecretAccessKey v1.SecretKeySelector `json:"secretAccessKey,omitempty"`
	Bucket          string               `json:"bucket,omitempty"`
	ObjectKey       string               `json:"objectKey,omitempty"`
	OutputPath      string               `json:"outputPath,omitempty"`
}

type ModelStore struct {
	S3 S3ModelStore `json:"s3,omitempty"`
}

type Model struct {
	ModelFormat string     `json:"modelFormat"`
	ModelStore  ModelStore `json:"modelStore"`
}

type Predictor struct {
	Name      string `json:"name"`
	Model     Model  `json:"model"`
	Condition string `json:"condition,omitempty"`
}

type Predictors []Predictor

// type ParallelPredictors []Predictors
type Steps []Step

//type ParallelSteps []Steps

// MLInferenceSpec defines the desired state of MLInference
type MLInferenceSpec struct {
	InferenceType string     `json:"inferenceType"`
	PreSteps      Steps      `json:"preSteps,omitempty"`
	Predictors    Predictors `json:"predictor"`
	PostSteps     Steps      `json:"postSteps,omitempty"`
}

type Status struct {
	Phase   string `json:"phase,omitempty"`
	Message string `json:"message,omitempty"`
}

type ServingStatus struct {
	FrontlineService *Status `json:"frontlineService,omitempty"`
	Pipeline         *Status `json:"pipeline,omitempty"`
}

// MLInferenceStatus defines the observed state of MLInference
type MLInferenceStatus struct {
	ServingURL         string         `json:"servingURL,omitempty"`
	ServingServiceName string         `json:"servingServiceName,omitempty"`
	ComponentStatus    *ServingStatus `json:"componentStatus,omitempty"`
	Phase              string         `json:"phase,omitempty"` //Running/Failed/Degraded/Error/Pending
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MLInference is the Schema for the mlinferences API
type MLInference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MLInferenceSpec `json:"spec,omitempty"`

	Status MLInferenceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MLInferenceList contains a list of MLInference
type MLInferenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MLInference `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MLInference{}, &MLInferenceList{})
}
