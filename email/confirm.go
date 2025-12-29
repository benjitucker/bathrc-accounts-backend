package email

type ConfirmData struct {
	Name string
	Link string
}

func (eh *EmailHandler) SendConfirm(recipient string, data *ConfirmData) {
	eh.SendEmailPretty(recipient, "confirm", data)
}
