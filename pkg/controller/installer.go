package controller

import (
	"context"
	"fmt"
	"github.com/numaproj-labs/numaserve/api/v1alpha1"
	numaflowV1alpha1 "github.com/numaproj/numaflow/pkg/apis/numaflow/v1alpha1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/cnf/structhash"
	dfv1 "github.com/numaproj-labs/numaserve/pkg/apis/numaflow/v1alpha1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (mlictx *MLInferenceOperatorContext) Reconcile() error {

	mlictx.logger.Info("Reconciling MLInference")

	// for each resource type:
	// look for object of that type with a label that matches this inference object name (in the same namespace)
	// if it doesn't exist: get the spec and create it
	// else Update existing

	// todo: any way to reduce the repetition of all of the logic?
	var err error
	// Pipeline:
	err = mlictx.reconcileISBSvc()
	if err != nil {
		return fmt.Errorf("error reconciling ISB Svc: %v", err)
	}
	err = mlictx.reconcilePipeline()
	if err != nil {
		return fmt.Errorf("error reconciling pipeline: %v", err)
	}
	err = mlictx.reconcileFrontlineService()
	if err != nil {
		return fmt.Errorf("error reconciling frontline service: %v", err)
	}
	err = mlictx.reconcileFrontlineApplication()
	if err != nil {
		return fmt.Errorf("error reconciling frontline application: %v", err)
	}

	// update Status based on ComponentStatus
	mlictx.propagateStatus()

	return nil
}

func (mlictx *MLInferenceOperatorContext) reconcilePipeline() error {

	var foundPipelines numaflowV1alpha1.PipelineList
	labels := ctrlClient.MatchingLabels{"mlinference": mlictx.mlInferenceObj.Name}
	mlictx.client.List(mlictx.ctx, &foundPipelines, labels)

	numFound := len(foundPipelines.Items)
	switch {
	case numFound == 0: // it doesn't exist: get the spec and create it

		newSpec, err := mlictx.createPipelineSpec(mlictx.mlInferenceObj, nil)
		if err != nil {
			return err
		}

		hash, err := structhash.Hash(newSpec.Spec.Vertices, 1)
		fmt.Println(hash)
		hash = hash[len(hash)-6:]
		fmt.Println(hash)
		if err != nil {
			return err
		}
		newSpec.Name = fmt.Sprintf("%s-%s", newSpec.Name, hash)
		fmt.Println(hash)
		fmt.Println("got here 1")

		mlictx.logger.Info("Creating Pipeline", "name", newSpec.Name, "namespace", newSpec.Namespace)
		err = mlictx.client.Create(mlictx.ctx, newSpec)
		if err != nil {
			return err
		}
		svcObj, err := mlictx.createPipelineServiceSpec(mlictx.mlInferenceObj)
		if err != nil {
			return err
		}
		err = mlictx.client.Create(mlictx.ctx, svcObj)
		//Todo Add already exist check
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				return nil
			}
			return err
		}
	case numFound == 1:
		mlictx.propagatePipelineStatus(foundPipelines.Items[0].Status)

		newSpec, err := mlictx.createPipelineSpec(mlictx.mlInferenceObj, &foundPipelines.Items[0])
		if err != nil {
			return err
		}

		newHash, err := structhash.Hash(newSpec.Spec.Vertices, 1)
		if err != nil {
			return err
		}
		newHash = newHash[len(newHash)-6:]
		oldHash, err := structhash.Hash(foundPipelines.Items[0].Spec.Vertices, 1)
		if err != nil {
			return err
		}
		oldHash = oldHash[len(oldHash)-6:]
		if newHash == oldHash {
			mlictx.logger.Info("There is no update")
			return nil
		} else {
			fmt.Println("not same")
			newHash = newHash[len(newHash)-6:]
			newSpec.Name = fmt.Sprintf("%s-%s", newSpec.Name, newHash)
			fmt.Println(newHash)
			fmt.Println("got here 1")
			mlictx.logger.Info("Creating Pipeline", "name", newSpec.Name, "namespace", newSpec.Namespace)
			fmt.Println("got here 2")
			newSpec.ResourceVersion = ""
			err = mlictx.client.Create(mlictx.ctx, newSpec)
			if err != nil {
				return err
			}

			mlictx.logger.Info("Starting Progressive rollout")
			go RolloutModel(foundPipelines.Items[0], mlictx.client)
			return nil
		}

		//mlictx.logger.Info("Updating Pipeline", "name", newSpec.Name, "namespace", newSpec.Namespace)
		//mlictx.client.Update(mlictx.ctx, newSpec)
		//if err != nil {
		//	mlictx.logger.Info("Updating Pipeline failed: will trying deleting/recreating", "err", err, "name", newSpec.Name, "namespace", newSpec.Namespace)
		//	err = mlictx.client.Delete(mlictx.ctx, &foundPipelines.Items[0]) //todo: any special DeleteOptions? need to verify this will block so we can call Create after
		//	if err != nil {
		//		return err
		//	}
		//	// create a new spec that doesn't have any of the history of the old one
		//	newSpec, err = mlictx.createPipelineSpec(mlictx.mlInferenceObj, nil)
		//	if err != nil {
		//		return err
		//	}
		//	err = mlictx.client.Create(mlictx.ctx, newSpec)
		//	if err != nil {
		//		return err
		//	}
		//}
	case numFound > 1: // todo: this could happen in the case of rolling out multiple
	}

	return nil
}

func (mlictx *MLInferenceOperatorContext) reconcileFrontlineService() error {

	// find Service by name
	var foundService corev1.Service
	namespacedName := k8stypes.NamespacedName{Namespace: mlictx.mlInferenceObj.Namespace, Name: frontLineServiceName(mlictx.mlInferenceObj.Name)}
	mlictx.mlInferenceObj.Status.ServingServiceName = frontLineServiceName(mlictx.mlInferenceObj.Name)
	mlictx.mlInferenceObj.Status.ServingURL = frontLineServiceURL(mlictx.mlInferenceObj.Name, mlictx.mlInferenceObj.Namespace)
	err := mlictx.client.Get(mlictx.ctx, namespacedName, &foundService)
	if err != nil {
		// doesn't exist: create a new one
		if apierrors.IsNotFound(err) {
			newSpec, err := mlictx.createFrontlineServiceSpec(mlictx.mlInferenceObj, nil)
			if err != nil {
				return err
			}

			mlictx.logger.Info("Creating Frontline Service", "name", newSpec.Name, "namespace", newSpec.Namespace)
			err = mlictx.client.Create(mlictx.ctx, newSpec)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// already exists; update it

		newSpec, err := mlictx.createFrontlineServiceSpec(mlictx.mlInferenceObj, &foundService)
		if err != nil {
			return err
		}

		mlictx.logger.Info("Updating Frontline Service", "name", newSpec.Name, "namespace", newSpec.Namespace)
		err = mlictx.client.Update(mlictx.ctx, newSpec)
		if err != nil {
			mlictx.logger.Info("Updating Frontline Service failed: will trying deleting/recreating", "err", err, "name", newSpec.Name, "namespace", newSpec.Namespace)
			err = mlictx.client.Delete(mlictx.ctx, &foundService) //todo: any special DeleteOptions? need to verify this will block so we can call Create after
			if err != nil {
				return err
			}
			newSpec, err = mlictx.createFrontlineServiceSpec(mlictx.mlInferenceObj, nil)
			if err != nil {
				return err
			}
			err = mlictx.client.Create(mlictx.ctx, newSpec)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func RolloutModel(oldPipeline numaflowV1alpha1.Pipeline, client client.Client) {
	var pipeline numaflowV1alpha1.Pipeline
	canDeletePipeline := false
	logger.Info("starting")

	namespacedName := k8stypes.NamespacedName{Namespace: oldPipeline.Namespace, Name: oldPipeline.Name}
	time.Sleep(60 * time.Second)
	for {
		logger.Info("starting")
		err := client.Get(context.Background(), namespacedName, &pipeline)

		if err != nil {
			logger.Error(err, "Error occurred during rollout")
		}
		var vtxs numaflowV1alpha1.VertexList
		labels := ctrlClient.MatchingLabels{"numaflow.numaproj.io/pipeline-name": oldPipeline.Name, "numaflow.numaproj.io/vertex-name": "http-in"}
		fmt.Println(labels)
		err = client.List(context.Background(), &vtxs, labels)
		if err != nil {
			logger.Error(err, "Error occurred during rollout")
		}
		fmt.Println(len(vtxs.Items))
		if len(vtxs.Items) == 0 {
			break
		}

		for idx, vtx := range pipeline.Spec.Vertices {
			if vtx.Name == "http-in" {
				replicas := vtxs.Items[0].Status.Replicas
				fmt.Println("Replicas", replicas)
				if vtxs.Items[0].Status.Replicas == 1 {
					canDeletePipeline = true
				}
				if vtx.Scale.Max == nil || *vtx.Scale.Max > 1 {
					vtx.Scale.Min = pointer.Int32(*vtx.Scale.Min - 1)
					vtx.Scale.Max = pointer.Int32(int32(replicas - 1))
					pipeline.Spec.Vertices[idx] = vtx
					fmt.Println("Replicas", *vtx.Scale.Max)
					break
				}

			}
		}
		time.Sleep(20 * time.Second)

		err = client.Update(context.Background(), &pipeline)
		if err != nil {
			logger.Error(err, "Error occurred during rollout")
		}
		if canDeletePipeline {
			logger.Info("Old pipeline is successfully deleted")
			err = client.Delete(context.Background(), &pipeline)
		}

	}

}

func (mlictx *MLInferenceOperatorContext) reconcileFrontlineApplication() error {

	// find any existing one by name
	var foundStatefulSet appv1.StatefulSet
	namespacedName := k8stypes.NamespacedName{Namespace: mlictx.mlInferenceObj.Namespace, Name: frontLineStatefulSetName(mlictx.mlInferenceObj.Name)}
	err := mlictx.client.Get(mlictx.ctx, namespacedName, &foundStatefulSet)
	if err != nil {
		// doesn't exist: create a new one
		if apierrors.IsNotFound(err) {

			newSpec, err := mlictx.createFrontlineApplication(mlictx.mlInferenceObj, nil)
			if err != nil {
				return err
			}

			mlictx.logger.Info("Creating Frontline Application", "name", newSpec.Name, "namespace", newSpec.Namespace)
			err = mlictx.client.Create(mlictx.ctx, newSpec)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// already exists; grab any status, and then try updating the spec

		mlictx.propagateStatefulSetStatus(foundStatefulSet.Status)

		// try updating the spec (which may or may not have changed)
		newSpec, err := mlictx.createFrontlineApplication(mlictx.mlInferenceObj, &foundStatefulSet)
		if err != nil {
			return err
		}

		mlictx.logger.Info("Updating Frontline Application", "name", newSpec.Name, "namespace", newSpec.Namespace)
		err = mlictx.client.Update(mlictx.ctx, newSpec)
		if err != nil {

			mlictx.logger.Info("Updating Frontline Application failed: will trying deleting/recreating", "err", err, "name", newSpec.Name, "namespace", newSpec.Namespace)
			err = mlictx.client.Delete(mlictx.ctx, &foundStatefulSet) //todo: any special DeleteOptions? need to verify this will block so we can call Create after
			if err != nil {
				return err
			}

			newSpec, err = mlictx.createFrontlineApplication(mlictx.mlInferenceObj, nil)
			if err != nil {
				return err
			}

			err = mlictx.client.Create(mlictx.ctx, newSpec)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func (mlictx *MLInferenceOperatorContext) reconcileISBSvc() error {

	// find any existing one by name
	var foundISBSvc numaflowV1alpha1.InterStepBufferService
	namespacedName := k8stypes.NamespacedName{Namespace: mlictx.mlInferenceObj.Namespace, Name: isbsvcName(mlictx.mlInferenceObj.Name)}
	err := mlictx.client.Get(mlictx.ctx, namespacedName, &foundISBSvc)

	if err != nil {
		// doesn't exist: create a new one
		if apierrors.IsNotFound(err) {

			newSpec, err := mlictx.createISBSvc(mlictx.mlInferenceObj, nil)
			if err != nil {
				return err
			}

			mlictx.logger.Info("Creating ISBSvc", "name", newSpec.Name, "namespace", newSpec.Namespace)
			err = mlictx.client.Create(mlictx.ctx, newSpec)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// already exists; update it

		newSpec, err := mlictx.createISBSvc(mlictx.mlInferenceObj, &foundISBSvc)
		if err != nil {
			return err
		}

		mlictx.logger.Info("Updating ISBSvc", "name", newSpec.Name, "namespace", newSpec.Namespace)
		err = mlictx.client.Update(mlictx.ctx, newSpec)
		if err != nil {

			mlictx.logger.Info("Updating ISBSvc failed: will trying deleting/recreating", "err", err, "name", newSpec.Name, "namespace", newSpec.Namespace)
			err = mlictx.client.Delete(mlictx.ctx, &foundISBSvc) //todo: any special DeleteOptions? need to verify this will block so we can call Create after
			if err != nil {
				return err
			}
			newSpec, err = mlictx.createISBSvc(mlictx.mlInferenceObj, nil)
			if err != nil {
				return err
			}

			err = mlictx.client.Create(mlictx.ctx, newSpec)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (mlictx *MLInferenceOperatorContext) createFrontlineServiceSpec(mlInferenceObj *v1alpha1.MLInference, existingService *corev1.Service) (*corev1.Service, error) {
	var newService corev1.Service
	if existingService != nil {
		existingService.DeepCopyInto(&newService)
	}

	selectorLabels := map[string]string{
		dfv1.KeyMLInferenceName: mlInferenceObj.Name,
	}

	newService.Spec = corev1.ServiceSpec{
		Type:       "ClusterIP",
		IPFamilies: []corev1.IPFamily{"IPv4"},
		Ports: []corev1.ServicePort{
			{Name: "web", Port: dfv1.FrontlineServicePort, TargetPort: intstr.FromInt(int(dfv1.FrontlineServicePort))},
		},
		Selector: selectorLabels,
	}

	if newService.Labels == nil {
		newService.Labels = make(map[string]string)
	}
	newService.Labels[dfv1.KeyMLInferenceName] = mlInferenceObj.Name
	newService.Namespace = mlictx.mlInferenceObj.Namespace
	newService.Name = frontLineServiceName(mlictx.mlInferenceObj.Name)
	newService.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(mlInferenceObj.GetObjectMeta(), v1alpha1.GroupVersion.WithKind("MLInference")),
	}

	return &newService, nil
}

func (mlictx *MLInferenceOperatorContext) createFrontlineApplication(mlInferenceObj *v1alpha1.MLInference, existingStatefulSet *appv1.StatefulSet) (*appv1.StatefulSet, error) {
	var newStatefulSet appv1.StatefulSet
	if existingStatefulSet != nil {
		existingStatefulSet.DeepCopyInto(&newStatefulSet)
	}

	defaultReplicas := int32(3)
	terminationGracePeriodSec := int64(10)

	isSync := "false"
	if mlictx.mlInferenceObj.Spec.InferenceType == dfv1.InferenceTypeSync {
		isSync = "true"
	}

	podSpec := &corev1.PodSpec{
		TerminationGracePeriodSeconds: &terminationGracePeriodSec,
		Containers: []corev1.Container{
			{
				Name:  "main",
				Image: "quay.io/numaio/mlserve:latest", // todo: this can come from a ConfigMap rather than being hard coded as latest
				Env: []corev1.EnvVar{
					{Name: "SERVICE_NAME", Value: frontLineServiceName(mlictx.mlInferenceObj.Name)},
					{Name: "NAMESPACE", Value: mlictx.mlInferenceObj.Namespace},
					{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
				},
				Ports: []corev1.ContainerPort{
					{Name: "web", ContainerPort: dfv1.FrontlineServicePort},
				},
				Args: []string{ //todo: there are other parameters...
					"frontline",
					"--backendurl",
					pipelineSourceURL(mlictx.mlInferenceObj),
					"--sync",
					isSync,
					//"--insecure",
					//"true",
				},
			},
		},
	}

	labels := map[string]string{
		dfv1.KeyMLInferenceName: mlInferenceObj.Name,
		dfv1.KeyApp:             dfv1.LabelFrontlineApp,
	}

	newStatefulSet.Spec = appv1.StatefulSetSpec{
		PodManagementPolicy: appv1.ParallelPodManagement,
		Replicas:            &defaultReplicas,
		ServiceName:         frontLineServiceName(mlictx.mlInferenceObj.Name),
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: *podSpec,
		},
	}

	if newStatefulSet.Labels == nil {
		newStatefulSet.Labels = make(map[string]string)
	}
	newStatefulSet.Labels[dfv1.KeyMLInferenceName] = mlInferenceObj.Name
	newStatefulSet.Labels[dfv1.KeyApp] = dfv1.LabelFrontlineApp
	newStatefulSet.Namespace = mlictx.mlInferenceObj.Namespace
	newStatefulSet.Name = frontLineStatefulSetName(mlictx.mlInferenceObj.Name)
	newStatefulSet.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(mlInferenceObj.GetObjectMeta(), v1alpha1.GroupVersion.WithKind("MLInference")),
	}

	return &newStatefulSet, nil
}

func (mlictx *MLInferenceOperatorContext) createISBSvc(mlInferenceObj *v1alpha1.MLInference, existingISBSvc *numaflowV1alpha1.InterStepBufferService) (*numaflowV1alpha1.InterStepBufferService, error) {
	var newISBSvc numaflowV1alpha1.InterStepBufferService
	if existingISBSvc != nil {
		existingISBSvc.DeepCopyInto(&newISBSvc)
	}

	newISBSvc.Spec = numaflowV1alpha1.InterStepBufferServiceSpec{
		JetStream: &numaflowV1alpha1.JetStreamBufferService{
			Version: "latest", //todo: make configurable image tag, also add persistence
		},
	}

	if newISBSvc.Labels == nil {
		newISBSvc.Labels = make(map[string]string)
	}
	newISBSvc.Labels[dfv1.KeyMLInferenceName] = mlInferenceObj.Name
	newISBSvc.Namespace = mlictx.mlInferenceObj.Namespace
	newISBSvc.Name = isbsvcName(mlInferenceObj.Name)
	newISBSvc.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(mlInferenceObj.GetObjectMeta(), v1alpha1.GroupVersion.WithKind("MLInference")),
	}

	return &newISBSvc, nil
}

func (mlictx *MLInferenceOperatorContext) createPipelineServiceSpec(mlInferenceObj *v1alpha1.MLInference) (*corev1.Service, error) {
	var newService corev1.Service

	selectorLabels := map[string]string{
		dfv1.KeyMLInferenceName:            mlInferenceObj.Name,
		"numaflow.numaproj.io/vertex-name": "http-in",
	}

	//app.kubernetes.io/component: vertex
	//app.kubernetes.io/managed-by: vertex-controller
	//app.kubernetes.io/part-of: numaflow
	//numaflow.numaproj.io/pipeline-name: sample-ml-server-pipeline
	//numaflow.numaproj.io/vertex-name: http-in

	newService.Spec = corev1.ServiceSpec{
		Type: "ClusterIP",
		Ports: []corev1.ServicePort{
			{Name: "web", Port: dfv1.FrontlineServicePort, TargetPort: intstr.FromInt(int(dfv1.FrontlineServicePort))},
		},
		Selector: selectorLabels,
	}

	if newService.Labels == nil {
		newService.Labels = make(map[string]string)
	}
	newService.Labels[dfv1.KeyMLInferenceName] = mlInferenceObj.Name
	newService.Namespace = mlictx.mlInferenceObj.Namespace
	newService.Name = fmt.Sprintf("%s-pipeline-source-service", mlictx.mlInferenceObj.Name)
	newService.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(mlInferenceObj.GetObjectMeta(), v1alpha1.GroupVersion.WithKind("MLInference")),
	}

	return &newService, nil
}
func (mlictx *MLInferenceOperatorContext) createPipelineSpec(mlInference *v1alpha1.MLInference, existing *numaflowV1alpha1.Pipeline) (*numaflowV1alpha1.Pipeline, error) {

	var newPipeline numaflowV1alpha1.Pipeline
	if existing != nil {
		existing.DeepCopyInto(&newPipeline)
	}

	mliSpec := mlInference.Spec

	// create basic pl spec with http in and out
	var plSpec = numaflowV1alpha1.PipelineSpec{
		InterStepBufferServiceName: isbsvcName(mlInference.Name),
		Limits: &numaflowV1alpha1.PipelineLimits{
			ReadBatchSize: pointer.Uint64(1),
		},
		Vertices: []numaflowV1alpha1.AbstractVertex{
			{
				Name: "http-in",

				Scale: numaflowV1alpha1.Scale{
					Min: pointer.Int32(2),
				},
				AbstractPodTemplate: numaflowV1alpha1.AbstractPodTemplate{
					Metadata: &numaflowV1alpha1.Metadata{
						Labels: map[string]string{
							dfv1.KeyMLInferenceName: mlInference.Name,
						},
					},
				},
				Source: &numaflowV1alpha1.Source{
					HTTP: &numaflowV1alpha1.HTTPSource{
						Service: true,
					},
				},
			},
		},
	}
	if mliSpec.PreSteps != nil {
		for _, step := range mliSpec.PreSteps {
			stepVertex := numaflowV1alpha1.AbstractVertex{
				Name: step.Name,
				Scale: numaflowV1alpha1.Scale{
					Min: pointer.Int32(1),
				},
				UDF: &step.UDF,
			}
			plSpec.Vertices = append(plSpec.Vertices, stepVertex)
		}
	}

	// iterate over all predictors
	for _, predictor := range mliSpec.Predictors {
		// derive image from modelFormat
		// todo: put mapping in a central ConfigMap
		imagePath := ""
		udfArgs := []string{}
		switch predictor.Model.ModelFormat {
		case "pytorch":
			imagePath = "quay.io/numaio/mlserve/inference:v0.1.3"
			udfArgs = []string{"python", "serve.py"}
		default:
			return nil, fmt.Errorf("invalid ModelFormat (no known image): %q", predictor.Model.ModelFormat)
		}

		predictorVtx := numaflowV1alpha1.AbstractVertex{
			Name: predictor.Name,
			Scale: numaflowV1alpha1.Scale{
				Min: pointer.Int32(1),
			},
			/*UDF: &numaflowV1alpha1.UDF{
				Builtin: &numaflowV1alpha1.Function{Name: "cat"},
			},*/
			UDF: &numaflowV1alpha1.UDF{
				Container: &numaflowV1alpha1.Container{
					Image: imagePath,
					Args:  udfArgs,
					VolumeMounts: []corev1.VolumeMount{
						{
							MountPath: predictor.Model.ModelStore.S3.OutputPath,
							Name:      "model-storage",
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "DEBUG",
							Value: "true",
						},
						{
							Name:  "OUTPUT_PATH",
							Value: fmt.Sprintf("%s/%s", predictor.Model.ModelStore.S3.OutputPath, dfv1.ModelFileName),
						},
						{
							Name:  "MODEL_KEY",
							Value: predictor.Model.ModelStore.S3.ObjectKey,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name:         "model-storage", // empty dir
					VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:  "predictor-init",
					Image: "quay.io/numaio/mlserve:latest",
					VolumeMounts: []corev1.VolumeMount{
						{
							MountPath: predictor.Model.ModelStore.S3.OutputPath,
							Name:      "model-storage",
						},
					},
					Args: []string{"init"},
					Env: []corev1.EnvVar{
						{
							Name:  "HOST",
							Value: predictor.Model.ModelStore.S3.Host,
						},
						{
							Name: "ACCESSKEYID",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &predictor.Model.ModelStore.S3.AccessKeyID,
							},
						},
						{
							Name: "SECRETKEYACCESS",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &predictor.Model.ModelStore.S3.SecretAccessKey,
							},
						},
						{
							Name:  "BUCKETNAME",
							Value: predictor.Model.ModelStore.S3.Bucket,
						},
						{
							Name:  "OBJECTKEY",
							Value: predictor.Model.ModelStore.S3.ObjectKey,
						},
						{
							Name:  "OUTPUTPATH",
							Value: fmt.Sprintf("%s/%s", predictor.Model.ModelStore.S3.OutputPath, dfv1.ModelFileName),
						},
					},
				},
			},
		}
		plSpec.Vertices = append(plSpec.Vertices, predictorVtx)
	}

	if mliSpec.PostSteps != nil {
		for _, step := range mliSpec.PostSteps {
			stepVertex := numaflowV1alpha1.AbstractVertex{
				Name: step.Name,
				Scale: numaflowV1alpha1.Scale{
					Min: pointer.Int32(1),
				},
				UDF: &step.UDF,
			}
			plSpec.Vertices = append(plSpec.Vertices, stepVertex)
		}
	}
	plSpec.Vertices = append(plSpec.Vertices, numaflowV1alpha1.AbstractVertex{
		Name: "http-out",
		Scale: numaflowV1alpha1.Scale{
			Min: pointer.Int32(1),
		},
		Sink: &numaflowV1alpha1.Sink{
			UDSink: &numaflowV1alpha1.UDSink{
				Container: numaflowV1alpha1.Container{
					Image: "quay.io/numaio/mlserve:latest",
					Args:  []string{"sink"},
				},
			},
		},
	})
	// assumes vertices are in order in/presteps/predictors/poststeps/out
	// may not be correct
	prevVtx := "http-in"
	for _, vtx := range plSpec.Vertices {
		var edge numaflowV1alpha1.Edge
		if prevVtx == vtx.Name {
			continue
		} else {
			edge.From = prevVtx
			edge.To = vtx.Name
			prevVtx = vtx.Name
		}
		plSpec.Edges = append(plSpec.Edges, edge)
	}
	newPipeline.Spec = plSpec
	newPipeline.Name = mlInference.Name + "-pipeline"
	if newPipeline.Labels == nil {
		newPipeline.Labels = make(map[string]string)
	}
	newPipeline.Labels["mlinference"] = mlInference.Name
	newPipeline.Namespace = mlInference.Namespace
	newPipeline.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(mlInference.GetObjectMeta(), v1alpha1.GroupVersion.WithKind("MLInference")),
	}

	return &newPipeline, nil

}

// Aggregate Frontline Application Phase and Pipeline Phase into a single Phase
func (mlictx *MLInferenceOperatorContext) propagateStatus() {

	prioritizedPhases := []string{"Failed", "Error", "Degraded", "Pending", "Running"}

	for _, phase := range prioritizedPhases {
		if mlictx.mlInferenceObj.Status.ComponentStatus == nil {
			mlictx.mlInferenceObj.Status.ComponentStatus = &v1alpha1.ServingStatus{}
		}
		if mlictx.mlInferenceObj.Status.ComponentStatus.FrontlineService == nil {
			mlictx.mlInferenceObj.Status.ComponentStatus.FrontlineService = &v1alpha1.Status{}
		}
		if mlictx.mlInferenceObj.Status.ComponentStatus.Pipeline == nil {
			mlictx.mlInferenceObj.Status.ComponentStatus.Pipeline = &v1alpha1.Status{}
		}
		if mlictx.mlInferenceObj.Status.ComponentStatus.FrontlineService.Phase == phase || mlictx.mlInferenceObj.Status.ComponentStatus.Pipeline.Phase == phase {
			mlictx.mlInferenceObj.Status.Phase = phase
			return
		}
	}

}

// Get StatefulSet (Frontline Application) Status and propagate it into MLInference Status
// todo: can possibly expand to look for individual Pod statuses
func (mlictx *MLInferenceOperatorContext) propagateStatefulSetStatus(statefulSetStatus appv1.StatefulSetStatus) {
	if mlictx.mlInferenceObj.Status.ComponentStatus == nil {
		mlictx.mlInferenceObj.Status.ComponentStatus = &v1alpha1.ServingStatus{}
	}
	if mlictx.mlInferenceObj.Status.ComponentStatus.FrontlineService == nil {
		mlictx.mlInferenceObj.Status.ComponentStatus.FrontlineService = &v1alpha1.Status{}
	}
	mlictx.mlInferenceObj.Status.ComponentStatus.FrontlineService.Phase, mlictx.mlInferenceObj.Status.ComponentStatus.FrontlineService.Message = extractStatefulSetStatus(statefulSetStatus)

}

func extractStatefulSetStatus(statefulSetStatus appv1.StatefulSetStatus) (phase string, message string) {
	if statefulSetStatus.AvailableReplicas == statefulSetStatus.Replicas && statefulSetStatus.CurrentReplicas == statefulSetStatus.Replicas {
		phase = "Running"
		message = ""
		return
	} else {
		phase = "Pending"
		message = fmt.Sprintf("Current Replicas (%d) or Available Replicas (%d) less than total number of replicas (%d)",
			statefulSetStatus.CurrentReplicas, statefulSetStatus.AvailableReplicas, statefulSetStatus.Replicas)
	}

	return
}

// Get Pipeline Status and propagate it into MLInference Status
// todo: can possibly expand to look for individual Pod statuses
func (mlictx *MLInferenceOperatorContext) propagatePipelineStatus(pipelineStatus numaflowV1alpha1.PipelineStatus) {
	if mlictx.mlInferenceObj.Status.ComponentStatus == nil {
		mlictx.mlInferenceObj.Status.ComponentStatus = &v1alpha1.ServingStatus{}
	}
	if mlictx.mlInferenceObj.Status.ComponentStatus.Pipeline == nil {
		mlictx.mlInferenceObj.Status.ComponentStatus.Pipeline = &v1alpha1.Status{}
	}
	mlictx.mlInferenceObj.Status.ComponentStatus.Pipeline.Phase, mlictx.mlInferenceObj.Status.ComponentStatus.Pipeline.Message = extractPipelineStatus(pipelineStatus)
}

func extractPipelineStatus(pipelineStatus numaflowV1alpha1.PipelineStatus) (phase string, message string) {
	return string(pipelineStatus.Phase), pipelineStatus.Message
}

func frontLineServiceName(mlInferenceName string) string {
	return fmt.Sprintf("%s-frontline-service", mlInferenceName)
}

func frontLineServiceURL(mlInferenceName string, namespaceName string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", frontLineServiceName(mlInferenceName), namespaceName)
}

func frontLineStatefulSetName(mlInferenceName string) string {
	return fmt.Sprintf("%s-frontline-application", mlInferenceName)
}

func pipelineName(mlInferenceName string) string {
	return fmt.Sprintf("%s-pipeline", mlInferenceName)
}

func pipelineSourceURL(mlInference *v1alpha1.MLInference) string {
	return fmt.Sprintf("https://%s-pipeline-source-service.%s.svc.cluster.local:8443/vertices/http-in", mlInference.Name, mlInference.Namespace)
}

func isbsvcName(mlInferenceName string) string {
	return fmt.Sprintf("%s-isbsvc", pipelineName(mlInferenceName))
}
