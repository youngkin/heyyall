[![Build Status](https://travis-ci.org/youngkin/heyyall.svg?branch=master)](https://travis-ci.org/youngkin/heyyall) [![Go Report Card](https://goreportcard.com/badge/github.com/youngkin/heyyall)](https://goreportcard.com/report/github.com/youngkin/heyyall)

# heyyall (Hey Y'all!)
<img src="heyyall.jpg" width="120" height="120" style="float: left; margin: 3px 20px 3px 0px; border: 1px solid #000000;">

`heyyall` is an HTTP load generator inspired by ['hey'](https://github.com/rakyll/hey). As the name suggests, `heyyall` says `hey` to multiple endpoints in a single test execution. It also supports multiple operations per endpoint, i.e., GET, POST, PUT, and DELETE. `hey` is limited to a single endpoint and operation per test execution. The metrics generated for each test execution are similar to `hey`, but summary and detail metrics for each endpoint are also generated.

`heyyall` was created to facilitate testing a set of related services each of which expose their own HTTP endpoints and resources. In order to test the application as a whole while exercising the full range of capability a more powerful tool was needed.

`heyyall` can also be configured to work with HTTPS services via a client certificate and private key.

# Examples

Running a test can be accomplished by running:

```
./heyyall -config <SomeConfigFile>
```

Configuration may be as simple as targeting a single endpoint:

``` JSON
{
    "RqstRate": 0,
    "MaxConcurrentRqsts": 50,
    "RunDuration": "0s",
    "NumRequests": 1000,
    "KeyFile": "/path/to/private/key/file",
    "CertFile": "path/to/certificate/file",
    "Endpoints": [
        {
            "URL": "https://accountd.kube/users/1",
            "Method": "GET",
            "RqstBody": "",
            "RqstPercent": 100,
            "KeyFile": "/path/to/private/key/file",
            "CertFile": "path/to/certificate/file",
        }
    ]
}
```

A more sophisticated configuration can target multiple endpoints and perform multiple operations (this is not an HTTPS example):

``` JSON
{
    "RqstRate": 1000,
    "MaxConcurrentRqsts": 13,
    "RunDuration": "0s",
    "NumRequests": 4,
    "Endpoints": [
        {
            "URL": "http://accountd.kube/users",
            "Method": "GET",
            "RqstBody": "",
            "RqstPercent": 50
        },
        {
            "URL": "http://accountd.kube/users",
            "Method": "POST",
            "RqstBody": "{\"accountid\":1,\"name\":\"Brian Wilson\",\"email\":\"goodvibrations@gmail.com\",\"role\":1,\"password\":\"helpmerhonda\"}",
            "RqstPercent": 25
        },
        {
            "URL": "http://accountd.kube/users/1",
            "Method": "PUT",
            "RqstBody": "{\"accountid\":1,\"id\":1,\"name\":\"BeachBoy Brian Wilson\",\"email\":\"goodvibrations@gmail.com\",\"role\":1,\"password\":\"helpmerhonda\"}",
            "RqstPercent": 25
        }
    ]
}
```

# How to get it/build it

`heyyall` is written in Go. There are several ways to install the program.

* If you have a Go development environment you can:
  * Clone the respository and build it yourself
  * Run `go install github.com/youngkin/heyyall`
* Download a binary from [releases page](https://github.com/youngkin/heyyall/releases). There are binaries for:
  * Windows
  * Linux AMD64 and ARM (v6 & v7)
  * MacOS
* Homebrew support is planned for the future
  
# Usage

Running `./heyyall -help` provides the following usage message:

```
Usage: heyyall -config <ConfigFileLocation> [flags...]

Options:
  -loglevel  Logging level. Default is 'WARN' (2). 0 is DEBUG, 1 INFO, up to 4 FATAL
  -out       Type of output report, 'text' or 'json'. Default is 'text'
  -nf        Normalization factor used to compress the output histogram by eliminating long tails.
             Lower values provide a finer grained view of the data at the expense of dropping data
             associated with the tail of the latency distribution. The latter is partly mitigated by
             including a final histogram bin containing the number of observations between it and
             the previous latency bin. While this doesn't show a detailed distribution of the tail,
             it does indicate how many observations are included in the tail. 10 is generally a good
             starting number but may vary depending on the actual latency distribution and range
             of latency values. The default is 0 which signifies no normalization will be performed.
             With very small latencies (microseconds) it's possible that smaller normalization values
             could cause the application to panic. Increasing the normalization factor will eliminate
             the issue.
  -cpus      Specifies how many CPUs to use for the test run. The default is 0 which specifies that
			 all CPUs should be used.
  -help     This usage message

  ```

A couple of these flags are worth discussiong in more detail. First, the `-out` flag. As stated in the usage text it is used to specify whether text or JSON output is desired. Text output is optimized to be human readable and it summarizes the low level details (e.g., full set of response latencies in a test run). JSON output is very detailed, can be voluminous, and is probably best consumed programatically if the text output is missing some desired detail. The `report.go` file in the `api` package contains the Go structs that control the JSON output.

The following shows an example of a test run specifiying text output:

``` text
Run Summary:
	        Total Rqsts: 2000
	          Rqsts/sec: 269.9186
	Run Duration (secs): 7.4096


Request Latency (secs): Min      Median   P75      P90      P95      P99
	                    0.0061   0.0544   0.1591   0.2660   0.5053   4.9642

Request Latency Histogram (secs):
	Latency   Observations
	[0.0110]     123	❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱
	[0.0220]     426	❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱
	[0.0330]     184	❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱
	[0.0440]     157	❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱
	[0.0550]     117	❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱
	[0.0660]      86	❱❱❱❱❱❱❱❱❱❱❱❱❱
	[0.0770]      70	❱❱❱❱❱❱❱❱❱❱❱
	[0.0881]      38	❱❱❱❱❱❱
	[0.0991]      37	❱❱❱❱❱❱
	[0.1101]      71	❱❱❱❱❱❱❱❱❱❱❱
	[0.1211]      47	❱❱❱❱❱❱❱
	[5.2166]     644	❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱



Endpoint Details(secs):
  http://accountd.kube/users:
	            Requests   Min        Median     P75        P90        P95        P99
	     GET:        260   0.0086     0.0675     0.1670     0.2533     0.4255     4.9257

  http://accountd.kube/users/1:
	            Requests   Min        Median     P75        P90        P95        P99
	     GET:        240   0.0077     0.0604     0.1585     0.2426     0.3642     4.4909

  http://accountd.kube/users/2000:
	            Requests   Min        Median     P75        P90        P95        P99
	  DELETE:       1000   0.0061     0.0512     0.1573     0.3047     0.5632     4.9944
	     GET:        500   0.0063     0.0496     0.1599     0.2578     0.4333     5.0647



Network Details (secs):
					Min      Median      P75      P90      P95      P99
	    DNS Lookup: 0.0000   0.0011   0.0019   0.0027   0.0029   0.0032
	TCP Conn Setup: 0.0000   0.0000   0.0010   0.0286   0.0926   0.1063
	 TLS Handshake: 0.0000   0.0000   0.0000   0.0000   0.0000   0.0000
	Rqst Roundtrip: 0.0060   0.0498   0.1540   0.2425   0.4063   4.9641
```

The other command line flag above is the `nf` or "Normalization Factor" flag.

Some endpoints may exhibit widely varying response times, from as little as a few microseconds to over a second. This can lead to a relatively useless histogram being generated when the test run completes. Here's an example:

```
./heyyall -config "testdata/oneEP1000Rqst.json" -loglevel 2 -out "text"

...

Request Latency Histogram (secs):
	Latency   Observations
	[0.5064]     940	❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱
	[1.0127]      50	❱❱❱❱❱
	[1.5191]       0
	[2.0255]       0
	[2.5318]       0
	[3.0382]       0
	[3.5446]       0
	[4.0509]       0
	[4.5573]       3
	[5.0637]       7	❱

...
```

In this execution 94% of the responses are in a single histogram bin. This isn't very helpful, hence the Normalization Factor. Specifying `-nf 10` has the following effect on the generated histogram:

```
./heyyall -config "testdata/oneEP1000Rqst.json" -loglevel 2 -out "text" -nf 10

...

Request Latency Histogram (secs):
	Latency   Observations
	[0.0189]       1
	[0.0378]     365	❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱
	[0.0567]     397	❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱
	[0.0756]     149	❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱❱
	[0.0945]      51	❱❱❱❱❱❱❱❱❱❱❱❱❱
	[0.1134]      10	❱❱❱
	[0.1323]       0
	[0.1512]       0
	[0.1701]       0
	[0.1890]       0
	[5.1241]      27	❱❱❱❱❱❱❱


...
```

Instead of 0.5 second bin widths, the widths are about 0.019 seconds. With the narrower widths come much more detail about the majority of the response latencies. However, there is a relatively long tail in both test executions. There are several responses with latencies over 5 seconds. These can be seen a little clearer in the first histogram. In the second histogram we lose this detail. All we see are that there were a total of 27 requests with latencies over 0.1134 seconds. 

Changing the Normalization Factor allows you to decide where in the range of response latencies you want to see the finer grained detail. It should be noted that with a narrower range of response latencies you may not need to specify the Normalization Factor at all.

# Configuration

Specifying `heyyall`'s runtime behavior is done via a configuration file as shown above. The configuration can be quite simple or quite complex depending on your needs. Configuration is specified via a JSON file. In general the JSON is specified as follows:

``` 
{
    "RqstRate": <Integer, specifies the request rate per second>,
    "MaxConcurrentRqsts": <Integer, specifies how many requests can be run concurrently>,
    "RunDuration": <String, specifies the length of the run. Must be `0s` if `NumRequests` is specified.>,
    "NumRequests": <Integer, specifies the total number of requests to be made. Must be `0` if `RunDuration` is specified>,
    "KeyFile": <String, specifies the path to a file containing a PEM encoded private key>,
    "CertFile": <String, specifies the path to a file containing a PEM encoded certificate>,
    "Endpoints": [
        {
            "URL": <String, the resource URL>,
            "Method":<String, the HTTP method. One of `GET`, `POST`, `PUT`, or `DELETE`>,
            "RqstBody": <String, the body of the request, e.g., the content to be `POST`ed>,
            "KeyFile": <String, specifies the path to a file containing a PEM encoded private key>,
            "CertFile": <String, specifies the path to a file containing a PEM encoded certificate>,
            "RqstPercent": <Integer, the relative percent of the total requests will be made to this endpoint and method>,
        },
        {
           ...
        }
    ]
}
```

There are a few items of note:

1. `RunDuration` and `NumRequests` are mutually exclusive.
2. The total of `RqstPercent` across all endpoints must sum to 100, as in 100%.
3. `MaxConcurrentRqsts` must be greater than or equal to the number of `Endpoints` specified. This is based on the assumption that specifying an `Endpoint` means the intention is to execute requests against that `Endpoint`. If the condition specified here isn't met than at least one `Endpoint` won't get requests. This is an artifact of the implementation, but it seems like a reasonable restriction.
4. `"KeyFile"` is optional and specifies a client's PEM encoded private key. It can be configured at both the global and Endpoint levels. If specified for an Endpoint it will override the global specification.
5. `"CertFile"` is optional and represent a client's PEM encoded public certificate. It can be configured at both the global and Endpoint levels. If specified for an Endpoint it will override the global specification.

The `config.go` file in the `api` package contains the Go struct definitions for the JSON configuration.



## HTTPS support

As mentioned above `heyyall` also supports client authentication and authorization via SSL on an HTTP request. The `"KeyFile"` and `"CertFile"` configuration fields provide the required information. These must both be PEM files.

The `internal/testhttpsserver` package contains the code for an HTTPS server that will authenticate and authorize a client certificate. This can be useful for testing `heyyall`'s HTTPS support. You will need a certificate and key files for both the server and client. It is possible to use the same certs/keys for both client and server.

# Runtime behavior

Unsurprisingly, the configuration affects the runtime behavior of the application. 

The design of the application calls for `RqstRate`, `MaxConcurrentRqsts`, and if appropriate, `NumRequests` to be split as evenly as possible across all configured `Endpoints`. As noted previously, `NumRequests` and `RunDuration` are mutually exclusive so the impact of `NumRequests` is only an factor when it is specified.

The results of the calculations that allocate these resources across `Endpoints` can result in multiple, concurrent, requests being sent to a single `Endpoint`. If this occurs then the total number of requests allocated to a single `Endpoint` will be split among the concurrent `Endpoint` executions. The `RqstRate` is likewise split across all `Endpoints` as well as among concurrent executions to a single `Endpoint`. If specified, `RunDuration` is the same for all `Endpoints`.

As a result of the above, the actual values specified for `RqstRate`, `MaxConcurrentRqsts`, and `NumRequests` can turn out to be more guidelines rather than a strict specification. This is due to rounding errors resulting from non-integer values resulting from calculations that are performed to allocate the `RqstRate` and the other config values across the `Endpoints`. For example, if 3 `Endpoints` are specified and the value for `MaxConcurrentRqsts` is 4 the calculation of concurrent requests per `Endpoint` is 1.33.... These kinds of results will always be rounded up. So the `Endpoint`s will have 2 concurrent requests each for an overall `MaxConcurrentRqsts` of 6, not 4. Keep this in mind when specifying the values for `RqstRate`, `MaxConcurrentRqsts`, `NumRequests`, and the overall number of configured `Endpoints`. The application does print warning messages for every calculation that is rounded up. For example the log messages below show the rounding that occurred for 3 different endpoints:

```
May 12 15:09:19.000 WRN EP: http://accountd.kube/users: epConcurrency, 1, was rounded up. The calcuation result was 0.990000
May 12 15:09:19.000 WRN EP: http://accountd.kube/users/1: epConcurrency, 1, was rounded up. The calcuation result was 0.990000
May 12 15:09:19.000 WRN EP: http://accountd.kube/users/2: epConcurrency, 2, was rounded up. The calcuation result was 1.020000
```

Messages like this will also be printed when `RqstRate` and `NumRqsts` are rounded up.

To manage request rate the implementation uses Go's `time.Sleep()` function. As documented:

    Sleep pauses the current goroutine for at least the duration d.

What this means is that `time.Sleep()` may and likely will sleep longer than specified. This behavior means `RunDuration` also turns out to be more of a guideline rather than a strict specification. So the actual run time of a test execution will be a little longer than specified. How much longer increases with the length of `RunDuration`. For example:

```
./heyyall -config "testdata/threeEPs33Pct.json"                                                                        

...

Run Results:
      "RunSummary": {
        "RqstRatePerSec": 120.76358089874817,
        "RunDuration": 11534934536,
        "RunDurationStr": "11.534934536s",
```

Note that even though a 10 second `RunDuration` was specified the actual run time was 11-plus seconds.

Most of these behaviors are a result of design decisions and as such can be changed with a different implementation. But alternate implementations may have their own idiosyncracies. If the behavior described here becomes an issue the design decisions can be revisited.

# Known issues

Some HTTPS services require TLS renegotiation. Up to and including Go 1.14 the Go crypto/tls implementation does not support TLS renegotiation. This may change as soon as the 1.15 release due out in August 2020. If a service does require TLS renegotiation a warning like the following will be printed and the request will not succeed.

```
WRN Requestor: error sending request error="Get \"https://prod.idrix.eu/secure/\": local error: tls: no renegotiation"
```

# Future plans

1. Support for other configuration and output format types may be added, for example YAML and output in CSV format could be added.
2. Ability to specify the number of requests to be run at an endpoint level. If added this would be a strict specification in the sense that measures will be taken to ensure that the exact number of requests will be run and restrictions will be put in place to ensure related calculations don't have non-integer results.
3. Ability to script scenarios comprised of multiple different requests to a single Endpoint. `heyyall` currently on supports a single request to a given endpoint.
4. Performance improvements may be needed. When compared to similar tools like `hey` it seems like the request throughput of `heyyall` is generally lower. It's not entirely clear that this is the case, but it needs more investigation.

# Similar tools

While I was familiar with tools like `hey` and `JMeter`, it turns out there vast universe of load generation tools out there. [Here's a great resource](https://github.com/denji/awesome-http-benchmark) that is up-to-date as of January 2020. Another tool that was brought to my attention is [Artillery](https://artillery.io/).
