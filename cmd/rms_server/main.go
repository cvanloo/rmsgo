package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/cvanloo/rmsgo"
)

const RemoteRoot = "/storage/"

var (
	varData     = flag.String("var", VarData, "Variable data directory")
	address     = flag.String("a", "", "Listener address")
	port        = flag.String("p", "8080", "Listener port")
	sroot       = flag.String("s", StorageRoot, "Storage directory (set based on `var' unless specified)")
	rroot       = flag.String("r", RemoteRoot, "Remote storage root")
	persistFile = flag.String("persist", PersistFile, "Restore server state from persistFile (set based on `var' unless specified)")
	origins     Origin
	allOrigins  = true
	help        = flag.Bool("h", false, "Print usage/help")
)

type Origin struct {
	Origins []string
}

var _ flag.Value = (*Origin)(nil)

func (o Origin) String() string {
	return fmt.Sprintf("%s", strings.Join(o.Origins, ", "))
}

func (o *Origin) Set(s string) error {
	origins := strings.Split(s, ",")
	for _, origin := range origins {
		o.Origins = append(o.Origins, strings.TrimSpace(origin))
	}
	return nil
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := rmsgo.NewLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r)
		duration := time.Since(start)
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
	flag.Var(&origins, "o", "Allowed origins (default is any)")
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *varData != VarData {
		if *sroot == StorageRoot {
			*sroot = *varData + Storage
		}
		if *persistFile == PersistFile {
			*persistFile = *varData + Persist
		}
	}

	fi, err := os.Stat(*persistFile)
	if err == nil && fi.IsDir() {
		*persistFile = path.Join(*persistFile, "persist.xml")
	}

	allOrigins = len(origins.Origins) == 0

	fmt.Println("--------------- CONFIG ---------------")
	fmt.Printf("   storage root `%s'\n", *sroot)
	fmt.Printf("    remote root `%s'\n", *rroot)
	fmt.Printf("   listening on `%s:%s'\n", *address, *port)
	if allOrigins {
		fmt.Println("allowed origins `*'")
	} else {
		fmt.Printf("allowed origins `%s'\n", origins)
	}
	fmt.Printf("   persist file `%s'\n", *persistFile)
	fmt.Println("--------------------------------------")

	if _, err := os.Stat(*sroot); err != nil {
		log.Fatalf("storage root does not exist: %v", err)
	}

	err = rmsgo.Configure(*rroot, *sroot,
		rmsgo.WithErrorHandler(func(err error) {
			log.Fatalf("remote storage: unhandled error: %v", err)
		}),
		rmsgo.WithMiddleware(logger),
		rmsgo.Optionally(!allOrigins, rmsgo.WithAllowedOrigins(origins.Origins)), // allow all is the default in opts
		rmsgo.WithAuthentication(func(r *http.Request, bearer string) (rmsgo.User, bool) {
			return rmsgo.UserReadWrite{}, true
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	fd, err := os.OpenFile(*persistFile, os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		log.Fatalf("failed to open or create persist file: %v", err)
	}
	err = rmsgo.Load(fd)
	if err != nil {
		if errors.Is(err, io.EOF) {
			log.Printf("server state was NOT restored: persist file is empty")
		} else {
			log.Fatalf("server state was NOT restored: %v", err)
		}
	}

	defer func() {
		_ = fd.Truncate(0)
		_, _ = fd.Seek(0, io.SeekStart)
		err := rmsgo.Persist(fd)
		if err != nil {
			log.Fatalf("failed to persist server state: %v", err)
		}
		log.Printf("wrote server state to persist file: %s", fd.Name())
	}()

	mux := http.NewServeMux()
	rmsgo.Register(mux)

	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%s", *address, *port),
		Handler: mux,
	}

	wg := sync.WaitGroup{}
	go func() {
		wg.Add(1)
		defer wg.Done()
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
		log.Println("received interrupt, shutting down...")
		err = srv.Shutdown(context.TODO())
		if err != nil {
			log.Printf("server shutdown with error: %v", err)
		}
	}
}
