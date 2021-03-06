# Sample Data API Server

## Purpose

This project is a hypothetical implementation of an 
API-server.

### Build

Run `go build`

### Usage

```
./data-api-server --port 8080 --datadir /data/api
```
More options:

```bash
Usage of ./data-api-server:
  -datadir string
       	Path to data directory (default "/tmp")
  -flushlog string
       	sets the flush trigger level (default "none")
  -host string
       	IP to bind to (default "0.0.0.0")
  -log string
       	sets the logging threshold (default "info")
  -port int
       	Port to listen to (default 8080)
  -redis string
       	Redis <host>:<port>:<db> (default "127.0.0.1:6379:0")
  -stderr
       	outputs to standard error (stderr)
```

Redis is somewhat optional, it will simply try to connect to the default host on each received event, fail, and print a log. The event will get stored.

## Event Types
Event types are registered in `main.go`. Valid events are `session_start`, `session_end` and `link_clicked`. The `EventType` struct is defined in `server/event.go`:
```go
type EventType struct {
	Name string
	Storage *Storage
}
```
Configuration per `EventType` can be added in the future. (Like separate rate-limit or validation options, list of expected/required params, etc)


## API Format

This version of the API always uses HTTP GET.

###### HTTP
  - Pro: Cleartext, infinitely extensible protocol
  - Con: Cleartext, infinitely extensible protocol (**FIXME** maybe?)

###### HTTP GET
  - Pro: Easier to log/trace/sniff by third-party tools since all of the data is in the same place. For instance you can enable web-request logging in the load balancer and automatically get a backup of all your API calls.
  - Pro: Easier to test/use by non-developers (ie. in a web browser)
  - Con: Does not support lengthy data as opposed to POST or PUT
  - Con: Can inadvertently get cached (In our current implementation, cache-buster GET parameters would get stored as well)
  - Con: Escaping may become a problem when hand-testing

## Request Format
The request format is:
```
/v1/<event>?ts=<timestamp>&param1=something&param2=something_else
```

`ts` is a unix-timestamp (UTC) in seconds. Future-timestamps, and past-timestamps beyond 1 day will get overwritten. There can be infinite (well, as long as they fit in the HTTP GET) number of additional parameters.

## Response
If the response is `HTTP 200 OK`, then the event is valid and it's probably stored. Response content is simply the word "Accepted". `HTTP 400` responses are given for invalid events. 


## Storage Format

The file format is TSV with embedded JSON, first column is the received timestamp of the event in nanoseconds, and second column is the JSON data.

The files are stored in `datadir` in this format:

```
<datadir>/<YYYY>/<MM>/<DD>/<HH>_<EventType>.tsv
```

To change the format, edit `DIRECTORY_FORMAT` and `FILE_FORMAT` in `storage.go`.

- If event types in storage are to be queried separately instead of looking at all the events of a specific date, the `{event}` field can be moved up the chain to its own directory.
- If a log collector (like Apache Flume, Fluentd, etc) is to be used to push data to storage, the directory scheme can be abandoned altogether and each event type can have its own file per-hour (or per-day).
- If the files are to be stored in AWS S3, using a random single-letter prefix (partition key) before the `YYYY` field is recommended. This would avoid hot partitions in the storage layer and prevent I/O bottlenecks. This would require additional development in `storage.go`. (See [Amazon S3 Performance Tips & Tricks](https://aws.amazon.com/blogs/aws/amazon-s3-performance-tips-tricks-seattle-hiring-event/)) 



## Caveats
- Bulk mode is not supported. A way to send bulk (ie. previously stored) events can be implemented, preferably with a protocol that natively supports batching.
- Events are validated in the same goroutine as the request, because event validation is currently a few string operations. This way we can tell the client if their event is "valid" or not using the HTTP status code in the response.
- If time-consuming validation tasks are needed, the server should always return `200 OK` on received data and do the actual processing/validation in a worker-pool.
- Events are written to storage in a single-threaded manner (one goroutine per file) due to the nature of the CSV-format. If we were to switch the filesystem with a data storage service, a worker-pool should be used so that events can be written in parallel.
- Client IP and other related metadata is not stored.
- Authentication (API key) is not implemented.
- Rate-limiting is not implemented, but it should be fairly easy using proper middleware.
- CSV escapes quotes, which is not good because the data is a JSON map and always has quotes in it. Another format (at least custom CSV dialect with quote-escaping disabled) would've been better.
- Tests are not provided.


## Statistics
- Redis is used to store aggregated event counts.
- Counts are stored in sorted sets by type, in the keys `eventsByType:<EventType>`. (This probably won't scale at all since adding elements is expensive)
- To get overall counts, make a request to the `/stats` endpoint:
```
$ curl 'http://:8080/stats'|jq .
```
```json
{
  "stats": {
    "link_clicked": 0,
    "session_end": 1,
    "session_start": 13
  }
}
```

- To get counts between two time periods (second-precision), make a request to the same endpoint but include `since` and/or `until` parameters:
```
$ curl 'http://:8080/stats?since=1472063303'|jq .
```
```json
{
  "since": 1472063303,
  "stats": {
    "link_clicked": 0,
    "session_end": 1,
    "session_start": 7
  },
  "until": 0
}
```
- Each event is assigned a unique-per-type id (`INCR eventCounter:<EventType>` is used) but this costs us a second call to Redis. A Lua-script can be used to piggyback the `INCR` and `ZADD` calls.
- Stats are collected in the same goroutine, an asynchronous solution would be to use a worker pool on a buffered channel.
- The data can actually be stored in Redis as well, and time-slices of it can be fetched semi-efficiently.
- A mechanism to expire old elements in the sorted sets is not implemented. The statistics will eventually become inconsistent, as Redis evicts keys.


## SDKs
- SDKs should store and retry each event until they get an `HTTP 200` from the server.
- If `HTTP 400` response is encountered, the event is deemed invalid by the server and should be discarded without further retries.
- Response body and/or headers (like `Content-Type`) are subject to change and should not be checked by the SDKs.

