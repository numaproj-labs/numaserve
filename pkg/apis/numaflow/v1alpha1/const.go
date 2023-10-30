package v1alpha1

const (
	InferenceTypeSync  = "sync"
	InferenceTypeAsync = "async"

	KeyMLInferenceName = "mlserve.numaproj.io/mlinference"
	KeyApp             = "mlinferences.mlserve.numaproj.io/app"

	LabelFrontlineApp = "mlserve-frontline"

	FrontlineServicePort = 8443
	ModelServingImage    = ""
	ModelFileName        = "model"
)
