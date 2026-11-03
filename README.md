# bathrc-accounts-backend

https://docs.github.com/en/actions/publishing-packages/publishing-docker-images

## Configuration

The Lambda reads these environment variables. Leave all but `TEST_MODE` unset in production.

| Variable | Effect |
|---|---|
| `TEST_MODE` | Any value other than `false` sends all emails to the test addresses only. |
| `TABLE_NAME_PREFIX` | Prepended to every DynamoDB table name, e.g. `dev-` gives `dev-TrainingSubmissions`. Unset: production tables. |
| `FORM_TITLE_SUFFIX` | Incoming Jotform form titles must end with this, e.g. ` DEV` accepts "Training DEV" and "Training Administration DEV" and rejects the production forms. Unset: production form titles. |
| `PAYMENT_MODE` | `bacs` (default) or `stripe`. `stripe` enables the `/redirect` endpoint below. Any other value stops the Lambda starting. |
| `TRAINING_FORM_ID` | The Jotform Training form's ID. The hourly catch-up fetches its recent submissions, and `/redirect` only accepts submissions from it. Unset: the live Training form. An invalid value stops the Lambda starting rather than falling back to the live form. |

## Endpoints

The Lambda's function URL serves two paths:

- `/redirect`: only when `PAYMENT_MODE=stripe`, otherwise 404. The Training form's Thank You page redirects the member's browser here with "Redirect with HTTP POST". Only the `submission_id` is read from the POST; the submission itself is fetched from the Jotform API, processed if the webhook hasn't already done so, and the member is shown a page saying what happens next.
- anything else: the Jotform webhook, as before.

A submission can arrive through the webhook, the redirect and the hourly catch-up. Whichever arrives first claims it with a conditional DynamoDB write and processes it; the others skip it, so records aren't overwritten and emails aren't sent twice.
