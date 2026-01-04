package jotform_webhook

// TrainingRequest Allows either type to be used interchangeably.
type TrainingRequest interface {
	GetRawRequest() *TrainingRawRequest
}

func (r *TrainingRawRequest) GetRawRequest() *TrainingRawRequest {
	return r
}

func (r *TrainingRawRequestWithID) GetRawRequest() *TrainingRawRequest {
	return &r.TrainingRawRequest
}
