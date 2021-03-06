<p align="center">
    <img alt="Aragorn logo, by Malenea" src="logo.png"></img>
</p>
<h4 align="center">Regression testing made easy</h4>
<p align="center">
    <a href="https://goreportcard.com/report/github.com/blippar/aragorn">
        <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/blippar/aragorn">
    </a>
    <a href="https://godoc.org/github.com/blippar/aragorn">
        <img alt="GoDoc" src="https://godoc.org/github.com/blippar/aragorn?status.svg">
    </a>
    <a href="https://github.com/blippar/aragorn/blob/master/LICENSE">
       <img alt="License" src="https://img.shields.io/github/license/blippar/aragorn.svg">
    </a>
    <a href="https://github.com/blippar/aragorn/releases/latest">
        <img alt="Latest Release" src="https://img.shields.io/github/release/blippar/aragorn.svg">
    </a>
    <a href="https://hub.docker.com/r/blippar/aragorn/">
        <img alt="Docker Image" src="https://img.shields.io/docker/automated/blippar/aragorn.svg">
    </a>
</p>

---

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

| Name      | Type                     | Description                   |
| --------- | ------------------------ | ----------------------------- |
| notifiers | `map[string]interface{}` | the notifiers configurations. |
| suites    | `[]SuiteConfig`          | List of suites to load.       |

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
  ]
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

| Name       | Type                       | Description                                                                                                                           |
| ---------- | -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| path       | `string`                   | Path to the `SuiteConfig` only used in `Config.suites`                                                                                |
| name       | `string`                   | **REQUIRED**. The name of this suite.                                                                                                 |
| type       | `string`                   | **REQUIRED**. `HTTP` or `GRPC` (currently only `HTTP` is implemented)                                                                 |
| runEvery   | `string`                   | A duration string parsable by time.ParseDuration specifying at each interval this test suite should be run. Exclusive with `runCron`. |
| runCron    | `string`                   | A cron-syntax string specifying when to run this test suite. Exclusive with `runEvery`                                                |
| retryCount | `int`                      | Number of time a test can be retried, if any error happened. (default 1)                                                              |
| retryWait  | `string`                   | Duration between each retry. (default 1s)                                                                                             |
| timeout    | `string`                   | Timeout specifies a time limit for each test. (default 30s)                                                                           |
| failFast   | `bool`                     | Stop after first test failure                                                                                                         |
| suite      | `HTTPSuite` or `GRPCSuite` | **REQUIRED**. An object describing the test suite itself. Depends on the field `type`.                                                |

Example:

```json
{
  "name": "Service 1 HTTP test suite",
  "type": "HTTP",
  "runEvery": "12h",
  "retryCount": 3,
  "retryWait": "100ms",
  "timeout": "10s",
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

| Name     | Type                | Description                                                                                                                    |
| -------- | ------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| url      | `string`            | **REQUIRED**. Base URL prepended to all `path` in each test request.                                                           |
| header   | `map[string]string` | List of request header fields to add to every test in this suite. Each test can overwrite the header fields set at this level. |
| oauth2   | `OAUTH2Config`      | Describes a 2-legged OAuth2 flow.                                                                                              |
| insecure | `bool`              | Insecure controls whether a client verifies the server's certificate chain and host name.                                      |

#### OAUTH2Config

See golang.org/x/oauth2/clientcredentials Config [documentation](https://godoc.org/golang.org/x/oauth2/clientcredentials#Config) for more info.

#### HTTPTest

| Name         | Type          | Description                                                          |
| ------------ | ------------- | -------------------------------------------------------------------- |
| id           | `string`      | Identifier use for stateful url templating tests.                    |
| name         | `string`      | **REQUIRED**. Name used to uniquely identify this test in the suite. |
| request      | `HTTPRequest` | Description of the HTTP request to perform.                          |
| expect       | `HTTPExpect`  | Expected result of the HTTP request.                                 |
| saveDocument | `bool`        | Save the response document for other tests.                          |

#### HTTPRequest

| Name      | Type                | Description                                                            |
| --------- | ------------------- | ---------------------------------------------------------------------- |
| path      | `string`            | Path appended to the base `url` set in the `HTTPBase`. (default: `/`)  |
| method    | `string`            | HTTP method of the request. (default: `GET`)                           |
| header    | `map[string]string` | List of request header fields.                                         |
| multipart | `map[string]string` | Multipart content of the request. Values started with a `@` are files. |
| formData  | `map[string]string` | Form data as application/x-url-encoded format.                         |
| body      | `HTTPDocument`      | Request body.                                                          |

#### HTTPExpect

| Name       | Type                | Description                                                              |
| ---------- | ------------------- | ------------------------------------------------------------------------ |
| statusCode | `int`               | Expected HTTP status code. Not checked if the value is -1 (default: 200) |
| header     | `map[string]string` | Expected key-value pairs in the HTTP header.                             |
| document   | `HTTPDocument`      | Expected document to be returned.                                        |
| jsonSchema | `HTTPObject`        | Expected JSON schema (1) to be returned.                                 |
| jsonValues | `HTTPObject`        | Expected Specific JSON values to be returned.                            |

1.  See [json-schema.org](http://json-schema.org/) and [Understanding JSON Schema](https://spacetelescope.github.io/understanding-json-schema/index.html) for more info.

#### HTTP URL Templating

The URL path and query can be constructed from previous tests through templating.

```json
{
  "tests": [
    {
      "id": "add_todo",
      "name": "Add Todo",
      "request": {
        "method": "POST",
        "path": "/todo"
      },
      "saveDocument": true
    },
    {
      "name": "Get Todo",
      "request": {
        "method": "GET",
        "path": "/todo/{{add_todo.id}}"
      }
    }
  ]
}
```

#### HTTPDocument

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

#### HTTPObject

Object must be a JSON object (`map[string]interface{}`). It can load a JSON
object from a file (see `Document` doc).

#### HTTP Example

This example shows how to create an HTTP test suite file that has 2 tests:

* The first one is a `GET http://localhost:8080/` request and expects a 200 OK JSON response `{"key": "value"}`.
* The second test is a `POST http://localhost:8080/echo` request with a 201 Created response and JSON body containing `{"key":"value", "a": [1, 2, 3], "b": {"c": "d"}}`. It expects a JSON response matching the JSON schema `schema.json` and the given JSON values.

```json
{
  "name": "Service 1",
  "type": "HTTP",
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
          "statusCode": 201,
          "header": {
            "Content-Type": "application/json"
          },
          "jsonSchema": { "$ref": "schema.json" },
          "jsonValues": {
            "key": "value",
            "a.0": 1,
            "a.length": 3,
            "b.c": "d"
          }
        }
      }
    ]
  }
}
```

### GRPC Suite

An GRPC test suite contains a base configuration and list of tests.

| Name               | Type           | Description                                                                                                                    |
| ------------------ | -------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| address            | `string`       | **REQUIRED**. target address for the connection to the server.                                                                 |
| protoSetPath       | `string`       | The path of a file containing an encoded FileDescriptorSet. (default: get the remote server proto via the GRPC reflection API) |
| tls                | `bool`         | TLS connection to the server. If false, use plain-text HTTP/2 (default: false)                                                 |
| caPath             | `string`       | File containing trusted root certificates for verifying the server.                                                            |
| serverHostOverride | `string`       | Override the virtual host name of authority (e.g. :authority header field) in requests.                                        |
| insecure           | `bool`         | Skip server certificate and domain verification.                                                                               |
| oauth2             | `OAUTH2Config` | Describes a 2-legged OAuth2 flow.                                                                                              |
| header             | `Header`       | List of request header fields to add to every test in this suite. Each test can overwrite the header fields set at this level. |
| tests              | `[]GRPCTest`   | **REQUIRED**. List of tests to run.                                                                                            |

#### GRPCTest

| Name    | Type          | Description                                                          |
| ------- | ------------- | -------------------------------------------------------------------- |
| name    | `string`      | **REQUIRED**. Name used to uniquely identify this test in the suite. |
| request | `GRPCRequest` | **REQUIRED**. Description of the GRPC request to perform.            |
| expect  | `GRPCExpect`  | Expected result of the GRPC request.                                 |

#### GRPCRequest

| Name     | Type                | Description                                                                                  |
| -------- | ------------------- | -------------------------------------------------------------------------------------------- |
| method   | `string`            | **REQUIRED**. GRPC method of the request. (e.g. `grpcexpect.testing.TestService/SimpleCall`) |
| header   | `map[string]string` | List of request header fields.                                                               |
| document | `GRPCDocument`      | Expected request message of the GRPC method.                                                 |

#### GRPCExpect

| Name     | Type                | Description                                   |
| -------- | ------------------- | --------------------------------------------- |
| Code     | `string`            | Expected GRPC code. (default: `OK`)           |
| header   | `map[string]string` | Expected key-value pairs in the GRPC header.  |
| document | `GRPCDocument`      | Expected response message of the GRPC method. |

#### GRPCDocument

Document is any type with some special behaviors like reference.

Load a JSON document from a file:

```json
{
  "$ref": "user.json"
}
```

#### GRPC Example

This example shows how to create an GRPC test suite file that has 2 tests.

```json
{
  "name": "Service 2",
  "type": "GRPC",
  "suite": {
    "address": "localhost:50051",
    "header": { "hello": "world" },
    "tests": [
      {
        "name": "Empty Call",
        "request": { "method": "grpcexpect.testing.TestService/EmptyCall" },
        "expect": { "code": "OK" }
      },
      {
        "name": "Simple Call",
        "request": {
          "method": "grpcexpect.testing.TestService/SimpleCall",
          "document": { "username": "world" }
        },
        "expect": {
          "code": "OK",
          "header": { "hello": "world" },
          "document": { "message": "Hello world!" }
        }
      }
    ]
  }
}
```

## Tracing

This project use [OpenTracing](http://opentracing.io/), a vendor-neutral open standard for distributed tracing. When aragorn run a test suite it will create a span that will be propagated in the context, a sub span is created for each test. More details are filled by the test suite package that implement the execution of the test. For example, the http call in the `httpexpect` package is traced in a sub span.

By default, the aragorn command will run with no tracer. You can set a tracer with the tracer flag. More info with `aragorn help exec`.

## Credits

Thanks to [Malenea](https://github.com/malenea) for his awesome work on the logo.

Aragorn's logo is licensed under the [Creative Commons Attribution 4.0 International License](http://creativecommons.org/licenses/by/4.0/) and was widely inspired on the original Go gopher designed by [Renee French](http://reneefrench.blogspot.com/).