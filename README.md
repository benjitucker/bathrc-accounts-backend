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
| `STRIPE_SECRET_KEY_PARAM` | Stripe mode only: name of the SSM parameter holding the Stripe restricted key (needs Checkout Sessions: Write). |
| `STRIPE_WEBHOOK_SECRET_PARAM` | Stripe mode only: name of the SSM parameter holding the Stripe webhook endpoint's signing secret (`whsec_...`). |
| `PAY_LINK_SECRET_PARAM` | Stripe mode only: name of the SSM parameter holding the secret that signs payment links. |
| `PUBLIC_BASE_URL_PARAM` | Stripe mode only: name of the SSM parameter holding this Lambda's function URL, for the payment links in emails. If unset, the address is taken from the first request received, so emails sent by the hourly job straight after a cold start may have no link. |
| `PAYMENT_WINDOW` | Stripe mode only: how long a member has to pay after being sent a payment link, e.g. `2h` (the default) or `15m`. |
| `TRAINING_FORM_ID` | The Jotform Training form's ID. The hourly catch-up fetches its recent submissions, and `/redirect` only accepts submissions from it. Unset: the live Training form. An invalid value stops the Lambda starting rather than falling back to the live form. |

## Endpoints

The Lambda's function URL serves these paths. All but the last exist only when `PAYMENT_MODE=stripe`, and return 404 otherwise.

- `/redirect`: the Training form's Thank You page redirects the member's browser here with "Redirect with HTTP POST". Only the `submission_id` is read from the POST; the submission itself is fetched from the Jotform API and processed if the webhook hasn't already done so. A booking that is ready to pay goes straight to a Stripe payment page; otherwise the member is shown a page saying what happens next.
- `/pay?s=<submission>&t=<token>`: a payment link. It creates a fresh Stripe payment page each time, so it doesn't expire. The token is an HMAC of the submission id, so links can't be made up for other bookings. Stripe returns members who don't pay here with `&cancelled=1`, which shows a "Pay now" button.
- `/payment-complete`: where Stripe returns the member after paying.
- `/stripe-webhook`: Stripe's payment events. The signature is checked against the raw body. A paid `checkout.session.completed` (or `checkout.session.async_payment_succeeded`) marks the bookings Paid, records the payment id and sends the "payment received" email. Duplicate payments, payments for dropped bookings and wrong amounts are emailed to the test address for a refund or a look.
- anything else: the Jotform webhook, as before.

A submission can arrive through the webhook, the redirect and the hourly catch-up. Whichever arrives first claims it with a conditional DynamoDB write and processes it; the others skip it, so records aren't overwritten and emails aren't sent twice.

## Emails (Stripe mode)

In Stripe mode, the emails that would show bank transfer details (request received, payment reminder and duplicate request) show the amount and a **Pay now** payment link instead. The bank transfer versions are unchanged. After a membership refresh, held bookings are checked again and the member is emailed a payment link at the right price: the member price if their membership is valid on the session date, otherwise the non-member price, with a sentence explaining any change from the price they asked for.

In test mode, every booking email goes to the test addresses only.

## Unpaid bookings (Stripe mode)

A member has `PAYMENT_WINDOW` (2 hours by default) to pay, starting when they are first sent a payment link: at booking, or, for a booking held for a membership check, when the refresh sends the link. The emails give the exact deadline in UK time.

After the deadline the payment link shows "the time to pay has passed". The hourly job releases a booking once its deadline is more than 35 minutes past: the booking is marked Dropped, its Jotform submission is deleted so the place can be booked again, and the member is emailed. Stripe payment pages last 31 minutes, so a page opened before the deadline has always expired before the booking is released.

## Training prices (Stripe mode)

In Stripe mode, each booking's prices are read from the Training form's definition when the booking is made, so prices stay editable in Jotform. Each venue needs two fields whose **unique names** are the venue's value in the form followed by `mem` and `nonmem`, compared ignoring case. For example, a venue with the value `RiverMead` needs `rivermeadmem` and `rivermeadnonmem`. Adding a venue needs no code change.

The original venues keep their existing field names: `WestWilts` uses `wwecmem`/`wwecnonmem`, `Widbrook` uses `widmem`/`widnonmem`, and `StockleyFarm` uses `slfmem`/`slfnonmem`.

The member is charged the member price if their membership is valid on the session date, and the non-member price otherwise. If a venue's prices can't be read, the booking is held and a "PRICES NOT FOUND" email, including the price fields' definitions, is sent to the test address.
