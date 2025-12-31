package main

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
	"log"
	"time"
)

func handleMembers(records []*db.MemberRecord) error {
	err := memberTable.PutAll(records)
	if err != nil {
		return err
	}

	emailHandler.SendEmail(testEmail, "jotform webhook: Training Admin",
		fmt.Sprintf("Uploaded member table with %d members\n", len(records)))

	// Work out which member training confirmations email have not been sent and send them

	receivedSubmissions, err := trainTable.GetAllOfState(db.ReceivedSubmissionState)
	if err != nil {
		return err
	}

	log.Printf("Got %d received submission records successfully", len(receivedSubmissions))
	for _, sub := range receivedSubmissions {
		log.Printf("Received Submission ID %s", sub.GetID())
	}

	// Update submissions that are in the past
	receivedSubmissions, err = updateInPastSubmissions(receivedSubmissions)
	if err != nil {
		return fmt.Errorf("failed to update in-past submissions: %w", err)
	}

	// send emails for membership status updates
	for _, submission := range receivedSubmissions {
		if submission.SubmissionState != db.ReceivedSubmissionState ||
			submission.ReceivedRequestEmailSent == true {
			// could have been updated by earlier linked submission
			continue
		}

		// Re check membership number using records just submitted, avoiding db cost
		updatedMemberRecord := findMemberInRecords(submission.MembershipNumber, records)

		linkedMemberRecords, linkedSubmissions, err := findSubmissionSet(submission.LinkedSubmissionIds, receivedSubmissions)
		if err != nil {
			return err
		}

		if submission.FoundMemberRecord == false {

			if updatedMemberRecord == nil {
				// no update so the problem persists

				// Send email to members of linked submissions warning that this
				// membership number is invalid, if the linked submission are valid themselves
				emailHandler.SendProblemMessage(linkedMemberRecords, submission, fmt.Sprintf(`
The additional session cound not be processed because the membership number %s is not valid. This means
that no sessions have been booked for you. Please submit a new training request for all sessions with the
correct information.
`, submission.MembershipNumber))

				// Drop the submission set
				for _, sub := range linkedSubmissions {
					sub.SubmissionState = db.DroppedSubmissionState
				}
				err = trainTable.PutAll(linkedSubmissions)
				if err != nil {
					return err
				}
				continue
			}

			submission.FoundMemberRecord = true
			err = trainTable.Put(submission, submission.GetID())
			if err != nil {
				return err
			}

			// At this point Membership has just been created, time to send the received request email
			// That will be handled by the following code...
		}

		// membership number should be valid at this point

		// note: this will be the case for brand-new members too
		if submission.ActualCurrMem != submission.RequestCurrMem {

			// [re] check the membership status if there is an update
			if updatedMemberRecord != nil {
				oldActual := submission.ActualCurrMem
				newActual := membershipDateCheck(
					updatedMemberRecord.MembershipValidFrom,
					updatedMemberRecord.MembershipValidTo,
					&submission.TrainingDate)
				if oldActual != newActual {
					submission.ActualCurrMem = newActual
					// update the db
					err = trainTable.Put(submission, submission.GetID())
					if err != nil {
						return err
					}
				}
			}

			// local function to send received request email and update db state
			sendEmailsAndUpdate := func(extraText string) error {
				// but only if the linked set all have valid members
				if len(linkedSubmissions) == len(linkedMemberRecords) {
					emailHandler.SendReceivedRequest(linkedMemberRecords, linkedSubmissions, extraText)

					// update linked submissions
					for _, sub := range linkedSubmissions {
						sub.ReceivedRequestEmailSent = true
					}
					err = trainTable.PutAll(linkedSubmissions)
					if err != nil {
						return err
					}
				}
				return nil
			}

			// if the membership is now correct we can go ahead and send the received request email
			if submission.ActualCurrMem == submission.RequestCurrMem {
				err = sendEmailsAndUpdate("")
				if err != nil {
					return err
				}
				continue
			}

			// no update to the member record was received so the problem remains. Time to send
			// the received request message to the member with a warning
			err = sendEmailsAndUpdate(fmt.Sprintf(
				`However, we find that your membership runs out before the training session. Please renew your memebrship with Sport80.`))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// findSubmissionSet returns the same pointers passed in recvSubs, so that updated using pointers
// affect the original source
func findSubmissionSet(linkedIds []string, recvSubs []*db.TrainingSubmission) ([]*db.MemberRecord, []*db.TrainingSubmission, error) {
	var members []*db.MemberRecord
	var submissions []*db.TrainingSubmission
	var err error
	for _, subId := range linkedIds {
		// Look for the sudID in the recvSubs first
		var submission *db.TrainingSubmission
		for _, recvSub := range recvSubs {
			if recvSub.GetID() == subId {
				submission = recvSub
				break
			}
		}
		if submission == nil {
			// fall bact to the database
			submission, err = trainTable.Get(subId)
			if err != nil {
				return nil, nil, err
			}
		}
		member, err := memberTable.Get(submission.MembershipNumber)
		if err != nil {
			return nil, nil, err
		}
		submissions = append(submissions, submission)
		if member != nil {
			members = append(members, member)
		}
	}
	return members, submissions, nil
}

func findMemberInRecords(number string, records []*db.MemberRecord) *db.MemberRecord {
	for _, record := range records {
		if record.MemberNumber == number {
			return record
		}
	}
	return nil
}

func isInPast(sub *db.TrainingSubmission) bool {
	return sub.TrainingDate.Before(time.Now())
}

func updateInPastSubmissions(submissions []*db.TrainingSubmission) ([]*db.TrainingSubmission, error) {
	var result []*db.TrainingSubmission
	for _, submission := range submissions {
		if isInPast(submission) {
			submission.SubmissionState = db.InPastSubmissionState
			// update
			err := trainTable.Put(submission, submission.GetID())
			if err != nil {
				return nil, err
			}
		} else {
			result = append(result, submission)
		}
	}
	return result, nil
}
