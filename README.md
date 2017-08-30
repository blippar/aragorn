# Description

Aragorn is a regression testing tool that can run a set of tests (tests suites) periodically. It acts as a server that watches for changes a list of directories. When a file describing a tests suite (a JSON file with the .suite.json extension, see example below for more information) is added to one of the watched directories, it is analysed, prepared and scheduled to be run based on its own configuration. If a file is deleted, it will remove this tests suite from the scheduler and it will not be run anymore. If a file is modified, the old tests suite is removed from the scheduler and a new one is created, it is equivalent to remove the file and create a new one.

When a tests suite is run, each of its tests is executed. For each test, there are 3 possible outcomes :
- the test's request is successful and its response is expected : nothing happens
- the test's request is successful but its response is unexpected : failures are reported
- the test's request can not be performed : the error is reported

> A request error can be an HTTP error while trying to make the request for example. Those errors will be retried 3 times, with a 30s pause in between each try. A failure is not retried.

Where the failures/errors are reported and how they are reported depends on the notifier configured. For now the only one implemented is the SlackNotifier. It is set at the initialization of a tests suite. During a tests suite execution, it stacks every potential failure and/or error. At then end of the exectution of the suite, it will send a notification describing what happened if there was any failure or error. If everything ran without any issue, no notification is sent.

# Quick Start

The directories to watch can be specified via command line positional arguments : `./aragorn dir1 dir2`.

You can humanize the log output with `-humanize`.

# Tests suites

## Creating a tests suite

A tests suite describes a combination of tests to be run. It is composed of some configuration fields and of the tests suite itself, that depends on the type set (`HTTP` or `GRPC`).

|Name|Type|Required|Description|
|-|-|-|-|
|name|`string`|✔️|The name of this suite.|
|type|`string`|✔️|`HTTP` or `GRPC` (currently only `HTTP` is implemented)|
|runEvery|`string`||A duration string parsable by time.ParseDuration specifying at each interval this tests suite should be run. Exclusive with `runCron`.|
|runCron|`string`||A cron-syntax string specifying when to run this tests suite. Exclusive with `runEvery`|
|slackWebhook|`string`|✔️|A Slack webhook used to post notifications in case this tests suite fails.|
|suite|`HTTPsuite` or `GRPCSuite`|✔️|An object describing the tests suite itself. Depends on the field `type`.|

Example :

```json
{
    "name": "Service 1 HTTP tests suite",
    "type": "HTTP",
    "runEvery": "12h",
    "slackWebhook": "https://hooks.slack.com/services/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",
    "suite": {"..."}
}
```

### HTTPSuite

An HTTP tests suite contains a base configuration and list of tests.

|Name|Type|Required|Description|
|-|-|-|-|
|base|`HTTPBase`|✔️|Base description of the tests in this suite like base URL or HTTP headers to add too all requests|
|tests|`[]HTTPTest`|✔️|List of tests to run.|

`HTTPBase`

|Name|Type|Required|Description|
|-|-|-|-|
|url|`string`|✔️|Base URL prepended to all `path` in each test request.|
|headers|`map[string]string`||List of headers to add to every test in this suite. Each test can overwrite a header set at this level.|

`HTTPTest`

|Name|Type|Required|Description|
|-|-|-|-|
|name|`string`|✔️|Name used to uniquely identify this test in the suite.|
|request|`HTTPRequest`|✔️|Description of the HTTP request to perform.|
|expect|`HTTPExpect`|✔️|Expected result of the HTTP request.|

`HTTPRequest`

|Name|Type|Required|Description|
|-|-|-|-|
|path|`string`||Path appended to the base `url` set in the `HTTPBase`. Defaults to `/`.|
|method|`string`||HTTP method of the request. Defaults to `GET`.|
|headers|`map[string]string`||HTTP headers of the request.|
|multipart|`map[string]string`||Multipart content of the request. Values started with a `@` are files.|
|formURLEncoded|`map[string][]string`||Form data as application/x-url-encoded format.|
|body|`string`||Request body. Can be a file, if starting with the character `@`.|


`HTTPExpect`

|Name|Type|Required|Description|
|-|-|-|-|
|statusCode|`int`|✔️|Expected HTTP status code.|
|headers|`map[string]string`||Expected HTTP headers.|
|jsonDocument|`string`||Expected JSON document to be returned. Format is not taken into account, only values are compared. Can be an inline string value (escaped JSON), or a separate JSON file, if starting with the character `@`.|
|jsonSchema|`string`||Expected JSON schema (1) to be returned.  Can be a file, if starting with the character `@`.|
|jsonValues|`string`||Specific JSON values (2) to be returned.  Can be a file, if starting with the character `@`.|

> (1) - see http://json-schema.org/ for more info.

> (2) - A set of jq queries : see [this link](https://github.com/jmoiron/jsonq/blob/e874b168d07ecc7808bc950a17998a8aa3141d82/README.md) for more info.

### GRPC Suite

Work in progress...

# Example

This example shows how to create an HTTP tests suite file that will run 2 tests every 12h :
- The first one is a `GET http://127.0.0.1:8080/` and expects a 200 JSON response `{"key": "value"}`.
- The second test is a `POST http://127.0.0.1:8080/echo` with a JSON body containing `{"key":"value"}`. It expects a JSON response with a Content-Length of 14 and matching the JSON schema `suites/schemas/schema.json`.

```json
{
    "name": "Service 1 HTTP tests suite",
    "type": "HTTP",
    "runEvery": "12h",
    "slackWebhook": "https://hooks.slack.com/services/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",
    "suite": {
        "base": {
            "url": "http://127.0.0.1:8080",
            "headers": {
                "Accept-Encoding": "application/json"
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
                    "headers": {
                        "Content-Type": "application/json",
                        "Content-Length": "15"
                    },
                    "jsonDocument": {"key": "value"}
                }
            },
            {
                "name": "echo",
                "request": {
                    "path": "/echo",
                    "method": "POST",
                    "headers": {
                        "Content-Type": "application/json"
                    },
                    "body": {"key":"value"}
                },
                "expect": {
                    "statusCode": 200,
                    "headers": {
                        "Content-Type": "application/json",
                        "Content-Length": "15"
                    },
                    "jsonSchema": "@suites/schemas/schema.json"
                }
            }
        ]
    }
}

```
