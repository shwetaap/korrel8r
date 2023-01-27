package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/logs"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/templaterule"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

var (
	log = logging.Log()
	ctx = context.Background()
)

func restConfig() *rest.Config {
	cfg, err := config.GetConfig()
	if err == nil {
		cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 1000)
	}
	return must.Must1(cfg, err)
}

func k8sClient(cfg *rest.Config) client.Client {
	log.V(2).Info("create k8s client")
	return must.Must1(client.New(cfg, client.Options{}))
}

func newEngine() *engine.Engine {
	log.V(2).Info("create engine")
	cfg := restConfig()
	e := engine.New()
	for _, x := range []struct {
		d      korrel8r.Domain
		create func() (korrel8r.Store, error)
	}{
		{k8s.Domain, func() (korrel8r.Store, error) { return k8s.NewStore(k8sClient(cfg), cfg) }},
		{alert.Domain, func() (korrel8r.Store, error) { return alert.NewOpenshiftAlertManagerStore(ctx, cfg) }},
		{logs.Domain, func() (korrel8r.Store, error) { return logs.NewOpenshiftLokiStackStore(ctx, k8sClient(cfg), cfg) }},
	} {
		log.V(2).Info("add domain", "domain", x.d)
		s, err := x.create()
		if err != nil {
			log.Error(err, "error creating store", "domain", x.d)
		}
		e.AddDomain(x.d, s)
	}
	// Load rules
	for _, path := range *rulePaths {
		must.Must(loadRules(e, path))
	}
	return e
}

// printer prints in the format requested by --output
type printer struct{ Print func(any) }

func newPrinter(w io.Writer) printer {
	switch *output {

	case "json":
		return printer{Print: func(v any) {
			if b, err := json.Marshal(v); err != nil {
				fmt.Fprintf(w, "%v\n", err)
			} else {
				fmt.Fprintf(w, "%v\n", string(b))
			}
		}}

	case "json-pretty":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return printer{Print: func(v any) { must.Must(encoder.Encode(v)) }}

	case "yaml":
		return printer{Print: func(v any) { fmt.Fprintf(w, "---\n%s", must.Must1(yaml.Marshal(v))) }}

	default:
		must.Must(fmt.Errorf("invalid output type: %v", *output))
		return printer{}
	}
}

func (p printer) Append(objects ...korrel8r.Object) {
	for _, o := range objects {
		p.Print(o)
	}
}

// loadRules from a file or walk a directory to find files.
func loadRules(e *engine.Engine, root string) error {
	log.V(3).Info("loading rules from", "root", root)
	return filepath.WalkDir(root, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		ext := filepath.Ext(path)
		if !info.Type().IsRegular() || (ext != ".yaml" && ext != ".yml" && ext != ".json") {
			return nil // Skip file
		}
		log.V(3).Info("loading rules", "path", path)
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := templaterule.Decode(f, e); err != nil {
			return fmt.Errorf("%v:0 error loading rules: %v", path, err)
		}
		return nil
	})
}