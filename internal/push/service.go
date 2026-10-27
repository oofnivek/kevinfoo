package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
)

var (
	ErrValidation    = errors.New("validation failed")
	ErrDisabled      = errors.New("push notifications are not configured")
	ErrNoSubscribers = errors.New("no subscribers for that user ID and topic")
)

// topicPattern accepts a UUID or any similarly long, hard-to-guess string.
// The topic is effectively the shared secret needed to push to a user, so a
// minimum length is enforced.
var topicPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{20,64}$`)

// allowedPushHosts are the push services browsers hand out endpoints for.
// The server POSTs to subscriber-supplied endpoints when sending, so
// anything else is rejected to keep it from being pointed at arbitrary
// (e.g. internal) URLs.
var allowedPushHosts = []string{
	"fcm.googleapis.com",        // Chrome, Edge (Chromium), Brave, Opera
	"push.services.mozilla.com", // Firefox
	"push.apple.com",            // Safari
	"notify.windows.com",        // legacy Edge
}

const (
	maxSendConcurrency = 10
	sendTimeout        = 5 * time.Second
)

// Message is the payload delivered to the service worker.
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Result summarizes a Send across a topic's subscribers.
type Result struct {
	Sent    int `json:"sent"`
	Failed  int `json:"failed"`
	Removed int `json:"removed"`
}

type Service struct {
	repo       Repository
	publicKey  string
	privateKey string
	subject    string
	client     *http.Client
	logger     *slog.Logger
}

// NewService returns a Service that signs pushes with the given VAPID keys.
// subject is a contact e-mail address or https URL for push services. If
// either key is empty, push is disabled and Subscribe/Send return
// ErrDisabled.
func NewService(repo Repository, publicKey, privateKey, subject string, logger *slog.Logger) *Service {
	return &Service{
		repo:       repo,
		publicKey:  publicKey,
		privateKey: privateKey,
		// webpush-go adds "mailto:" to non-https subjects itself, so a
		// subject already written as "mailto:..." would be sent as
		// "mailto:mailto:...", which Apple's push service rejects.
		subject: strings.TrimPrefix(strings.TrimSpace(subject), "mailto:"),
		client:  &http.Client{Timeout: sendTimeout},
		logger:  logger,
	}
}

func (s *Service) Enabled() bool {
	return s.publicKey != "" && s.privateKey != ""
}

// PublicKey is the VAPID application server key browsers subscribe with.
func (s *Service) PublicKey() string {
	return s.publicKey
}

// Subscribe registers a browser under sub's user ID and topic.
func (s *Service) Subscribe(ctx context.Context, sub Subscription) error {
	if !s.Enabled() {
		return ErrDisabled
	}
	userID, topic, err := normalizeTopic(sub.UserID, sub.Topic)
	if err != nil {
		return err
	}
	sub.UserID, sub.Topic = userID, topic
	if err := validateEndpoint(sub.Endpoint); err != nil {
		return err
	}
	if sub.P256dh == "" || sub.Auth == "" || len(sub.P256dh) > 200 || len(sub.Auth) > 100 {
		return fmt.Errorf("%w: invalid subscription keys", ErrValidation)
	}
	return s.repo.Save(ctx, sub)
}

// Unsubscribe removes one user ID + topic registration for a browser.
func (s *Service) Unsubscribe(ctx context.Context, endpoint, userID, topic string) error {
	if endpoint == "" {
		return fmt.Errorf("%w: endpoint is required", ErrValidation)
	}
	userID, topic, err := normalizeTopic(userID, topic)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, endpoint, userID, topic)
}

// Topics returns the registrations for a browser.
func (s *Service) Topics(ctx context.Context, endpoint string) ([]Subscription, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("%w: endpoint is required", ErrValidation)
	}
	return s.repo.ListByEndpoint(ctx, endpoint)
}

// Send pushes msg to every browser registered under userID + topic.
// Browsers the push service reports as gone (404/410) are deleted.
func (s *Service) Send(ctx context.Context, userID, topic string, msg Message) (Result, error) {
	if !s.Enabled() {
		return Result{}, ErrDisabled
	}

	userID, topic, err := normalizeTopic(userID, topic)
	if err != nil {
		return Result{}, err
	}

	msg.Title = strings.TrimSpace(msg.Title)
	msg.Body = strings.TrimSpace(msg.Body)
	if msg.Title == "" {
		return Result{}, fmt.Errorf("%w: title is required", ErrValidation)
	}
	if len(msg.Title) > 200 || len(msg.Body) > 1000 {
		return Result{}, fmt.Errorf("%w: message is too long", ErrValidation)
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return Result{}, fmt.Errorf("encode push message: %w", err)
	}

	subs, err := s.repo.ListByTopic(ctx, userID, topic)
	if err != nil {
		return Result{}, err
	}
	if len(subs) == 0 {
		return Result{}, ErrNoSubscribers
	}

	var (
		mu     sync.Mutex
		result Result
		wg     sync.WaitGroup
		sem    = make(chan struct{}, maxSendConcurrency)
	)
	for _, sub := range subs {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			sent, gone := s.sendOne(ctx, sub, payload)
			if gone {
				if err := s.repo.DeleteEndpoint(ctx, sub.Endpoint); err != nil {
					s.logger.Error("delete expired push subscription", "error", err)
				}
			}

			mu.Lock()
			defer mu.Unlock()
			switch {
			case sent:
				result.Sent++
			case gone:
				result.Removed++
			default:
				result.Failed++
			}
		}()
	}
	wg.Wait()

	return result, nil
}

// sendOne delivers payload to sub, reporting whether it was accepted and
// whether the subscription no longer exists.
func (s *Service) sendOne(ctx context.Context, sub Subscription, payload []byte) (sent, gone bool) {
	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
	}, &webpush.Options{
		HTTPClient:      s.client,
		Subscriber:      s.subject,
		VAPIDPublicKey:  s.publicKey,
		VAPIDPrivateKey: s.privateKey,
		TTL:             24 * 60 * 60,
	})
	if err != nil {
		s.logger.Warn("send push notification", "host", endpointHost(sub.Endpoint), "error", err)
		return false, false
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return true, false
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return false, true
	default:
		// Push services explain rejections in the body, e.g. Apple's
		// {"reason":"BadJwtToken"}; it never contains subscriber data.
		reason, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		s.logger.Warn("push service rejected notification",
			"host", endpointHost(sub.Endpoint),
			"status", resp.StatusCode,
			"reason", strings.TrimSpace(string(reason)),
			"subject", s.subject,
		)
		return false, false
	}
}

// normalizeTopic validates userID and topic, returning userID in canonical
// lowercase UUID form so differently-cased input matches the same user.
func normalizeTopic(userID, topic string) (string, string, error) {
	id, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return "", "", fmt.Errorf("%w: user ID must be a UUID", ErrValidation)
	}
	topic = strings.TrimSpace(topic)
	if !topicPattern.MatchString(topic) {
		return "", "", fmt.Errorf("%w: topic must be 20-64 letters, digits, - or _", ErrValidation)
	}
	return id.String(), topic, nil
}

func validateEndpoint(endpoint string) error {
	if len(endpoint) > 2048 {
		return fmt.Errorf("%w: endpoint is too long", ErrValidation)
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" {
		return fmt.Errorf("%w: endpoint must be an https URL", ErrValidation)
	}
	host := u.Hostname()
	for _, allowed := range allowedPushHosts {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return nil
		}
	}
	return fmt.Errorf("%w: unsupported push service %q", ErrValidation, host)
}

func endpointHost(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
