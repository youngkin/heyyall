[![Build Status](https://travis-ci.org/youngkin/heyyall.svg?branch=master)](https://travis-ci.org/youngkin/heyyall) [![Go Report Card](https://goreportcard.com/badge/github.com/youngkin/heyyall)](https://goreportcard.com/report/github.com/youngkin/heyyall)

# heyyall (Hey Y'all!)
<img src="heyyall.jpg" width="120" height="120" style="float: left; margin: 3px 20px 3px 0px; border: 1px solid #000000;">

`heyyall` is an HTTP load generator inspired by ['hey'](https://github.com/rakyll/hey). As the name suggests, `heyyall` says `hey` to multiple endpoints in a single test execution. It also supports multiple operations per endpoint, i.e., GET, POST, PUT, and DELETE. `hey` is limited to a single endpoint and operation per test execution. The metrics generated for each test execution are similar to `hey`, but summary and detail metrics for each endpoint are also generated.

`heyyall` was created to facilitate testing a set of related services each of which expose their own HTTP endpoints and resources. In order to test the application as a whole while exercising the full range of capability a more powerful tool was needed.

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
    "Endpoints": [
        {
            "URL": "http://accountd.kube/users/1",
            "Method": "GET",
            "RqstBody": "",
            "RqstPercent": 100
        }
    ]
}
```

A more sophisticated configuration can target multiple endpoints and perform multiple operations:

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
* Download a binary
  * [Windows 386](https://drive.google.com/open?id=1Wp-2hXBDixR4mBBxjiDcGOA81kgEAMH4)
  * [Windows AMD64](https://drive.google.com/open?id=1lVOQ6FuM2BYYYEMRisGM1t0CWOcRRIVQ)
  * [Linux AMD64](https://drive.google.com/open?id=1kdJrjwgJhpLK6D1u904s_XdUPcfQaA0N)
  * [Linux ARM](https://drive.google.com/open?id=1DIouRKCHaJLsPudyuMxG00F-NOgrMR6d)
  * [Mac(Darwin) AMD64](https://drive.google.com/open?id=14QxEznpurlYKMkw5lM9S5OlkhRUcp7n5)
* Homebrew support is planned for the future
  
# Usage

Running `./heyyall -help` provides the following usage message:

```
Usage: heyyall -config <ConfigFileLocation> [flags...]

Options:
  -loglevel Logging level. Default is 'WARN' (2). 0 is DEBUG, 1 INFO, up to 4 FATAL
  -detail   Detail level of output report, 'short' or 'long'. Default is 'long'
  -nf       Normalization factor used to compress the output histogram by eliminating long tails.
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
  -cpus     Specifies how many CPUs to use for the test run. The default is 0 which specifies that
            all CPUs should be used.
  -help     This usage message
  ```

One command line flag above is worth a little more discussion, the `nf` or "Normalization Factor" flag. 

Some endpoints may exhibit widely varying response times, from as little as a few microseconds to over a second. This can lead to a relatively useless histogram being generated when the test run completes. Here's an example:

```
./heyyall -config "testdata/oneEP1000Rqst.json" -loglevel 2 -detail "short"                                                  

Response Time Histogram (seconds):
	Latency		Number of Observations
	-------		----------------------
	[ 0.508]	    953	>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	[ 1.015]	     10	>
	[ 1.523]	     34	>>>>
	[ 2.030]	      0
	[ 2.538]	      0
	[ 3.045]	      0
	[ 3.553]	      0
	[ 4.060]	      0
	[ 4.568]	      0
	[ 5.076]	      1
```

In this execution over 95% of the responses are in a single histogram bin. This isn't very helpful, hence the Normalization Factor. Specifying `-nf 10` has the following effect on the generated histogram:

```
./heyyall -config "testdata/oneEP1000Rqst.json" -loglevel 2 -detail "short" -nf 10                                     

Response Time Histogram (seconds):
	Latency		Number of Observations
	-------		----------------------
	[ 0.023]	      0
	[ 0.046]	     24	>>>>>>>
	[ 0.069]	    105	>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	[ 0.092]	    345	>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	[ 0.115]	    256	>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	[ 0.138]	    132	>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	[ 0.161]	     56	>>>>>>>>>>>>>>>>
	[ 0.184]	     43	>>>>>>>>>>>>
	[ 0.207]	     10	>>>
	[ 0.230]	      7	>>
	[ 5.136]	     22	>>>>>>
```

Instead of 0.5 second bin widths, the widths are about 0.023 seconds. With the narrower widths come much more detail about the majority of the response latencies. However, there is a relatively long tail in both test executions. There are several responses with latencies over 1 second. These can be seen a little clearer in the first histogram. In the second histogram we lose this detail. All we see are that there were a total of 22 requests with latencies over 0.23 seconds. 

Changing the Normalization Factor allows you to decide where in the range of response latencies you want to see the finer grained detail. It should be noted that with a narrower range of response latencies you may not need to specify the Normalization Factor at all.

# Configuration

Specifying `heyyall`'s runtime behavior is done via a configuration file as shown above. The configuration can be quite simple or quite complex depending on your needs. Configuration is specified via a JSON file. In general the JSON is specified as follows:

``` 
{
    "RqstRate": <Integer, specifies the request rate per second>,
    "MaxConcurrentRqsts": <Integer, specifies how many requests can be run concurrently>,
    "RunDuration": <String, specifies the length of the run. Must be `0s` if `NumRequests` is specified.>,
    "NumRequests": <Integer, specifies the total number of requests to be made. Must be `0` if `RunDuration` is specified>,
    "Endpoints": [
        {
            "URL": <String, the resource URL>,
            "Method":<String, the HTTP method. One of `GET`, `POST`, `PUT`, or `DELETE`>,
            "RqstBody": <String, the body of the request, e.g., the content to be `POST`ed>,
            "RqstPercent": <Integer, the relative percent of the total requests will be made to this endpoint and method>,
        },
        {
           ...
        }
    ]
}
```

There are a couple of items of note:

1. `RunDuration` and `NumRequests` are mutually exclusive.
2. The total of `RqstPercent` across all endpoints must sum to 100, as in 100%.
3. `MaxConcurrentRqsts` must be greater than or equal to the number of `Endpoints` specified. This is based on the assumption that specifying an `Endpoint` means the intention is to execute requests against that `Endpoint`. If the condition specified here isn't met than at least one `Endpoint` won't get requests. This is an artifact of the implementation, but it seems like a reasonable restriction.

# Runtime behavior

Unsurprisingly, the configuration affects the runtime behavior of the application. 

The design of the application calls for `RqstRate`, `MaxConcurrentRqsts`, and if appropriate, `NumRequests` to be split as evenly as possible across all configured `Endpoints`. As noted previously, `NumRequests` and `RunDuration` are mutually exclusive so the impact of `NumRequests` is only an factor when it is specified.

The results of the calculations that allocate these resources across `Endpoints` can result in multiple, concurrent, requests being sent to a single `Endpoint`. If this occurs then the total number of requests allocated to a single `Endpoint` will be split among the concurrent `Endpoint` executions. The `RqstRate` is likewise split across all `Endpoints` as well as among concurrent executions to a single `Endpoint`. If specified, `RunDuration` is the same for all `Endpoints`.

As a result of the above, the actual values specified for `RqstRate`, `MaxConcurrentRqsts`, and `NumRequests` can turn out to be more along the lines of a suggestion rather than a strict specification. This is due to rounding errors resulting from non-integer values resulting from calculations that are performed to allocate the `RqstRate` and the other config values across the `Endpoints`. For example, if 3 `Endpoints` are specified and the value for `MaxConcurrentRqsts` is 4 the calculation of concurrent requests per `Endpoint` is 1.33.... These kinds of results will always be rounded up. So the `Endpoint`s will have 2 concurrent requests each for an overall `MaxConcurrentRqsts` of 6, not 4. Keep this in mind when specifying the values for `RqstRate`, `MaxConcurrentRqsts`, `NumRequests`, and the overall number of configured `Endpoints`. The application does print warning messages for every calculation that is rounded up. For example the log messages below show the rounding that occurred for 3 different endpoints:

```
May 12 15:09:19.000 WRN EP: http://accountd.kube/users: epConcurrency, 1, was rounded up. The calcuation result was 0.990000
May 12 15:09:19.000 WRN EP: http://accountd.kube/users/1: epConcurrency, 1, was rounded up. The calcuation result was 0.990000
May 12 15:09:19.000 WRN EP: http://accountd.kube/users/2: epConcurrency, 2, was rounded up. The calcuation result was 1.020000
```

Messages like this will also be printed when `RqstRate` and `NumRqsts` are rounded up.

To manage request rate the implementation uses Go's `time.Sleep()` function. As documented:

    Sleep pauses the current goroutine for at least the duration d.

What this means is that `time.Sleep()` may and likely will sleep longer than specified. This behavior means `RunDuration` also turns out to be more of a suggestion rather than a strict specification. So the actual run time of a test execution will be a little longer than specified. How much longer increases with the length of `RunDuration`. For example:

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

# Future plans

1. Currently the application generates only summary level statistics per endpoint instead of breaking them out by operation such as GET or PUT. This will be changed.
2. More metrics will be added to gain feature parity with `hey`. Mainly this means metrics for latency distribution (i.e., quantiles) and high level TCP/IP and HTTP related metrics like DNS lookup latencies and HTTP request write latencies.
3. Support for other configuration and output format types may be added, for example YAML and output in CSV format could be added.
4. Ability to specify the number of requests to be run at an endpoint level. If added this would be a strict specification in the sense that measures will be taken to ensure that the exact number of requests will be run and restrictions will be put in place to ensure related calculations don't have non-integer results.
