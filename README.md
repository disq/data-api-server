# Sample Data API Server

## Purpose

This project is a hypothetical implementation of an 
API-server.

### Build

Run `go build`

### Usage

Run `./data-api-server --port 8080 --datadir /data/api`. More options:

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

## Event Types
Event types are defined in `main.go`. Currently defined events are: `session_start`, `session_end`, `link_clicked`.

## API Format

This version of the API always uses HTTP GET.

###### HTTP
  - Pro: Cleartext, infinitely extensible protocol
  - Con: Cleartext, infinitely extensible protocol (**FIXME** maybe?)

###### HTTP GET
  - Pro: Easier to log/trace/sniff by third-party tools since all of the data is in the same place. For instance you can enable web-request logging in the load balancer and automatically get a backup of all your API calls.
  - Pro: Easier to test/use by non-developers (ie. in a web browser)
  - Con: Does not support lengthy data as opposed to POST or PUT
  - Con: Can inadvertently get cached
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

The files are stored in `datadir` in this format:

```
<datadir>/<YYYY>/<MM>/<DD>/<HH>_<event name>.tsv
```

(**FIXME** add random-letter-of-the-alphabet partitioning?)

The file format is TSV with embedded JSON, first column is the timestamp of the event, and second column is the JSON data. (**FIXME** check Spark load formats)

## Statistics

**TODO**
