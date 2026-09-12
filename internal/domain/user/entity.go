// Package user contains the pure business entity and rules for the User
// aggregate. It does not know about Fiber, HTTP or MongoDB.
package user

import "time"

// User is the core domain entity representing a registered user.
type User struct {
	ID        string
	Name      string
	Email     string
	Password  string
	CreatedAt time.Time
}
