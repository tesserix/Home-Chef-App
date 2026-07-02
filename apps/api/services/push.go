package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"golang.org/x/oauth2/google"
)

// PushService handles sending push notifications via FCM HTTP v1 API
type PushService struct {
	projectID string
	available bool
	mu        sync.RWMutex
}

var (
	pushService *PushService
	pushOnce    sync.Once
)

// GetPushService returns the singleton push service
func GetPushService() *PushService {
	pushOnce.Do(func() {
		pushService = &PushService{}
	})
	return pushService
}

// InitPushService initialises the push notification service.
// It relies on Google Application Default Credentials (Workload Identity on GKE).
func InitPushService() error {
	svc := GetPushService()
	svc.projectID = config.AppConfig.GCSProjectID

	// Validate that default credentials are available
	_, err := google.FindDefaultCredentials(nil, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		log.Printf("Warning: FCM push notifications unavailable — no default credentials: %v", err)
		svc.available = false
		return nil // non-fatal
	}

	svc.available = true
	log.Printf("Push notification service initialised (FCM project=%s)", svc.projectID)
	return nil
}

// fcmMessage is the FCM HTTP v1 request envelope
type fcmMessage struct {
	Message fcmMessageBody `json:"message"`
}

type fcmMessageBody struct {
	Token        string            `json:"token"`
	Notification *fcmNotification  `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Android      *fcmAndroid       `json:"android,omitempty"`
	APNS         *fcmAPNS          `json:"apns,omitempty"`
}

// fcmAndroid carries Android-specific message options for actionable notifications.
type fcmAndroid struct {
	Priority     string           `json:"priority,omitempty"` // "high"
	Notification *fcmAndroidNotif `json:"notification,omitempty"`
}

type fcmAndroidNotif struct {
	ChannelID string `json:"channel_id,omitempty"`
	Sound     string `json:"sound,omitempty"`
	Priority  string `json:"notification_priority,omitempty"` // "PRIORITY_MAX"
}

// fcmAPNS carries Apple Push Notification Service options for actionable notifications.
type fcmAPNS struct {
	Payload *fcmAPNSPayload   `json:"payload,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type fcmAPNSPayload struct {
	APS *fcmAPS `json:"aps,omitempty"`
}

type fcmAPS struct {
	Category         string `json:"category,omitempty"`          // iOS notification category identifier
	Sound            string `json:"sound,omitempty"`             // "default"
	ContentAvailable int    `json:"content-available,omitempty"` // 1 for background updates
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// getAccessToken obtains a short-lived OAuth2 token using default credentials
func (s *PushService) getAccessToken() (string, error) {
	creds, err := google.FindDefaultCredentials(nil, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return "", fmt.Errorf("push: failed to find credentials: %w", err)
	}
	tok, err := creds.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("push: failed to get token: %w", err)
	}
	return tok.AccessToken, nil
}

// sendFCMMessage sends a fully-constructed fcmMessageBody to FCM HTTP v1 API.
// Both sendToToken and SendActionablePush use this shared path.
func (s *PushService) sendFCMMessage(body *fcmMessageBody) error {
	s.mu.RLock()
	available := s.available
	s.mu.RUnlock()

	if !available {
		log.Printf("Push skipped (not available): title=%s", body.Notification.Title)
		return nil
	}

	accessToken, err := s.getAccessToken()
	if err != nil {
		return err
	}

	msg := fcmMessage{Message: *body}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("push: failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", s.projectID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("push: failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("push: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push: FCM returned status %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("Push notification sent: token=%s...%s title=%s", body.Token[:6], body.Token[len(body.Token)-4:], body.Notification.Title)
	return nil
}

// sendToToken sends a push notification to a single FCM device token
func (s *PushService) sendToToken(token, title, body string, data map[string]string) error {
	return s.sendFCMMessage(&fcmMessageBody{
		Token:        token,
		Notification: &fcmNotification{Title: title, Body: body},
		Data:         data,
	})
}

// SendPushNotification sends a push notification to a single user by looking up their FCM token
func SendPushNotification(userID uuid.UUID, title, body string, data map[string]string) error {
	var user models.User
	if err := database.DB.Select("id, fcm_token").First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("push: user %s not found: %w", userID, err)
	}

	if user.FCMToken == "" {
		log.Printf("Push skipped: user %s has no FCM token", userID)
		return nil
	}

	return GetPushService().sendToToken(user.FCMToken, title, body, data)
}

// SendActionablePush sends a push notification with platform-specific action metadata.
// Use for vendor new-order notifications that need lock-screen Accept/Reject buttons.
// androidChannelID: e.g. "new-orders"; iosCategory: e.g. "new_order"
func SendActionablePush(userID uuid.UUID, title, body, androidChannelID, iosCategory string, data map[string]string) error {
	var user models.User
	if err := database.DB.Select("id, fcm_token").First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("push: user %s not found: %w", userID, err)
	}
	if user.FCMToken == "" {
		log.Printf("Push skipped: user %s has no FCM token", userID)
		return nil
	}
	return GetPushService().sendFCMMessage(&fcmMessageBody{
		Token:        user.FCMToken,
		Notification: &fcmNotification{Title: title, Body: body},
		Data:         data,
		Android: &fcmAndroid{
			Priority: "high",
			Notification: &fcmAndroidNotif{
				ChannelID: androidChannelID,
				Sound:     "default",
				Priority:  "PRIORITY_MAX",
			},
		},
		APNS: &fcmAPNS{
			Payload: &fcmAPNSPayload{
				APS: &fcmAPS{
					Category: iosCategory,
					Sound:    "default",
				},
			},
		},
	})
}

// SubscribeToFCMTopic adds the device token to a topic so it receives
// topic broadcasts. Used by the notification-preferences handler when
// a chef opts INTO a category. No-op when the push service isn't
// configured — local dev without GCP creds shouldn't fail on FCM ops.
//
// The Firebase Instance ID (IID) API takes batches of up to 1000 tokens
// per call; for single-token mobile-driven flows we just send one.
func SubscribeToFCMTopic(token, topic string) error {
	return iidTopicCall("POST", token, topic)
}

// UnsubscribeFromFCMTopic is SubscribeToFCMTopic's pair — used when a
// chef opts OUT of a category. Errors are surfaced to the caller so
// they can decide whether to log + move on (the prefs handler does).
func UnsubscribeFromFCMTopic(token, topic string) error {
	return iidTopicCall("DELETE", token, topic)
}

// iidTopicCall hits the Firebase Instance ID batchAdd / batchRemove
// endpoint with a one-token batch. Returns nil on 2xx or when the
// push service is disabled; surfaces transport errors and non-2xx
// status otherwise so the caller can record metrics.
func iidTopicCall(method, token, topic string) error {
	svc := GetPushService()
	svc.mu.RLock()
	available := svc.available
	svc.mu.RUnlock()
	if !available {
		return nil
	}
	if token == "" || topic == "" {
		return fmt.Errorf("fcm topic call: token and topic required")
	}

	accessToken, err := svc.getAccessToken()
	if err != nil {
		return fmt.Errorf("fcm topic call: access token: %w", err)
	}

	// Escape the token before splicing it into the path — it's caller-supplied
	// and could otherwise inject extra path segments into the IID URL.
	endpoint := "https://iid.googleapis.com/iid/v1/" + url.PathEscape(token) + "/rel/topics/" + topic
	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return fmt.Errorf("fcm topic call: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token_auth", "true")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fcm topic call: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("fcm topic call: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// SendPushToMultiple sends a push notification to multiple users
func SendPushToMultiple(userIDs []uuid.UUID, title, body string, data map[string]string) error {
	var users []models.User
	if err := database.DB.Select("id, fcm_token").Where("id IN ? AND fcm_token != ''", userIDs).Find(&users).Error; err != nil {
		return fmt.Errorf("push: failed to query users: %w", err)
	}

	svc := GetPushService()
	var lastErr error
	for _, u := range users {
		if u.FCMToken == "" {
			continue
		}
		if err := svc.sendToToken(u.FCMToken, title, body, data); err != nil {
			log.Printf("Push failed for user %s: %v", u.ID, err)
			lastErr = err
		}
	}
	return lastErr
}
