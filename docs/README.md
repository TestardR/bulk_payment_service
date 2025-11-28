# Interview exercise - Backend Banking

We have a wide variety of customers. Some of them are big organizations that need to perform a large number of transfers. For example, paying the salaries of hundreds of employees at the end of the month.

Performing those transfers one by one would be painful and time consuming for our customers, and for that reason we allow them to perform transfers _in bulk_.

Your mission, should you choose to accept it, is to write a web service to receive bulk transfer requests from a single account and:
1. Verify the validity of the request: whether the customer has enough funds for all the transfers in the request. If the customer does not have enough funds, the entire request should be denied;
2. If the request must be denied, return a 422 HTTP response;
3. Otherwise, add the transfers to the database, update the customer's balance, and return a 201 HTTP response.

## Instructions

You may use the language of your choice.

You can access the Internet for information.

You should favour commonly-used libraries whenever possible.

Prepare a document (ex: README.md) describing your solution, the approach you used to solve the problem, issues faced, potential improvements, instructions about how to run your solution, etc. (tip: think like you are submitting this solution as a pull request to be reviewed by your teammates).

When you are finished, gather your files in a zip or tgz archive and submit by email.

## Time

The recommended amount of time to spend on the test is 3 hours.

Note that you're not expected to deliver a "perfect" solution. We are interested in your thought process, and the general problem solving approach, and we will consider all submissions even if incomplete/unfinished.

That said, if you'd like to take more time to improve, or "finish" your solution, feel free to do so (just let us know beforehand), and we will consider the final submission as well.

## Resources at your disposal

You can find the following files to help you on your way:
- `sample1.json`, `sample2.json`: the bulk transfer requests you receive,
- `payment_accounts.sqlite`: a sample SQLite database you can use.

The bulk transfer request is a JSON document, with the contract:

```yaml
openapi: 3.0.0
info:
  title: Bulk Transfer API
  version: 1.0.0
  description: API for processing bulk transfer requests
paths:
  /transfers/bulk:
    post:
      summary: Create a bulk transfer request
      description: Submit a bulk transfer request with multiple credit transfer entries.
      operationId: createBulkTransferRequest
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [organization_bic, organization_iban, credit_transfers]
              properties:
                organization_bic:
                  type: string
                  description: The BIC of customer's organization.
                  example: QONTRR2X
                organization_iban:
                  type: string
                  description: The IBAN of customer's organization.
                  example: FR7612345678901234567890123
                credit_transfers:
                  type: array
                  description: List of individual credit transfer entries.
                  items:
                    type: object
                    required: [amount, currency, counterparty_bic, counterparty_iban, counterparty_name, description]
                    properties:
                      amount:
                        type: string
                        description: Amount of the transfer.
                        example: "13.22"
                      currency:
                        type: string
                        example: EUR
                        description: Currency of the transfer (always EUR).
                      counterparty_bic:
                        type: string
                        description: The BIC of the counterparty's bank.
                        example: DEUTDEFF
                      counterparty_iban:
                        type: string
                        description: The IBAN of the counterparty's account.
                        example: DE89370400440532013000
                      counterparty_name:
                        type: string
                        description: Name of the counterparty.
                        example: Lufthansa
                      description:
                        type: string
                        description: Description of the transfer.
                        example: Ski equipment
      responses:
        '204':
          description: Bulk transfer request created successfully.
```

You can use https://editor.swagger.io to visualize the OpenAPI contract.

**Note: the OpenAPI only specifies successful responses. It is up to you to handle non-successful cases.**

## The bank's database

You can use the provided `payment_accounts.sqlite`. It contains the customer's account and transactions linked to them.

> You can also use your own database, but in this case you must provide a Dockerfile allowing us to properly run your solution, along with the means to create the necessary databases, tables, seed data, etc.

The schema of the provided `payment_accounts.sqlite` is composed of two tables described below.

### `bank_accounts`

Structure:

- `id` *INTEGER*: Unique internal identifier of customer bank accounts.
- `organization_name` *TEXT* Name of the organization that opened the account.
- `balance_cents` *INTEGER* Current balance of the account.
- `iban` *TEXT* IBAN of the account.
- `bic` *TEXT* BIC of the account.

### `transactions`

- `id` *INTEGER* Transaction unique identifier.
- `counterparty_name` *TEXT* Name of the counterparty of the transaction.
- `counterparty_iban` *TEXT* IBAN of the counterparty.
- `counterparty_bic` *TEXT* BIC of the counterparty.
- `amount_cents` *INTEGER* Amount, in cents, of the transaction (negative for
  debit, positive for credit).
- `amount_currency` *TEXT* Currency of the transaction (generally, EUR).
- `bank_account_id` *INTEGER* `id` of the related `bank_account`.
- `description` *TEXT* Description of the transaction.
