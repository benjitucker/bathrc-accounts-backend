# bathrc-accounts-backend

https://docs.github.com/en/actions/publishing-packages/publishing-docker-images

## Configuration

The Lambda reads these environment variables. Leave the last two unset in production.

| Variable | Effect |
|---|---|
| `TEST_MODE` | Any value other than `false` sends all emails to the test addresses only. |
| `TABLE_NAME_PREFIX` | Prepended to every DynamoDB table name, e.g. `dev-` gives `dev-TrainingSubmissions`. Unset: production tables. |
| `FORM_TITLE_SUFFIX` | Incoming Jotform form titles must end with this, e.g. ` DEV` accepts "Training DEV" and "Training Administration DEV" and rejects the production forms. Unset: production form titles. |
