package console

import (
	"fmt"
	"net/url"

	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
)

// Console manages references and URLs for an openshift console.
type Console struct {
	BaseURL *url.URL
	e       *engine.Engine
}

func New(baseURL *url.URL, e *engine.Engine) *Console {
	return &Console{BaseURL: baseURL, e: e}
}

func (c *Console) converters() (converters []korrel8.RefConverter) {
	for _, d := range c.e.Domains() {
		if cvt, ok := d.(korrel8.RefConverter); ok {
			converters = append(converters, cvt)
		}
		s, _ := c.e.Store(d.String())
		if cvt, ok := s.(korrel8.RefConverter); ok {
			converters = append(converters, cvt)
		}
	}
	return converters
}

func (c *Console) RefConsoleToStore(consoleRef uri.Reference) (korrel8.Class, uri.Reference, error) {
	for _, cvt := range c.converters() {
		if class, ref, err := cvt.RefConsoleToStore(consoleRef); err == nil {
			return class, ref, err
		}
	}
	return nil, uri.Reference{}, fmt.Errorf("cannot convert console URI: %v", consoleRef)
}

func (c *Console) RefStoreToConsole(storeRef uri.Reference) (uri.Reference, error) {
	for _, cvt := range c.converters() {
		if ref, err := cvt.RefStoreToConsole(storeRef); err == nil {
			return ref, err
		}
	}
	return uri.Reference{}, fmt.Errorf("cannot convert console URI: %v", storeRef)
}
