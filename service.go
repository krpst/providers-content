package main

import (
	"context"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	defaultTimeOut = 10 * time.Second
)

// ContentProvider provides some content based on user IP address.
type ContentProvider interface {
	// Fetch returns ContentItem.
	Fetch(ctx context.Context, userIP string, count, offset int) ([]*ContentItem, error)
}

// ContentProviderService is an implementation of ContentProvider.
type ContentProviderService struct {
	contentClients       map[Provider]Client
	config               ContentMix
	timeOut              time.Duration
	currentConfigsNumber int
}

// NewContentProviderService returns a new instance of ContentProviderService.
func NewContentProviderService(
	contentClients map[Provider]Client,
	config ContentMix,
	opts ...ConfigOption,
) ContentProvider {
	s := &ContentProviderService{
		contentClients:       contentClients,
		config:               config,
		timeOut:              defaultTimeOut,
		currentConfigsNumber: len(config),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// ConfigOption configures the services.
type ConfigOption func(*ContentProviderService)

// WithTimeOut provides a specific timeout.
func WithTimeOut(timeOut time.Duration) ConfigOption {
	return func(pr *ContentProviderService) {
		pr.timeOut = timeOut
	}
}

// Fetch returns ContentItem.
func (pr *ContentProviderService) Fetch(ctx context.Context, userIP string, count, offset int) ([]*ContentItem, error) {
	ctx, cancel := context.WithTimeout(ctx, pr.timeOut)
	defer cancel()

	var mutex sync.Mutex
	ch := make(chan bool)
	// we should fill responses in order, to discard values after errors.
	responses := make([]ProviderResponse, count)

	group, _ := errgroup.WithContext(ctx)
	for i := 0; i < count; i++ {
		config := pr.provideConfig(i + offset)
		number := i

		group.Go(func() error {
			providerContent, err := pr.contentClients[config.Type].GetContent(userIP, 1)
			if err != nil && config.Fallback != nil {
				providerContent, err = pr.contentClients[*config.Fallback].GetContent(userIP, 1)
			}

			mutex.Lock()
			defer mutex.Unlock()

			responses[number] = ProviderResponse{
				Content: providerContent,
				Error:   err,
			}

			ch <- true
			return nil
		})
	}

	finalContent := make([]*ContentItem, 0, count)

	pr.waitForResults(ch, responses, count)

	// check responses by order
	for _, response := range responses {
		if response.Error != nil {
			break
		}
		finalContent = append(finalContent, response.Content...)
	}

	return finalContent, nil
}

// provides a config based on config number.
func (pr *ContentProviderService) provideConfig(configNumber int) ContentConfig {
	return pr.config[configNumber%pr.currentConfigsNumber]
}

// waitForResults waits for results.
// Function can exit early:
// * by timeout;
// * if we have errors before not ended requests - there are no use to wait for all requests.
func (pr *ContentProviderService) waitForResults(ch chan bool, responses []ProviderResponse, count int) {
	chTimeout := make(chan bool)

	// exit by timeout
	time.AfterFunc(pr.timeOut, func() { chTimeout <- true })

	providersDone := 0
	for {
		select {
		case <-ch:
			providersDone++

			previousProvidesReturnedContent := true
			for _, response := range responses {
				// if previous providers returned an error - we shouldn't wait for others.
				if previousProvidesReturnedContent && response.Error != nil {
					return
				}

				previousProvidesReturnedContent = response.Content != nil && response.Error != nil
			}

			// all providers returned content
			if providersDone == count {
				return
			}
		case <-chTimeout:
			log.Printf("stop waiting for providers - got timeout")
			return
		}
	}
}
