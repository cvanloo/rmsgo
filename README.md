# Go remoteStorage Library

An implementation of the
[remoteStorage](https://datatracker.ietf.org/doc/html/draft-dejong-remotestorage-21)
protocol written in Go.

```sh
go get -u github.com/cvanloo/rmsgo
```

## Example Usage

```go
package main

import (
    "os"
    "github.com/cvanloo/rmsgo"
)

const (
    PersistFile = "/var/rms/persist"
    RemoteRoot  = "/storage/"
    StorageRoot = "/var/rms/storage/"
)

func main() {
    opts, err := rmsgo.Configure(RemoteRoot, StorageRoot)
    if err != nil {
        log.Fatal(err)
    }
    opts.UseErrorHandler(func(err error) {
        log.Fatalf("remote storage: unhandled error: %v", err)
    })
    opts.UseAuthentication(func(r *http.Request, bearer string) (rmsgo.User, bool) {
        // [!] TODO: Your authentication logic here...
        //       Return one of your own users.
        return rmsgo.ReadWriteUser{}, true
    })

    persistFile, err := os.Open(PersistFile)
    if err != nil {
        log.Fatal(err)
    }

    // Restore server state
    err = rmsgo.Load(persistFile)
    if err != nil {
        log.Fatal(err)
    }

    defer func() {
        // At shutdown: persist server state
        err = rmsgo.Persist(persistFile)
        if err != nil {
            log.Fatal(err)
        }
    }()

    // Register remote storage endpoints to the http.DefaultServeMux
    rmsgo.Register(nil)
    http.ListenAndServe(":8080", nil) // [!] TODO: Use TLS
}
```

## With Request Logging

```go
func logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        lrw := rmsgo.NewLoggingResponseWriter(w)

        // [!] pass request on to remote storage server
        next.ServeHTTP(lrw, r)

        duration := time.Since(start)

        // - Mom! Can we have slog?
        // - No darling, we have slog at home.
        // slog at home:
        log.Printf("%v", map[string]any{
            "method":   r.Method,
            "uri":      r.RequestURI,
            "duration": duration,
            "status":   lrw.Status,
            "size":     lrw.Size,
        })
    })
}

func main() {
    opts, err := rmsgo.Configure(RemoteRoot, StorageRoot)
    if err != nil {
        log.Fatal(err)
    }

    // [!] Register custom middleware
    opts.UseMiddleware(logger)

    // [!] Other configuration...

    rmsgo.Register(nil)
    http.ListenAndServe(":8080", nil) // [!] TODO: Use TLS
}
```

## All Configuration Options

- \[Required] `Setup` 
  - remoteRoot: URL path below which the server is accessible. (e.g. "/storage/")
  - storageRoot: Location on server's file system to store remoteStorage documents. (e.g. "/var/rms/storage/")
- \[Recommended] `UseAuthentication` configure how requests are authenticated and control access permissions of users.
- \[Recommended] `UseAllowedOrigins` allow-list of hosts that may make requests to the server. Per default any host is allowed.
- \[Optional] `UseAllowOrigin` for more control, specify a function that decides based on the request if it is allowed or not. If this option is specified, `UseAllowedOrigins` has no effect.
- \[Not Recommended] `AllowAnyReadWrite` allow even unauthenticated requests to create, read, and delete any documents on the server. Has no effect if `UseAuthentication` is specified.
- \[Optional] `UseErrorHandler` to catch unhandled errors. Default behavior is to `log.Printf` the error.
- \[Optional] `UseMiddleware` to intercept requests before they are passed to the remote storage handler.

`Register` registers the remote storage handler to a ServeMux.
