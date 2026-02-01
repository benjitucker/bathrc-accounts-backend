package email

import (
	"archive/zip"
	"benjitucker/bathrc-accounts/db"
	"bytes"
	"fmt"
	"io"
	"strings"
)

type IntroData struct {
	FirstName    string
	MemberNumber string
}

func (eh *EmailHandler) SendAppIntro(member *db.MemberRecord) {
	if eh.appIntroFile == nil {
		var err error
		eh.appIntroFile, err = eh.introPdfBytes()
		if err != nil {
			fmt.Printf("Failed to read app intro PDF: %v", err)
			return
		}
	}
	//email := member.Email TODO
	email := "ben@churchfarmmonktonfarleigh.co.uk"
	eh.SendEmailPrettyAttach([]string{email}, "intro", &IntroData{
		FirstName:    member.FirstName,
		MemberNumber: member.MemberNumber,
	}, "Training App Instructions.pdf", eh.appIntroFile)
}

func (eh *EmailHandler) introPdfBytes() ([]byte, error) {
	// Decompress Training App Instructions.zip
	zipBytes, err := assetsFS.ReadFile("assets/Training App Instructions.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to read app instructions zip: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to open app instructions zip: %w", err)
	}

	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".pdf") {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open pdf in zip: %w", err)
			}
			appIntroFile, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read pdf in zip: %w", err)
			}
			return appIntroFile, nil
		}
	}
	return nil, fmt.Errorf("no PDF found in Training App Instructions.zip")
}
