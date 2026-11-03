package main

import (
	"errors"
	"fmt"
)

// ============================================================================
// 1. IMPLICIT INTERFACES (Structural Typing)
// ============================================================================
// Unlike Java/C#, structs do not declare `implements InterfaceName`.
// Any type whose method set includes these signatures satisfies Notifier.
type Notifier interface {
	SendNotification(message string) error
	GetTargetAddress() string
}

// ============================================================================
// 2. STRUCT COMPOSITION (Embedding over Inheritance)
// ============================================================================
// Go has no class hierarchy. Embedding reuses fields (and methods) by promotion.
type AuditMetadata struct {
	CreatedBy string
	Version   int
}

type User struct {
	ID    string
	Email string
	AuditMetadata
}

// If any method mutates the receiver, give ALL methods pointer receivers.
// Method set of *User includes these; method set of User does not.
// Consequence: User does not satisfy Notifier; *User does.
func (u *User) GetTargetAddress() string {
	return u.Email
}

func (u *User) UpdateEmail(newEmail string) error {
	if newEmail == "" {
		return errors.New("email cannot be empty")
	}
	u.Email = newEmail
	u.Version++ // promoted field from AuditMetadata
	return nil
}

func (u *User) SendNotification(msg string) error {
	fmt.Printf("[Go Email Service] Sending to %s (v%d): '%s'\n", u.Email, u.Version, msg)
	return nil
}

// ============================================================================
// 3. ANOTHER TYPE SATISFYING THE SAME INTERFACE
// ============================================================================
type SlackBot struct {
	WebhookURL string
	Channel    string
}

// Value receivers are appropriate when the type is never mutated.
// Then both SlackBot and *SlackBot satisfy Notifier.
func (s SlackBot) GetTargetAddress() string {
	return s.Channel
}

func (s SlackBot) SendNotification(msg string) error {
	fmt.Printf("[Go Slack Service] Posting to %s: '%s'\n", s.Channel, msg)
	return nil
}

func main() {
	fmt.Println("=== Senior Polyglot Course: Day 1 (Go) ===")

	fmt.Println("\n--- Zero values are valid (empty) states ---")
	var empty User
	fmt.Printf("Zero User: ID=%q Email=%q CreatedBy=%q Version=%d\n",
		empty.ID, empty.Email, empty.CreatedBy, empty.Version)

	user := User{
		ID:    "usr-8832",
		Email: "alex.senior@enterprise.io",
		AuditMetadata: AuditMetadata{
			CreatedBy: "system-init",
			Version:   1,
		},
	}

	bot := SlackBot{
		WebhookURL: "https://hooks.slack.com/services/xxx",
		Channel:    "#backend-alerts",
	}

	// []Notifier{user, bot} does not compile: User is not a Notifier.
	channels := []Notifier{&user, bot}

	fmt.Println("\n--- Dispatching via implicit interfaces ---")
	for _, ch := range channels {
		_ = ch.SendNotification("High CPU utilization detected on node-04.")
	}

	fmt.Println("\n--- Value copy vs pointer mutation ---")
	copied := user // independent copy; later mutations of user do not affect it
	fmt.Printf("Before update:  user=%s v%d | copied=%s v%d\n",
		user.Email, user.Version, copied.Email, copied.Version)

	if err := user.UpdateEmail("alex.lead@enterprise.io"); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Printf("After update:   user=%s v%d | copied=%s v%d\n",
		user.Email, user.Version, copied.Email, copied.Version)

	fmt.Println("\n--- Interface holds *User, so dispatch sees the mutation ---")
	_ = channels[0].SendNotification("Follow-up after email change.")
}
