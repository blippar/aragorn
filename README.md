# Aragorn

## Description

Aragorn is a regression testing tool that can run a set of tests (test suites)
periodically. It acts as a server that watches for changes a list of
directories. When a file describing a test suite (a JSON file with the
`.suite.json` extension, see example below for more information) is added to one
of the watched directories, it is analysed, prepared and scheduled to be run
based on its own configuration. If a file is deleted, it will remove this test
suite from the scheduler and it will not be run anymore. If a file is modified,
the old test suite is removed from the scheduler and a new one is created, it is
equivalent to remove the file and create a new one.

When a test suite is run, each of its tests is executed. For each test, there
are 3 possible outcomes:

* the test's request is successful and its response is expected: nothing happens
* the test's request is successful but its response is unexpected: failures are
  reported
* the test's request can not be performed : the error is reported

> A request error can be an HTTP error while trying to make the request for
> example. Those errors will be retried 3 times, with a 30s pause in between
> each try. A failure is not retried.

Where the failures/errors are reported and how they are reported depends on the
notifier configured. For now the only one implemented is the SlackNotifier. It
is set at the initialization of a test suite. During a test suite execution, it
stacks every potential failure and/or error. At the end of the suite execution,
it will send a notification describing what happened if there was any
failure or error. If everything ran without any issue, no notification is sent.

## Quick Start

You can add Aragorn to your project by doing `aragorn init`

The directories to watch can be specified via command line positional arguments
: `./aragorn run dir1 dir2`.

## Test Suites

### Creating a test suite

#### SuiteConfig

A test suite describes a combination of tests to be run. It is composed of some
configuration fields for the scheduling and notification handling. The tests are described in the suite field depending on the type field (`HTTP` or `GRPC`).

| Name           | Type                       | Required | Description                                                                                                                           |
| -------------- | -------------------------- | :------: | ------------------------------------------------------------------------------------------------------------------------------------- |
| name           | `string`                   |    ✔️    | The name of this suite.                                                                                                               |
| type           | `string`                   |    ✔️    | `HTTP` or `GRPC` (currently only `HTTP` is implemented)                                                                               |
| runEvery       | `string`                   |          | A duration string parsable by time.ParseDuration specifying at each interval this test suite should be run. Exclusive with `runCron`. |
| runCron        | `string`                   |          | A cron-syntax string specifying when to run this test suite. Exclusive with `runEvery`                                                |
| slack.webhook  | `string`                   |    ✔️    | A Slack webhook used to post notifications in case this test suite fails.                                                             |
| slack.username | `string`                   |    ✔️    | A Slack username used to post notifications in case this test suite fails.                                                            |
| slack.channel  | `string`                   |    ✔️    | A Slack channel used to post notifications in case this test suite fails.                                                             |
| suite          | `HTTPSuite` or `GRPCSuite` |    ✔️    | An object describing the test suite itself. Depends on the field `type`.                                                              |

Example :

```json
{
    "name": "Service 1 HTTP test suite",
    "type": "HTTP",
    "runEvery": "12h",
    "slack": {
      "webhook": "https://hooks.slack.com/services/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",
      "username": "test-bot",
      "channel": "testing"
    },
    "suite": {"..."}
}
```

#### HTTPSuite

An HTTP test suite contains a base configuration and list of tests.

| Name  | Type         | Required | Description                                 |
| ----- | ------------ | :------: | ------------------------------------------- |
| base  | `HTTPBase`   |    ✔️    | Base description of the tests in this suite |
| tests | `[]HTTPTest` |    ✔️    | List of tests to run.                       |

##### HTTPBase

| Name       | Type                | Required | Description                                                                                                                    |
| ---------- | ------------------- | :------: | ------------------------------------------------------------------------------------------------------------------------------ |
| url        | `string`            |    ✔️    | Base URL prepended to all `path` in each test request.                                                                         |
| header     | `map[string]string` |          | List of request header fields to add to every test in this suite. Each test can overwrite the header fields set at this level. |
| oauth2     | `OAUTH2Config`      |          | Describes a 2-legged OAuth2 flow.                                                                                              |
| retryCount | `int`               |          | Number of time the HTTP request can be retry. (default 1)                                                                      |
| retryWait  | `int`               |          | Duration between each retry in second. (default 1s)                                                                            |

##### OAUTH2Config

See golang.org/x/oauth2/clientcredentials Config [documentation](https://godoc.org/golang.org/x/oauth2/clientcredentials#Config) for more info.

##### HTTPTest

| Name    | Type          | Required | Description                                            |
| ------- | ------------- | :------: | ------------------------------------------------------ |
| name    | `string`      |    ✔️    | Name used to uniquely identify this test in the suite. |
| request | `HTTPRequest` |    ✔️    | Description of the HTTP request to perform.            |
| expect  | `HTTPExpect`  |    ✔️    | Expected result of the HTTP request.                   |

##### HTTPRequest

| Name      | Type                | Required | Description                                                             |
| --------- | ------------------- | :------: | ----------------------------------------------------------------------- |
| path      | `string`            |          | Path appended to the base `url` set in the `HTTPBase`. Defaults to `/`. |
| method    | `string`            |          | HTTP method of the request. Defaults to `GET`.                          |
| header    | `map[string]string` |          | List of request header fields.                                          |
| multipart | `map[string]string` |          | Multipart content of the request. Values started with a `@` are files.  |
| formData  | `map[string]string` |          | Form data as application/x-url-encoded format.                          |
| body      | `Document`          |          | Request body.                                                           |

##### HTTPExpect

| Name       | Type                | Required | Description                                  |
| ---------- | ------------------- | :------: | -------------------------------------------- |
| statusCode | `int`               |    ✔️    | Expected HTTP status code.                   |
| header     | `map[string]string` |          | Expected key-value pairs in the HTTP header. |
| document   | `Document`          |          | Expected document to be returned.            |
| jsonSchema | `Object`            |          | Expected JSON schema (1) to be returned.     |
| jsonValues | `Object`            |          | Specific JSON values to be returned.         |

1. See [json-schema.org](http://json-schema.org/) for more info.

##### Document

Document is any type with some special behaviors like reference.

Load a JSON document from a file:

```json
{
  "$ref": "user.json"
}
```

Load a RAW document from a file:

```json
{
  "$ref": "image.jpg",
  "$raw": true
}
```

Inline RAW document:

```json
{
  "$raw": "OK"
}
```

##### Object

Object must be a JSON object (`map[string]interface{}`). It can load a JSON
object from a file (see `Document` doc).

#### GRPC Suite

Work in progress...

## Example

This example shows how to create an HTTP test suite file that will run 2 tests
every 12h:

* The first one is a `GET http://localhost:8080/` and expects a 200 JSON
  response `{"key": "value"}`.
* The second test is a `POST http://localhost:8080/echo` with a JSON body
  containing `{"key":"value", "a": [1, 2, 3], "b": {"c": "d"}}`. It expects a JSON response matching the JSON schema `schema.json` and the given JSON values.

```json
{
  "name": "Service 1 HTTP test suite",
  "type": "HTTP",
  "runEvery": "12h",
  "slack": {
    "webhook":
      "https://hooks.slack.com/services/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",
    "username": "test-bot",
    "channel": "testing"
  },
  "suite": {
    "base": {
      "url": "http://localhost:8080",
      "header": {
        "Accept-Encoding": "application/json"
      },
      "oauth2": {
        "clientID": "id",
        "clientSecret": "secret",
        "tokenURL": "https://localhost:8080/token"
      }
    },
    "tests": [
      {
        "name": "root",
        "request": {
          "path": "/",
          "method": "GET"
        },
        "expect": {
          "statusCode": 200,
          "header": {
            "Content-Type": "application/json",
            "Content-Length": "15"
          },
          "jsonDocument": { "key": "value" }
        }
      },
      {
        "name": "echo",
        "request": {
          "path": "/echo",
          "method": "POST",
          "header": {
            "Content-Type": "application/json"
          },
          "body": { "key": "value" }
        },
        "expect": {
          "statusCode": 200,
          "header": {
            "Content-Type": "application/json"
          },
          "jsonSchema": "@schema.json",
          "jsonValues": {
            "key": "value",
            "a.0": 1,
            "b.c": "d"
          }
        }
      }
    ]
  }
}
```
