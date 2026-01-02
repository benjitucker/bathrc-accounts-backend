package main

import (
	"benjitucker/bathrc-accounts/db"
	"testing"
)

func Test_calcDistance(t *testing.T) {
	type args struct {
		submission        *db.TrainingSubmission
		transactionRecord *db.TransactionRecord
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "happy",
			args: args{
				submission: &db.TrainingSubmission{
					PaymentReference: "ABCD",
				},
				transactionRecord: &db.TransactionRecord{
					Description: "ABCD",
				},
			},
			want: 0,
		},
		{
			name: "maybe",
			args: args{
				submission: &db.TrainingSubmission{
					PaymentReference: "ABCD",
				},
				transactionRecord: &db.TransactionRecord{
					Description: "some text A ABCD more text",
				},
			},
			want: 0,
		},
		{
			name: "maybe",
			args: args{
				submission: &db.TrainingSubmission{
					PaymentReference: "ABCD",
				},
				transactionRecord: &db.TransactionRecord{
					Description: "some text ADCD more text",
				},
			},
			want: 2,
		},
		{
			name: "maybe3",
			args: args{
				submission: &db.TrainingSubmission{
					PaymentReference: "ABCD",
				},
				transactionRecord: &db.TransactionRecord{
					Description: "***tABCFt***",
				},
			},
			want: 2,
		},
		{
			name: "maybe6",
			args: args{
				submission: &db.TrainingSubmission{
					PaymentReference: "ABCD",
				},
				transactionRecord: &db.TransactionRecord{
					Description: "***tAB CDt***",
				},
			},
			want: 1,
		},
		{
			name: "maybe6",
			args: args{
				submission: &db.TrainingSubmission{
					PaymentReference: "ABCD",
				},
				transactionRecord: &db.TransactionRecord{
					Description: "***tAB CFt***",
				},
			},
			want: 2,
		},
		{
			name: "nomatch",
			args: args{
				submission: &db.TrainingSubmission{
					PaymentReference: "ABCD",
				},
				transactionRecord: &db.TransactionRecord{
					Description: "**BBCF++",
				},
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calcDistance(tt.args.submission, tt.args.transactionRecord); got != tt.want {
				t.Errorf("calcDistance() = %v, want %v", got, tt.want)
			}
		})
	}
}
