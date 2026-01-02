package jotform_webhook

type TempUpload struct {
	UploadStatement []string `json:"q4_uploadStatement"`
}

type TrainingAdminRawRequest struct {
	Slug          string     `json:"slug"`
	SubmitSource  string     `json:"submitSource"`
	SubmitDate    UnixMillis `json:"submitDate"`
	BuildDate     UnixMillis `json:"buildDate"`
	SendEmailsNow string     `json:"q7_typeA"`
	EventID       string     `json:"event_id"`
	TimeToSubmit  string     `json:"timeToSubmit"`
	TempUpload    TempUpload `json:"temp_upload"`
	FileServer    string     `json:"file_server"`
	UploadURLs    []string   `json:"uploadStatement"`
	Path          string     `json:"path"`

	// For test:
	ExtraCSV *string `json:"extraCsv"`
}

func (TrainingAdminRawRequest) FormKind() string {
	return "Training Administration"
}
