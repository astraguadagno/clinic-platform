package directory

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrUnauthorized = errors.New("directory unauthorized")

type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	ProfessionalID *string   `json:"professional_id,omitempty"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type BootstrapAccessParams struct {
	AdminEmail     string
	AdminPassword  string
	DoctorEmail    string
	DoctorPassword string
}

const bootstrapDoctorProfessionalID = "8d933ba7-5fae-4a20-9f2c-2f5589f9f522"

func (r *Repository) BootstrapAccess(ctx context.Context, params BootstrapAccessParams) error {
	email := normalizeEmail(params.AdminEmail)
	password := strings.TrimSpace(params.AdminPassword)
	doctorEmail := normalizeEmail(params.DoctorEmail)
	doctorPassword := strings.TrimSpace(params.DoctorPassword)
	if email == "" || password == "" || doctorEmail == "" || doctorPassword == "" {
		return ErrValidation
	}

	adminPasswordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	doctorPasswordHash, err := bcrypt.GenerateFromPassword([]byte(doctorPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	roles := []struct {
		Name        string
		Description string
	}{
		{Name: "admin", Description: "Platform administrator"},
		{Name: "doctor", Description: "Healthcare professional user"},
	}

	for _, role := range roles {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO roles (name, description)
			VALUES ($1, $2)
			ON CONFLICT (name) DO NOTHING
		`, role.Name, role.Description); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO professionals (id, first_name, last_name, specialty)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, bootstrapDoctorProfessionalID, "Elena", "Martinez", "general medicine"); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, 'admin')
		ON CONFLICT (email) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    role = EXCLUDED.role,
		    professional_id = NULL,
		    active = TRUE,
		    updated_at = NOW()
	`, email, string(adminPasswordHash)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, role, professional_id)
		VALUES ($1, $2, 'doctor', $3)
		ON CONFLICT (email) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    role = EXCLUDED.role,
		    professional_id = EXCLUDED.professional_id,
		    active = TRUE,
		    updated_at = NOW()
	`, doctorEmail, string(doctorPasswordHash), bootstrapDoctorProfessionalID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) AuthenticateUser(ctx context.Context, email, password string) (User, error) {
	normalizedEmail, normalizedPassword, err := validateLoginParams(email, password)
	if err != nil {
		return User{}, err
	}

	var user User
	var passwordHash string
	var professionalID sql.NullString

	err = r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, role, professional_id, active, created_at, updated_at
		FROM users
		WHERE email = $1
	`, normalizedEmail).Scan(
		&user.ID,
		&user.Email,
		&passwordHash,
		&user.Role,
		&professionalID,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUnauthorized
	}
	if err != nil {
		return User{}, err
	}
	if !user.Active {
		return User{}, ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(normalizedPassword)); err != nil {
		return User{}, ErrUnauthorized
	}
	if professionalID.Valid {
		user.ProfessionalID = &professionalID.String
	}

	return user, nil
}

func (r *Repository) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(tokenHash) == "" || expiresAt.IsZero() {
		return ErrValidation
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO auth_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt.UTC())

	return err
}

func (r *Repository) GetUserBySessionToken(ctx context.Context, tokenHash string, now time.Time) (User, error) {
	if strings.TrimSpace(tokenHash) == "" {
		return User{}, ErrUnauthorized
	}

	var user User
	var professionalID sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.role, u.professional_id, u.active, u.created_at, u.updated_at
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.expires_at > $2
	`, tokenHash, now.UTC()).Scan(
		&user.ID,
		&user.Email,
		&user.Role,
		&professionalID,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUnauthorized
	}
	if err != nil {
		return User{}, err
	}
	if !user.Active {
		return User{}, ErrUnauthorized
	}
	if professionalID.Valid {
		user.ProfessionalID = &professionalID.String
	}

	return user, nil
}

func validateLoginParams(email, password string) (string, string, error) {
	normalizedEmail := normalizeEmail(email)
	normalizedPassword := strings.TrimSpace(password)
	if normalizedEmail == "" || normalizedPassword == "" {
		return "", "", ErrValidation
	}

	return normalizedEmail, normalizedPassword, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
