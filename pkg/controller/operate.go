package controller

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/numaproj-labs/numaserve/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MLInferenceOperatorContext struct {
	ctx            context.Context
	logger         logr.Logger
	client         client.Client
	mlInferenceObj *v1alpha1.MLInference
}

func NewMLInferenceOperateContext(ctx context.Context, logger logr.Logger, client client.Client, obj *v1alpha1.MLInference) *MLInferenceOperatorContext {
	return &MLInferenceOperatorContext{
		client:         client,
		ctx:            ctx,
		mlInferenceObj: obj,
		logger:         logger,
	}
}

func (mlictx *MLInferenceOperatorContext) Process() error {
	mlictx.logger.Info("Processing MLInference")

	// process MLInference Spec changes, as well as changes that may have been made to the Specs or Status of the resources owned by the MLInference
	if mlictx.Validate() {
		err := mlictx.Reconcile()
		if err != nil {
			return err //todo: may want to still do the stuff below but return an error at the end?
		}
	}

	// Update original object back (Status may have changed)
	// todo: consider that Status could be processed outside of the Reconciliation loop
	// (i.e. if just one managed component changes does it really make sense to do all of Reconciliation?)
	err := mlictx.client.Update(mlictx.ctx, mlictx.mlInferenceObj)
	if err != nil {
		return err
	}

	return nil
}
