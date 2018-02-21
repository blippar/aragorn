# Aragorn

## Description

Aragorn is a regression testing tool that can run a set of tests periodically.
It acts as a server that can execute or schedule test suites.
(a JSON file with the `.suite.json` extension, see example below for more information)

When a test suite is run, each of its tests is executed. For each test, there
are 3 possible outcomes:

* the test's request is successful and its response is expected: nothing happens
* the test's request is successful but its response is unexpected: failures are reported
* the test's request can not be performed : the error is reported

> A request error can be an HTTP error while trying to make the request for
> example. Those errors will be retried 3 times, with a 30s pause in between
> each try. A failure is not retried.

Where the failures/errors are reported and how they are reported depends on the
notifier configured. For now the only one implemented is the SlackNotifier. It
is set at the initialization of a test suite. During a test suite execution, it
stacks every potential failure and/or error. At the end of the suite execution,
it will send a notification describing what happened if there was any failure or error.

## Quick Start

You can add Aragorn to your project by doing `aragorn init`

The directories to watch can be specified via command line positional arguments
: `aragorn watch dir1 dir2`.

## Config

The config is only used by the run command.
It contains a list of suites to execute or schedule and the notifiers.

| Name       | Type                     | Description                                          |
| ---------- | ------------------------ | ---------------------------------------------------- |
| notifiers  | `map[string]interface{}` | the notifiers configurations.                        |
| suites     | `[]SuiteConfig`          | List of suites to load.                              |
| consoleLog | `bool`                   | Log the suite reports on the console. (default true) |

Example:

```json
{
  "notifiers": {
    "slack": {
      "webhook": "https://hooks.slack.com/services/T/B/1",
      "verbose": true
    }
  },
  "suites": [
    {
      "path": "./test/service.suite.json",
      "runEvery": "1h",
      "failFast": true,
      "suite": {
        "base": {
          "url": "https://example.com",
          "insecure": true
        }
      }
    }
  ],
  "consoleLog": false
}
```

## Notifiers

### Slack

| Name     | Type     | Description                                       |
| -------- | -------- | ------------------------------------------------- |
| webhook  | `string` | URL to the Incoming Webhook for the HTTP request. |
| username | `string` | Username of the bot.                              |
| channel  | `string` | Channel where the notification will be send.      |
| verbose  | `bool`   | Log all tests.                                    |

## Test Suites

### SuiteConfig

A test suite describes a combination of tests to be run. It is composed of some
configuration fields for the scheduling and notification handling. The tests are described in the suite field depending on the type field (`HTTP` or `GRPC`).

| Name     | Type                       | Description                                                                                                                           |
| -------- | -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| path     | `string`                   | Path to the `SuiteConfig` only used in `Config.suites`                                                                                |
| name     | `string`                   | **REQUIRED**. The name of this suite.                                                                                                 |
| type     | `string`                   | **REQUIRED**. `HTTP` or `GRPC` (currently only `HTTP` is implemented)                                                                 |
| runEvery | `string`                   | A duration string parsable by time.ParseDuration specifying at each interval this test suite should be run. Exclusive with `runCron`. |
| runCron  | `string`                   | A cron-syntax string specifying when to run this test suite. Exclusive with `runEvery`                                                |
| failFast | `bool`                     | Stop after first test failure                                                                                                         |
| suite    | `HTTPSuite` or `GRPCSuite` | **REQUIRED**. An object describing the test suite itself. Depends on the field `type`.                                                |

Example:

```json
{
    "name": "Service 1 HTTP test suite",
    "type": "HTTP",
    "runEvery": "12h",
    "suite": {"..."}
}
```

### HTTPSuite

An HTTP test suite contains a base configuration and list of tests.

| Name  | Type         | Description                                               |
| ----- | ------------ | --------------------------------------------------------- |
| base  | `HTTPBase`   | **REQUIRED**. Base description of the tests in this suite |
| tests | `[]HTTPTest` | **REQUIRED**. List of tests to run.                       |

#### HTTPBase

| Name       | Type                | Description                                                                                                                    |
| ---------- | ------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| url        | `string`            | **REQUIRED**. Base URL prepended to all `path` in each test request.                                                           |
| header     | `map[string]string` | List of request header fields to add to every test in this suite. Each test can overwrite the header fields set at this level. |
| oauth2     | `OAUTH2Config`      | Describes a 2-legged OAuth2 flow.                                                                                              |
| retryCount | `int`               | Number of time the HTTP request can be retry. (default 1)                                                                      |
| retryWait  | `int`               | Duration between each retry in second. (default 1s)                                                                            |
| timeout    | `int`               | Timeout specifies a time limit for each request in second. (default 30s)                                                       |
| insecure   | `bool`              | Insecure controls whether a client verifies the server's certificate chain and host name.                                      |

#### OAUTH2Config

See golang.org/x/oauth2/clientcredentials Config [documentation](https://godoc.org/golang.org/x/oauth2/clientcredentials#Config) for more info.

#### HTTPTest

| Name    | Type          | Description                                                          |
| ------- | ------------- | -------------------------------------------------------------------- |
| name    | `string`      | **REQUIRED**. Name used to uniquely identify this test in the suite. |
| request | `HTTPRequest` | **REQUIRED**. Description of the HTTP request to perform.            |
| expect  | `HTTPExpect`  | **REQUIRED**. Expected result of the HTTP request.                   |

#### HTTPRequest

| Name      | Type                | Description                                                             |
| --------- | ------------------- | ----------------------------------------------------------------------- |
| path      | `string`            | Path appended to the base `url` set in the `HTTPBase`. Defaults to `/`. |
| method    | `string`            | HTTP method of the request. Defaults to `GET`.                          |
| header    | `map[string]string` | List of request header fields.                                          |
| multipart | `map[string]string` | Multipart content of the request. Values started with a `@` are files.  |
| formData  | `map[string]string` | Form data as application/x-url-encoded format.                          |
| body      | `Document`          | Request body.                                                           |
| timeout   | `int`               | Timeout specifies a time limit for the request in second. (default 30s) |

#### HTTPExpect

| Name       | Type                | Description                                  |
| ---------- | ------------------- | -------------------------------------------- |
| statusCode | `int`               | **REQUIRED**. Expected HTTP status code.     |
| header     | `map[string]string` | Expected key-value pairs in the HTTP header. |
| document   | `Document`          | Expected document to be returned.            |
| jsonSchema | `Object`            | Expected JSON schema (1) to be returned.     |
| jsonValues | `Object`            | Specific JSON values to be returned.         |

1. See [json-schema.org](http://json-schema.org/) and [Understanding JSON Schema](https://spacetelescope.github.io/understanding-json-schema/index.html) for more info.

#### Document

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

#### Object

Object must be a JSON object (`map[string]interface{}`). It can load a JSON
object from a file (see `Document` doc).

### GRPC Suite

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
          "jsonSchema": { "$ref": "schema.json" },
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
