package controller

import dfv1 "github.com/numaproj-labs/numaserve/pkg/apis/numaflow/v1alpha1"

// validate if MLInference spec meets requirements
func (mlictx *MLInferenceOperatorContext) Validate() bool {

	mlictx.logger.Info("validating MLInference")
	mliSpec := mlictx.mlInferenceObj.Spec

	// must have at least 1 predictor item
	if len(mliSpec.Predictors) == 0 {
		return false
	}

	// inference type must be async or sync
	if mliSpec.InferenceType != dfv1.InferenceTypeAsync && mliSpec.InferenceType != dfv1.InferenceTypeSync {
		return false
	}

	// validate predictors
	predictors := mliSpec.Predictors
	for _, predictor := range predictors {
		// model format should be set (pytorch for now)
		if predictor.Model.ModelFormat != "pytorch" {
			return false
		}
		// check model store - TODO
		// if predictor.Model.ModelStore.S3 != nil {

		// }
	}
	return true
}
