[![Build Status](https://travis-ci.org/apiaryio/heroku-datadog-drain-golang.svg?branch=master)](https://travis-ci.org/apiaryio/heroku-datadog-drain-golang)

# Heroku Datadog Drain

Golang version of [NodeJS](https://github.com/ozinc/heroku-datadog-drain)

Funnel metrics from multiple Heroku apps into DataDog using statsd.

## Supported Heroku metrics:

- Heroku Router response times, status codes, etc.
- Application errors
- Custom metrics
- Heroku Dyno [runtime metrics](https://devcenter.heroku.com/articles/log-runtime-metrics)
- (beta) Heroku Runtime Language Metrics - we add support for [golang](https://devcenter.heroku.com/articles/language-runtime-metrics-go#getting-started) used in Heroku, next step add this to send to Datadog too for self monitoring app.

## Get Started

### Clone the Github repository

```bash
git clone git@github.com:busbud/heroku-datadog-drain-golang.git
cd heroku-datadog-drain-golang
```

### Setup Heroku, add your statsd url and basic auth creds

```
heroku create
heroku config:set BASIC_AUTH_USERNAME=user BASIC_AUTH_PASSWORD=pass STATSD_URL=somewhere.com
```

> **OPTIONAL**: Setup Heroku build packs, including the Datadog DogStatsD client.
If you already have a StatsD client running, see the STATSD_URL configuration option below.


```
heroku buildpacks:add heroku/go
heroku config:set HEROKU_APP_NAME=$(heroku apps:info|grep ===|cut -d' ' -f2)
```

Don't forget [set right golang version](https://devcenter.heroku.com/articles/go-support#go-versions).

```
heroku config:add GOVERSION=go1.10
```

### Deploy to Heroku.

```
git push heroku master
heroku ps:scale web=1
```

### Add the Heroku log drain using the app slug and password created above.

```
heroku drains:add https://<username>:<password>@<this-log-drain-app-slug>.herokuapp.com?app=<your-app-slug> --app <your-app-slug>
```

## Configuration
```bash
STATSD_URL=..             # Required. Set to: localhost:8125
DATADOG_API_KEY=...       # Required if STATSD_URL is not set. Datadog API Key - https://app.datadoghq.com/account/settings#api
BASIC_AUTH_USERNAME=...   # Required. Basic auth username to access drain.
BASIC_AUTH_USERNAME=...   # Required. Basic auth password to access drain.
DATADOG_DRAIN_DEBUG=..    # Optional. If DEBUG is set, a lot of stuff will be logged :)
EXCLUDED_TAGS: path,host  # Optional. Recommended to solve problem with tags limit (1000)
```
The rationale for `EXCLUDED_TAGS` is that the `path=` tag in Heroku logs includes the full HTTP path - including, for instance, query parameters. This makes very easy to swarm Datadog with numerous distinct tag/value pairs; and Datadog has a hard limit of 1000 such distinct pairs. When the limit is breached, they blacklist the entire metric.

## Heroku settings

You need use Standard dynos and better and enable `log-runtime-metrics` in heroku labs for every application.

```bash
heroku labs:enable log-runtime-metrics -a APP_NAME
```

This adds basic metrics (cpu, memory etc.) into logs.

## Custom Metrics

If you want to log some custom metrics just format the log line like following:

```
app web.1 - info: responseLogger: metric#tag#route=/parser metric#request_id=11747467-f4ce-4b06-8c99-92be968a02e3 metric#request_length=541 metric#response_length=5163 metric#parser_time=5ms metric#eventLoop.count=606 metric#eventLoop.avg_ms=515.503300330033 metric#eventLoop.p50_ms=0.8805309734513275 metric#eventLoop.p95_ms=3457.206896551724 metric#eventLoop.p99_ms=3457.206896551724 metric#eventLoop.max_ms=5008
```
We support:

 * `metric#` and `sample#` for gauges
 * `metric#tag` for tags.
 * `count#` for counter increments
 * `measure#` for histograms

more info [here](https://docs.datadoghq.com/guides/dogstatsd/#data-types)

## Overriding prefix and tags with drain query params

To change the prefix use the drain of form:
`https://<your-app-slug>:<password>@<this-log-drain-app-slug>.herokuapp.com?prefix=abcd.`

To change tags use the drain of form:
`https://<your-app-slug>:<password>@<this-log-drain-app-slug>.herokuapp.com?tags=xyz,abcd`
